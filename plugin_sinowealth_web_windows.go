//go:build windows

package main

import (
	"fmt"
	"io"
	"net/http"
)

type sinoWealthAPIReq struct {
	Op         string                   `json:"op"`
	Preset     string                   `json:"preset,omitempty"`
	ColorOrder string                   `json:"colorOrder,omitempty"`
	Buttons    []SinoWealthButtonAction `json:"buttons,omitempty"`
	Config     *SinoWealthConfigPatch   `json:"config,omitempty"`
	Link       *SinoWealthHardwareLink  `json:"link,omitempty"`
}

func (p *SinoWealthPlugin) buildWebPayload(a *App, device *SinoWealthConfigState) map[string]any {
	enabled := a.isPluginEnabled(p.ID(), p.DefaultEnabled())
	p.stateMu.RLock()
	st := device
	if st == nil {
		st = p.lastState
	}
	backupPath := p.backupPath
	hasBackup := p.hasBackup
	lastErr := p.lastError
	lastReadAt := ""
	if !p.lastReadAt.IsZero() {
		lastReadAt = p.lastReadAt.Format("15:04:05")
	}
	lastSyncMsg := p.lastSyncMsg
	p.stateMu.RUnlock()

	var profileLink SinoWealthHardwareLink
	profileName := ""
	a.mu.RLock()
	if a.editorProfileIndex >= 0 && a.editorProfileIndex < len(a.config.Profiles) {
		prof := a.config.Profiles[a.editorProfileIndex]
		profileLink = prof.SinoWealth
		profileName = prof.Name
	}
	a.mu.RUnlock()

	return map[string]any{
		"ok":                true,
		"enabled":           enabled,
		"detected":          st != nil,
		"device":            st,
		"backupPath":        backupPath,
		"hasBackup":         hasBackup,
		"lastReadAt":        lastReadAt,
		"lastError":         lastErr,
		"lastSyncMessage":   lastSyncMsg,
		"editorProfileName": profileName,
		"profileLink":       profileLink,
		"colorOrder":        p.colorOrder(a),
		"hidKeys":           sinoWealthHIDKeys,
	}
}

func (p *SinoWealthPlugin) handleAPI(w http.ResponseWriter, r *http.Request, a *App) {
	if r.Method == http.MethodGet {
		if a.isPluginEnabled(p.ID(), p.DefaultEnabled()) {
			st, _, err := p.ReadDevice(a)
			if err == nil {
				writeJSON(w, p.buildWebPayload(a, &st))
				return
			}
		}
		writeJSON(w, p.buildWebPayload(a, nil))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req sinoWealthAPIReq
	if !decodeJSON(w, r, &req) {
		return
	}

	switch req.Op {
	case "read":
		st, _, err := p.ReadDevice(a)
		if err != nil {
			writeError(w, err)
			return
		}
		resp := p.buildWebPayload(a, &st)
		resp["message"] = fmt.Sprintf("%s (FW %s) の設定を読み込みました。", st.ProductName, st.Firmware)
		writeJSON(w, resp)

	case "apply-preset":
		if !a.isPluginEnabled(p.ID(), p.DefaultEnabled()) {
			writeError(w, fmt.Errorf("先に SinoWealth プラグインを有効にしてください。"))
			return
		}
		btns, err := sinoWealthPresetButtons(req.Preset)
		if err != nil {
			writeError(w, err)
			return
		}
		st, err := p.WriteButtons(a, btns)
		if err != nil {
			writeError(w, err)
			return
		}
		resp := p.buildWebPayload(a, &st)
		resp["message"] = "オンボードボタン設定を書き込み、検証を完了しました。"
		writeJSON(w, resp)

	case "write-buttons":
		if !a.isPluginEnabled(p.ID(), p.DefaultEnabled()) {
			writeError(w, fmt.Errorf("先に SinoWealth プラグインを有効にしてください。"))
			return
		}
		if len(req.Buttons) == 0 {
			writeError(w, fmt.Errorf("書き込むボタン設定が空です。"))
			return
		}
		st, err := p.WriteButtons(a, req.Buttons)
		if err != nil {
			writeError(w, err)
			return
		}
		resp := p.buildWebPayload(a, &st)
		resp["message"] = "オンボードのボタン割り当てをマウスへ書き込み、検証しました。"
		writeJSON(w, resp)

	case "write-config":
		if !a.isPluginEnabled(p.ID(), p.DefaultEnabled()) {
			writeError(w, fmt.Errorf("先に SinoWealth プラグインを有効にしてください。"))
			return
		}
		if req.Config == nil {
			writeError(w, fmt.Errorf("書き込む設定データがありません。"))
			return
		}
		st, err := p.WriteConfigPatch(a, *req.Config)
		if err != nil {
			writeError(w, err)
			return
		}
		resp := p.buildWebPayload(a, &st)
		resp["message"] = fmt.Sprintf("マウス本体設定を書き込みました（%dHz / アクティブ DPI #%d）。", st.ReportRateHz, st.ActiveDPIIndex+1)
		writeJSON(w, resp)

	case "restore-backup":
		st, err := p.RestoreInitialBackup(a)
		if err != nil {
			writeError(w, err)
			return
		}
		resp := p.buildWebPayload(a, &st)
		resp["message"] = "初回バックアップからマウス本体の設定とボタンを復元しました。"
		writeJSON(w, resp)

	case "save-color-order":
		order := normalizeColorOrder(req.ColorOrder)
		a.mu.Lock()
		a.config.Plugins.SinoWealthColorOrder = order
		err := a.saveConfigLocked()
		a.mu.Unlock()
		if err != nil {
			writeError(w, err)
			return
		}
		st, _, err := p.ReadDevice(a)
		if err != nil {
			resp := p.buildWebPayload(a, nil)
			resp["message"] = fmt.Sprintf("LED色順を %s に保存しました。", order)
			writeJSON(w, resp)
			return
		}
		resp := p.buildWebPayload(a, &st)
		resp["message"] = fmt.Sprintf("LED色順を %s に変更しました（青が緑に見える場合は RBG を選んで再適用してください）。", order)
		writeJSON(w, resp)

	case "save-profile-link":
		if req.Link == nil {
			writeError(w, fmt.Errorf("連動設定データがありません。"))
			return
		}
		link := *req.Link
		if link.ReportRateHz != 0 {
			if _, err := sinoWealthRateToRaw(link.ReportRateHz); err != nil {
				writeError(w, err)
				return
			}
		}
		if link.ActiveDPISlot < 0 || link.ActiveDPISlot > sinoWealthDPISlots {
			writeError(w, fmt.Errorf("連動DPIスロットは0(変更しない)〜%dで指定してください。", sinoWealthDPISlots))
			return
		}
		if link.LEDMode < 0 || link.LEDMode > 3 {
			writeError(w, fmt.Errorf("連動LEDモードが不正です: %d", link.LEDMode))
			return
		}
		if link.LEDMode == 1 {
			if _, _, _, err := parseHexRGB(link.LEDColor); err != nil {
				writeError(w, fmt.Errorf("モード連動LED色が不正です: %w", err))
				return
			}
		}
		a.mu.Lock()
		if a.editorProfileIndex < 0 || a.editorProfileIndex >= len(a.config.Profiles) {
			a.mu.Unlock()
			writeError(w, fmt.Errorf("編集中のプロファイルがありません。"))
			return
		}
		a.config.Profiles[a.editorProfileIndex].SinoWealth = link
		prof := a.config.Profiles[a.editorProfileIndex]
		isEffective := a.editorProfileIndex == a.activeProfileIndex
		err := a.saveConfigLocked()
		a.mu.Unlock()
		if err != nil {
			writeError(w, err)
			return
		}
		if isEffective {
			p.OnEffectiveProfileChanged(a, prof)
		}
		resp := p.buildWebPayload(a, nil)
		resp["message"] = fmt.Sprintf("プロファイル「%s」のマウス本体連動設定（DPI・レート・色）を保存しました。", prof.Name)
		resp["state"] = a.buildWebState()
		writeJSON(w, resp)

	default:
		writeError(w, fmt.Errorf("未知の SinoWealth プラグイン操作です: %s", req.Op))
	}
}

func (p *SinoWealthPlugin) handleUIJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, sinoWealthPluginUIJS)
}

const sinoWealthPluginUIJS = `(function(){
'use strict';
const $ = id => document.getElementById(id);
const esc = s => String(s == null ? '' : s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
let swData = null;
let swBusy = false;
let swButtonsDraft = [];
let swDPIDraft = [];
let swDirtyButtons = false;
let swDirtyConfig = false;
let swDirtyLink = false;
let swLastProfileName = '';

function notify(msg, isErr){
  const el = $('message');
  if(!el) return;
  el.textContent = msg || '';
  el.classList.toggle('error', !!isErr);
}

async function swFetch(method, body){
  swBusy = true;
  setButtonsDisabled(true);
  try{
    const opts = {method, cache:'no-store'};
    if(method === 'POST'){
      opts.headers = {'Content-Type':'application/json'};
      opts.body = JSON.stringify(body || {});
    }
    const res = await fetch('/api/plugins/sinowealth', opts);
    const j = await res.json().catch(()=>({}));
    if(!res.ok || j.ok === false) throw new Error(j.error || ('HTTP ' + res.status));
    swData = j;
    if(j.message) notify(j.message, false);
    render();
    return j;
  } catch(e){
    notify('マウス設定プラグイン: ' + e.message, true);
    throw e;
  } finally {
    swBusy = false;
    setButtonsDisabled(false);
  }
}

function setButtonsDisabled(disabled){
  const root = $('sinoWealthPluginHost');
  if(!root) return;
  root.querySelectorAll('button[data-sw-act]').forEach(b => { b.disabled = disabled; });
}

function keyOptionsHTML(selectedUsage){
  const keys = (swData && swData.hidKeys) || [];
  const groups = {};
  keys.forEach(k => {
    if(!groups[k.group]) groups[k.group] = [];
    groups[k.group].push(k);
  });
  let html = '';
  Object.keys(groups).forEach(g => {
    html += '<optgroup label="' + esc(g) + '">';
    groups[g].forEach(k => {
      html += '<option value="' + k.usage + '"' + (Number(k.usage) === Number(selectedUsage) ? ' selected' : '') + '>' + esc(k.name) + ' (0x' + Number(k.usage).toString(16).toUpperCase().padStart(2,'0') + ')</option>';
    });
    html += '</optgroup>';
  });
  return html;
}

function buildValueControl(btn, idx){
  if(btn.kind === 'mouse'){
    const opts = [['left','左クリック'],['right','右クリック'],['middle','中クリック'],['x1','戻る (X1)'],['x2','進む (X2)']];
    return '<select data-sw-btn-val="' + idx + '">' + opts.map(o => '<option value="' + o[0] + '"' + (btn.value===o[0]?' selected':'') + '>' + o[1] + '</option>').join('') + '</select>';
  }
  if(btn.kind === 'wheel'){
    return '<select data-sw-btn-val="' + idx + '"><option value="up"' + (btn.value==='up'?' selected':'') + '>ホイール上</option><option value="down"' + (btn.value==='down'?' selected':'') + '>ホイール下</option></select>';
  }
  if(btn.kind === 'dpi'){
    const opts = [['up','DPI +'],['down','DPI -'],['cycle','DPI 循環']];
    return '<select data-sw-btn-val="' + idx + '">' + opts.map(o => '<option value="' + o[0] + '"' + (btn.value===o[0]?' selected':'') + '>' + o[1] + '</option>').join('') + '</select>';
  }
  if(btn.kind === 'media'){
    const opts = [['play_pause','再生/一時停止'],['mute','ミュート'],['vol_up','音量アップ'],['vol_down','音量ダウン'],['next','次の曲'],['prev','前の曲'],['stop','停止']];
    return '<select data-sw-btn-val="' + idx + '">' + opts.map(o => '<option value="' + o[0] + '"' + (btn.value===o[0]?' selected':'') + '>' + o[1] + '</option>').join('') + '</select>';
  }
  if(btn.kind === 'special'){
    const opts = [['disabled','無効'],['rate_cycle','ポーリングレート切替'],['led_cycle','LEDモード切替'],['profile_cycle','プロファイル切替']];
    return '<select data-sw-btn-val="' + idx + '">' + opts.map(o => '<option value="' + o[0] + '"' + (btn.value===o[0]?' selected':'') + '>' + o[1] + '</option>').join('') + '</select>';
  }
  if(btn.kind === 'key'){
    const mod = Number(btn.modifiers || 0);
    return '<div class="sw-key-wrap">' +
      '<select data-sw-btn-key="' + idx + '">' + keyOptionsHTML(btn.hidUsage || 0x68) + '</select>' +
      '<label class="sw-mod"><input type="checkbox" data-sw-mod="' + idx + '" data-bit="1"' + ((mod&1)?' checked':'') + '>Ctrl</label>' +
      '<label class="sw-mod"><input type="checkbox" data-sw-mod="' + idx + '" data-bit="2"' + ((mod&2)?' checked':'') + '>Shift</label>' +
      '<label class="sw-mod"><input type="checkbox" data-sw-mod="' + idx + '" data-bit="4"' + ((mod&4)?' checked':'') + '>Alt</label>' +
      '<label class="sw-mod"><input type="checkbox" data-sw-mod="' + idx + '" data-bit="8"' + ((mod&8)?' checked':'') + '>Win</label>' +
      '</div>';
  }
  return '<span class="muted">' + esc(btn.display || btn.rawHex || '―') + '</span>';
}

function renderButtonsTable(){
  const tbody = $('swBtnRows');
  if(!tbody) return;
  tbody.innerHTML = '';
  const kinds = [
    ['mouse','マウスボタン'],
    ['key','キーボード (F13〜F24等)'],
    ['dpi','DPI操作'],
    ['media','メディアキー'],
    ['wheel','ホイール'],
    ['special','特殊 / 無効']
  ];
  swButtonsDraft.forEach((b, idx) => {
    const tr = document.createElement('tr');
    const kindKnown = kinds.some(k => k[0] === b.kind);
    const kindOpts = kinds.map(k => '<option value="' + k[0] + '"' + (b.kind===k[0]?' selected':'') + '>' + k[1] + '</option>').join('') +
      (!kindKnown ? '<option value="' + esc(b.kind) + '" selected>' + esc(b.kind) + ' (保持)</option>' : '');
    tr.innerHTML =
      '<td class="num">' + (idx + 1) + '</td>' +
      '<td><b>' + esc(b.label) + '</b></td>' +
      '<td><select data-sw-btn-kind="' + idx + '">' + kindOpts + '</select></td>' +
      '<td>' + buildValueControl(b, idx) + '</td>' +
      '<td><code>' + esc(b.rawHex || '') + '</code></td>';
    tbody.appendChild(tr);
  });

  tbody.querySelectorAll('select[data-sw-btn-kind]').forEach(sel => {
    sel.addEventListener('change', () => {
      const i = Number(sel.dataset.swBtnKind);
      const k = sel.value;
      swButtonsDraft[i].kind = k;
      if(k === 'mouse') swButtonsDraft[i].value = 'left';
      else if(k === 'wheel') swButtonsDraft[i].value = 'up';
      else if(k === 'dpi') swButtonsDraft[i].value = 'up';
      else if(k === 'media') swButtonsDraft[i].value = 'play_pause';
      else if(k === 'special') swButtonsDraft[i].value = 'disabled';
      else if(k === 'key') { swButtonsDraft[i].hidUsage = 0x68 + Math.min(i, 11); swButtonsDraft[i].modifiers = 0; swButtonsDraft[i].value = ''; }
      swDirtyButtons = true;
      renderButtonsTable();
    });
  });
  tbody.querySelectorAll('select[data-sw-btn-val]').forEach(sel => {
    sel.addEventListener('change', () => {
      const i = Number(sel.dataset.swBtnVal);
      swButtonsDraft[i].value = sel.value;
      swDirtyButtons = true;
    });
  });
  tbody.querySelectorAll('select[data-sw-btn-key]').forEach(sel => {
    sel.addEventListener('change', () => {
      const i = Number(sel.dataset.swBtnKey);
      swButtonsDraft[i].hidUsage = Number(sel.value);
      swButtonsDraft[i].value = '0x' + Number(sel.value).toString(16);
      swDirtyButtons = true;
    });
  });
  tbody.querySelectorAll('input[data-sw-mod]').forEach(cb => {
    cb.addEventListener('change', () => {
      const i = Number(cb.dataset.swMod);
      const bit = Number(cb.dataset.bit);
      let m = Number(swButtonsDraft[i].modifiers || 0);
      if(cb.checked) m |= bit; else m &= ~bit;
      swButtonsDraft[i].modifiers = m;
      swDirtyButtons = true;
    });
  });
}

function renderDPITable(dev){
  const tbody = $('swDPIRows');
  if(!tbody) return;
  tbody.innerHTML = '';
  const maxDpi = (dev && dev.maxDpi) || 12000;
  swDPIDraft.forEach((slot, idx) => {
    const tr = document.createElement('tr');
    const isActive = dev && Number($('swActiveDPI').value) === idx;
    tr.innerHTML =
      '<td class="num">#' + (idx + 1) + '</td>' +
      '<td class="check"><input type="checkbox" data-sw-dpi-en="' + idx + '"' + (slot.enabled ? ' checked' : '') + '></td>' +
      '<td><input type="number" data-sw-dpi-val="' + idx + '" min="100" max="' + maxDpi + '" step="100" value="' + (slot.dpi || 800) + '"' + (slot.enabled ? '' : ' disabled') + '> DPI</td>' +
      '<td><input type="color" data-sw-dpi-col="' + idx + '" value="' + esc(slot.color || '#FFFFFF') + '"></td>' +
      '<td>' + (isActive ? '<span class="pill ok">アクティブ</span>' : '<span class="muted">―</span>') + '</td>';
    tbody.appendChild(tr);
  });
  tbody.querySelectorAll('input[data-sw-dpi-en]').forEach(cb => {
    cb.addEventListener('change', () => {
      const i = Number(cb.dataset.swDpiEn);
      swDPIDraft[i].enabled = cb.checked;
      if(cb.checked && (!swDPIDraft[i].dpi || swDPIDraft[i].dpi < 100)) swDPIDraft[i].dpi = 800;
      swDirtyConfig = true;
      renderDPITable(swData && swData.device);
    });
  });
  tbody.querySelectorAll('input[data-sw-dpi-val]').forEach(inp => {
    inp.addEventListener('input', () => {
      swDPIDraft[Number(inp.dataset.swDpiVal)].dpi = Number(inp.value) || 800;
      swDirtyConfig = true;
    });
  });
  tbody.querySelectorAll('input[data-sw-dpi-col]').forEach(inp => {
    inp.addEventListener('input', () => {
      swDPIDraft[Number(inp.dataset.swDpiCol)].color = inp.value;
      swDirtyConfig = true;
    });
  });
}

function ensureHost(){
  const host = $('sinoWealthPluginHost');
  if(!host || host.dataset.ready) return host;
  host.dataset.ready = '1';
  host.innerHTML =
    '<div class="sw-panel">' +
      '<div class="sw-header">' +
        '<div>' +
          '<h3 class="sw-title">Ray Pawn+ / SinoWealth マウス設定</h3>' +
          '<p class="muted sw-sub" id="swDeviceSummary">マウス状態を確認しています…</p>' +
        '</div>' +
        '<div class="toolbar">' +
          '<button type="button" id="swReload" data-sw-act="1">実機から再読み込み</button>' +
          '<button type="button" id="swRestore" data-sw-act="1" class="danger">初期バックアップから復元</button>' +
        '</div>' +
      '</div>' +
      '<div id="swOfflineBox" class="callout warn hidden"><b>マウス未検出</b><div id="swOfflineDetail" class="muted">VID 258A / PID 1007 の SinoWealth マウスを接続して「実機から再読み込み」を押してください。</div></div>' +
      '<div id="swMainBody" class="hidden">' +
        '<div class="sw-stats">' +
          '<div class="metric"><div class="k">ファームウェア / センサー</div><div class="v" id="swStatFW">―</div></div>' +
          '<div class="metric"><div class="k">ポーリングレート</div><div class="v" id="swStatRate">―</div></div>' +
          '<div class="metric"><div class="k">現在の DPI</div><div class="v" id="swStatDPI">―</div></div>' +
          '<div class="metric"><div class="k">初期バックアップ</div><div class="v" id="swStatBackup">―</div></div>' +
        '</div>' +

        '<div class="sw-block">' +
          '<div class="sw-block-head"><div><b>MouseButtonMapper 連携ワンクリック最適化</b><div class="muted">Windows の隠しショートカット（Win+F16 シャットダウン等）と衝突しない F13 / F18 / F19 / F20 へ安全に書き換えます（センサー設定には触れません）。</div></div></div>' +
          '<div class="toolbar">' +
            '<button type="button" id="swPresetSide" data-sw-act="1" class="primary">右サイド2ボタンを F13 / F18 に最適化（推奨）</button>' +
            '<button type="button" id="swPresetAll" data-sw-act="1">右サイド(F13/F18) ＋ DPI±(F19/F20) 全9ボタン化</button>' +
            '<button type="button" id="swPresetFactory" data-sw-act="1">Ray Pawn+ 標準ボタン配置に戻す</button>' +
          '</div>' +
        '</div>' +

        '<div class="sw-block">' +
          '<div class="sw-block-head"><div><b>モード（プロファイル）ごとのマウス色・設定連動</b><div class="muted" id="swLinkProfileDesc">編集中のプロファイルへ切り替わった際、マウスのLED色・DPI・ポーリングレートを自動で切り替えます。</div></div></div>' +
          '<div class="row">' +
            '<label class="checkline"><input type="checkbox" id="swLinkEnabled"> このプロファイル適用時にマウス本体を連動切替する</label>' +
            '<div class="field"><label for="swLinkLEDMode">モード時のLED</label><select id="swLinkLEDMode"><option value="0">変更しない</option><option value="1">単色点灯（色を指定）</option><option value="2">消灯</option><option value="3">レインボー</option></select></div>' +
            '<div class="field" id="swLinkColorWrap"><label for="swLinkColor">モードの色</label><div class="row"><input type="color" id="swLinkColor" value="#0000FF"><div class="toolbar" aria-label="色プリセット">' +
              '<button type="button" class="small" data-sw-swatch="#0000FF">青</button>' +
              '<button type="button" class="small" data-sw-swatch="#00D8FF">シアン</button>' +
              '<button type="button" class="small" data-sw-swatch="#00FF44">緑</button>' +
              '<button type="button" class="small" data-sw-swatch="#FFD000">黄</button>' +
              '<button type="button" class="small" data-sw-swatch="#FF5500">橙</button>' +
              '<button type="button" class="small" data-sw-swatch="#FF0022">赤</button>' +
              '<button type="button" class="small" data-sw-swatch="#AA00FF">紫</button>' +
              '<button type="button" class="small" data-sw-swatch="#FFFFFF">白</button>' +
            '</div></div></div>' +
          '</div>' +
          '<div class="row" style="margin-top:8px">' +
            '<div class="field"><label for="swLinkRate">連動ポーリングレート</label><select id="swLinkRate"><option value="0">変更しない</option><option value="125">125 Hz</option><option value="250">250 Hz</option><option value="500">500 Hz</option><option value="1000">1000 Hz</option></select></div>' +
            '<div class="field"><label for="swLinkDPI">連動DPIスロット</label><select id="swLinkDPI"><option value="0">変更しない</option><option value="1">スロット #1</option><option value="2">スロット #2</option><option value="3">スロット #3</option><option value="4">スロット #4</option><option value="5">スロット #5</option><option value="6">スロット #6</option><option value="7">スロット #7</option><option value="8">スロット #8</option></select></div>' +
            '<button type="button" id="swSaveLink" data-sw-act="1" class="primary">このプロファイルの連動設定を保存</button>' +
          '</div>' +
          '<div class="muted" id="swSyncStatus" style="margin-top:6px"></div>' +
        '</div>' +

        '<div class="sw-split">' +
          '<div class="sw-block">' +
            '<div class="sw-block-head"><div><b>オンボードボタン割り当て（全9ボタン）</b><div class="muted">ボタン設定 (0x12) のみを Read-Modify-Write で安全に書き込みます。</div></div>' +
            '<button type="button" id="swSaveButtons" data-sw-act="1" class="primary">オンボードボタン設定をマウスへ書き込む</button></div>' +
            '<div class="tablewrap"><table><thead><tr><th class="num">#</th><th>物理ボタン</th><th>種別</th><th>割り当て内容</th><th>Raw</th></tr></thead><tbody id="swBtnRows"></tbody></table></div>' +
          '</div>' +

          '<div class="sw-block">' +
            '<div class="sw-block-head"><div><b>センサー・ポーリングレート・DPI・LED 設定</b><div class="muted">変更箇所のみ上書きし、後半 LED テーブルとセンサーIDを自動保護します。</div></div>' +
            '<button type="button" id="swSaveConfig" data-sw-act="1" class="primary">センサー・DPI・LED設定をマウスへ書き込む</button></div>' +
            '<div class="row" style="margin-bottom:10px">' +
              '<div class="field"><label for="swReportRate">ポーリングレート</label><select id="swReportRate"><option value="125">125 Hz</option><option value="250">250 Hz</option><option value="500">500 Hz</option><option value="1000">1000 Hz</option></select></div>' +
              '<div class="field"><label for="swActiveDPI">アクティブ DPI スロット</label><select id="swActiveDPI"><option value="0">#1</option><option value="1">#2</option><option value="2">#3</option><option value="3">#4</option><option value="4">#5</option><option value="5">#6</option><option value="6">#7</option><option value="7">#8</option></select></div>' +
              '<div class="field"><label for="swRGBMode">LED エフェクト</label><select id="swRGBMode"><option value="-1">変更しない</option><option value="0">OFF (消灯)</option><option value="1">Rainbow</option><option value="2">単色点灯</option><option value="3">7色ブリージング</option><option value="4">Tail</option><option value="5">スペクトラム</option><option value="9">Wave</option></select></div>' +
              '<div class="field"><label for="swSingleColor">本体LED単色</label><input type="color" id="swSingleColor" value="#0000FF"></div>' +
              '<div class="field"><label for="swColorOrder">LED色順 (緑⇔青が逆ならRBG)</label><select id="swColorOrder"><option value="RGB">RGB (標準)</option><option value="RBG">RBG (G/B入替)</option></select></div>' +
            '</div>' +
            '<div class="tablewrap"><table><thead><tr><th class="num">#</th><th class="check">有効</th><th>DPI</th><th>色</th><th>状態</th></tr></thead><tbody id="swDPIRows"></tbody></table></div>' +
          '</div>' +
        '</div>' +
      '</div>' +
    '</div>';

  $('swReload').addEventListener('click', () => { swDirtyButtons=false; swDirtyConfig=false; swFetch('POST', {op:'read'}); });
  $('swRestore').addEventListener('click', () => {
    if(!confirm('初回読み込み時に保存したバックアップから、マウス本体の設定とボタンを工場出荷状態へ復元しますか？')) return;
    swDirtyButtons=false; swDirtyConfig=false;
    swFetch('POST', {op:'restore-backup'});
  });
  $('swPresetSide').addEventListener('click', () => {
    swDirtyButtons=false;
    swFetch('POST', {op:'apply-preset', preset:'mbm-side-f13-f14'});
  });
  $('swPresetAll').addEventListener('click', () => {
    swDirtyButtons=false;
    swFetch('POST', {op:'apply-preset', preset:'mbm-all-f13-f16'});
  });
  $('swPresetFactory').addEventListener('click', () => {
    swDirtyButtons=false;
    swFetch('POST', {op:'apply-preset', preset:'factory-pawn-plus'});
  });
  $('swSaveButtons').addEventListener('click', () => {
    swDirtyButtons=false;
    swFetch('POST', {op:'write-buttons', buttons: swButtonsDraft});
  });
  $('swSaveConfig').addEventListener('click', () => {
    swDirtyConfig=false;
    const rgbMode = Number($('swRGBMode').value);
    swFetch('POST', {
      op: 'write-config',
      config: {
        reportRateHz: Number($('swReportRate').value),
        activeDpiIndex: Number($('swActiveDPI').value),
        dpiSlots: swDPIDraft,
        rgbEffectMode: rgbMode,
        singleColor: $('swSingleColor').value,
        singleBrightness: 4,
        syncActiveDpiColor: rgbMode === 2,
        colorOrder: $('swColorOrder').value
      }
    });
  });
  $('swColorOrder').addEventListener('change', () => {
    swFetch('POST', { op: 'save-color-order', colorOrder: $('swColorOrder').value });
  });
  $('swSaveLink').addEventListener('click', () => {
    swDirtyLink=false;
    swFetch('POST', {
      op: 'save-profile-link',
      link: {
        Enabled: $('swLinkEnabled').checked,
        ReportRateHz: Number($('swLinkRate').value) || 0,
        ActiveDPISlot: Number($('swLinkDPI').value) || 0,
        LEDMode: Number($('swLinkLEDMode').value) || 0,
        LEDColor: $('swLinkColor').value || '#0000FF'
      }
    });
  });
  host.querySelectorAll('button[data-sw-swatch]').forEach(btn => {
    btn.addEventListener('click', () => {
      $('swLinkEnabled').checked = true;
      $('swLinkLEDMode').value = '1';
      $('swLinkColor').value = btn.dataset.swSwatch;
      $('swLinkColorWrap').classList.remove('hidden');
      swDirtyLink = true;
    });
  });
  ['swReportRate','swActiveDPI','swRGBMode','swSingleColor'].forEach(id => {
    $(id).addEventListener('change', () => { swDirtyConfig = true; if(id==='swActiveDPI') renderDPITable(swData && swData.device); });
    $(id).addEventListener('input', () => { swDirtyConfig = true; });
  });
  ['swLinkEnabled','swLinkRate','swLinkDPI','swLinkLEDMode','swLinkColor'].forEach(id => {
    $(id).addEventListener('change', () => {
      swDirtyLink = true;
      $('swLinkColorWrap').classList.toggle('hidden', $('swLinkLEDMode').value !== '1');
    });
    $(id).addEventListener('input', () => { swDirtyLink = true; });
  });
  return host;
}

function render(){
  const host = ensureHost();
  if(!host || !swData) return;
  const dev = swData.device;
  $('swOfflineBox').classList.toggle('hidden', !!dev);
  $('swMainBody').classList.toggle('hidden', !dev);
  $('swRestore').disabled = swBusy || !swData.hasBackup || !dev;
  if(!dev){
    $('swDeviceSummary').textContent = swData.lastError || '対応マウス未検出';
    $('swOfflineDetail').textContent = swData.lastError || 'VID 258A / PID 1007 のマウスを接続して再読み込みしてください。';
    return;
  }
  $('swDeviceSummary').textContent = dev.productName + ' · VID 0x' + dev.vendorId.toString(16).toUpperCase() + ' / PID 0x' + dev.productId.toString(16).toUpperCase() + ' · 最終読込 ' + (swData.lastReadAt || '―');
  $('swStatFW').textContent = 'FW ' + dev.firmware + ' / ' + dev.sensorName;
  $('swStatRate').textContent = (dev.reportRateHz || 0) + ' Hz';
  const actSlot = dev.dpiSlots && dev.dpiSlots[dev.activeDpiIndex];
  $('swStatDPI').textContent = actSlot ? ('#' + (dev.activeDpiIndex+1) + ' · ' + actSlot.dpi + ' DPI') : '―';
  $('swStatBackup').textContent = swData.hasBackup ? '自動保存済み' : '未作成';

  if(swData.editorProfileName !== swLastProfileName){
    swLastProfileName = swData.editorProfileName || '';
    swDirtyLink = false;
  }
  $('swLinkProfileDesc').textContent = '編集対象プロファイル「' + (swData.editorProfileName || '―') + '」へ切り替わったとき、マウスのLED色・DPI・ポーリングレートを自動で切り替えます。';
  if(!swDirtyLink){
    const lk = swData.profileLink || {};
    $('swLinkEnabled').checked = !!lk.Enabled;
    $('swLinkRate').value = String(lk.ReportRateHz || 0);
    $('swLinkDPI').value = String(lk.ActiveDPISlot || 0);
    $('swLinkLEDMode').value = String(lk.LEDMode || 0);
    $('swLinkColor').value = (lk.LEDColor && /^#[0-9a-fA-F]{6}$/.test(lk.LEDColor)) ? lk.LEDColor : '#0000FF';
    $('swLinkColorWrap').classList.toggle('hidden', $('swLinkLEDMode').value !== '1');
  }
  $('swSyncStatus').textContent = swData.lastSyncMessage || '';
  $('swColorOrder').value = swData.colorOrder || 'RGB';

  if(!swDirtyConfig){
    $('swReportRate').value = String(dev.reportRateHz || 500);
    $('swActiveDPI').value = String(dev.activeDpiIndex || 0);
    $('swRGBMode').value = String(dev.rgbEffectMode != null ? dev.rgbEffectMode : -1);
    $('swSingleColor').value = (dev.singleColor && /^#[0-9a-fA-F]{6}$/.test(dev.singleColor)) ? dev.singleColor : '#0000FF';
    swDPIDraft = (dev.dpiSlots || []).map(s => Object.assign({}, s));
    renderDPITable(dev);
  }
  if(!swDirtyButtons){
    swButtonsDraft = (dev.buttons || []).map(b => Object.assign({}, b));
    renderButtonsTable();
  }
}

window.addEventListener('mbm:plugin-tab-opened', () => {
  ensureHost();
  swFetch('GET').catch(()=>{});
});
window.addEventListener('mbm:profile-changed', () => {
  if(swData) swFetch('GET').catch(()=>{});
});
document.addEventListener('DOMContentLoaded', () => {
  ensureHost();
  swFetch('GET').catch(()=>{});
});
if(document.readyState !== 'loading'){
  ensureHost();
  swFetch('GET').catch(()=>{});
}
})();
`
