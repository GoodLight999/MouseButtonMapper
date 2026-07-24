package main

const webHTML = `<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>MouseButtonMapper</title>
<style>
:root{color-scheme:light dark;--bg:#f5f6f8;--panel:#fff;--panel2:#f9fafb;--text:#17202a;--muted:#667085;--line:#d0d5dd;--accent:#0969da;--accent2:#0759b7;--ok:#177245;--warn:#9a6700;--danger:#b42318;--selected:#e8f2ff;--button:#fff;--shadow:0 1px 2px rgba(16,24,40,.08)}
@media(prefers-color-scheme:dark){:root{--bg:#111418;--panel:#1b2027;--panel2:#161b22;--text:#e6edf3;--muted:#9da7b3;--line:#3a424d;--accent:#58a6ff;--accent2:#79b8ff;--ok:#56d364;--warn:#e3b341;--danger:#ff7b72;--selected:#17324d;--button:#232a33;--shadow:none}}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font-family:"Segoe UI","Yu Gothic UI","Meiryo",sans-serif;font-size:14px}button,input,select{font:inherit}button{border:1px solid var(--line);background:var(--button);color:var(--text);border-radius:6px;padding:7px 12px;cursor:pointer}button:hover:not(:disabled){border-color:var(--accent)}button:disabled{opacity:.48;cursor:not-allowed}.primary{background:var(--accent);border-color:var(--accent);color:#fff;font-weight:700}.primary:hover:not(:disabled){background:var(--accent2)}.danger{color:var(--danger)}.large{padding:10px 18px;font-size:15px}.app{max-width:1500px;margin:0 auto;padding:18px}.topbar{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;margin-bottom:12px}.title{font-size:24px;font-weight:700;margin:0}.subtitle{color:var(--muted);margin-top:4px}.badges{display:flex;flex-wrap:wrap;justify-content:flex-end;gap:7px}.badge{border:1px solid var(--line);border-radius:999px;padding:5px 9px;background:var(--panel);white-space:nowrap}.badge.ok{color:var(--ok)}.badge.warn{color:var(--warn)}.badge.danger{color:var(--danger)}.message{min-height:42px;margin-bottom:12px;border:1px solid var(--line);background:var(--panel);border-radius:7px;padding:10px 12px;color:var(--muted)}.message.error{border-color:var(--danger);color:var(--danger)}.section{background:var(--panel);border:1px solid var(--line);border-radius:8px;margin-bottom:14px;box-shadow:var(--shadow)}.section-head{padding:13px 15px;border-bottom:1px solid var(--line);display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap}.section-title{font-size:17px;font-weight:700}.section-help{color:var(--muted);margin-top:3px}.section-body{padding:14px}.row{display:flex;align-items:center;gap:9px;flex-wrap:wrap}.row+.row{margin-top:10px}.field{display:flex;flex-direction:column;gap:5px;min-width:170px}.field.grow{flex:1;min-width:240px}.field label{font-weight:600}.field small,.help{color:var(--muted)}input[type=text],input[type=number],select{border:1px solid var(--line);background:var(--panel2);color:var(--text);border-radius:6px;padding:8px 9px;min-height:36px}input[type=text]{width:100%}input[type=number]{width:110px}select{min-width:180px}input[type=checkbox]{width:17px;height:17px;vertical-align:middle;accent-color:var(--accent)}.checkline{display:inline-flex;align-items:center;gap:7px;font-weight:600}.split{display:grid;grid-template-columns:minmax(520px,1.2fr) minmax(370px,.8fr);gap:14px}@media(max-width:1000px){.split{grid-template-columns:1fr}.app{padding:10px}.topbar{flex-direction:column}.badges{justify-content:flex-start}}.subpanel{border:1px solid var(--line);border-radius:7px;background:var(--panel2);padding:12px}.subpanel-title{font-weight:700;margin-bottom:9px}.metrics{display:grid;grid-template-columns:repeat(4,minmax(160px,1fr));gap:8px;margin:10px 0}@media(max-width:900px){.metrics{grid-template-columns:repeat(2,1fr)}}@media(max-width:520px){.metrics{grid-template-columns:1fr}}.metric{border:1px solid var(--line);border-radius:6px;background:var(--panel2);padding:9px}.metric .k{color:var(--muted);font-size:12px}.metric .v{font-weight:700;margin-top:3px;word-break:break-all}.tablewrap{border:1px solid var(--line);border-radius:6px;overflow:auto;max-height:390px;background:var(--panel)}table{border-collapse:collapse;width:100%;min-width:850px}th,td{border-bottom:1px solid var(--line);padding:7px 8px;text-align:left;vertical-align:middle}th{position:sticky;top:0;background:var(--panel2);z-index:1;font-weight:700}tbody tr{cursor:pointer}tbody tr:hover{background:color-mix(in srgb,var(--selected) 52%,transparent)}tbody tr.selected{background:var(--selected)}td.check,th.check{text-align:center;width:48px}td.priority{width:70px;text-align:center}.truncate{max-width:280px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.toolbar{display:flex;flex-wrap:wrap;gap:7px;margin-top:9px}.editor-grid{display:grid;grid-template-columns:1fr 1fr;gap:10px}@media(max-width:650px){.editor-grid{grid-template-columns:1fr}}.span2{grid-column:1/-1}.inline-field{display:flex;gap:7px;align-items:flex-end}.inline-field .field{flex:1}.statusbox{border-left:4px solid var(--accent);background:var(--panel2);padding:10px 12px;margin-top:10px}.statusbox .main{font-weight:700}.statusbox .detail{color:var(--muted);margin-top:4px;white-space:pre-wrap}.rule-actions{display:flex;justify-content:space-between;gap:10px;flex-wrap:wrap;margin-top:11px}.record-buttons{display:flex;gap:9px;flex-wrap:wrap}.record{font-weight:700;border-width:2px;padding:10px 16px}.record.active{background:var(--warn);color:#111;border-color:var(--warn)}.savebar{display:flex;align-items:center;justify-content:flex-end;gap:10px;flex-wrap:wrap;border-top:1px solid var(--line);margin-top:12px;padding-top:12px}.footer-grid{display:grid;grid-template-columns:1fr 1fr;gap:12px}@media(max-width:800px){.footer-grid{grid-template-columns:1fr}}code{font-family:Consolas,monospace;font-size:12px;word-break:break-all}.muted{color:var(--muted)}.hidden{display:none!important}
</style>
</head>
<body>
<div class="app">
  <div class="topbar">
    <div><h1 class="title">マウスボタンの割り当て</h1><div class="subtitle" id="versionText">MouseButtonMapper</div></div>
    <div class="badges">
      <span class="badge" id="runBadge">読込中</span>
      <span class="badge" id="effectiveBadge">適用中: 読込中</span>
      <span class="badge" id="autoBadge">自動切替: 読込中</span>
    </div>
  </div>
  <div class="message" id="message" aria-live="polite">設定を読み込んでいます。</div>

  <section class="section">
    <div class="section-head"><div><div class="section-title">プロファイル</div><div class="section-help">「通常時」と「編集対象」は別々に選べます。アプリ自動切替中でも、ほかのプロファイルを編集できます。</div></div></div>
    <div class="section-body">
      <div class="row">
        <div class="field"><label for="baseProfile">通常時に使用するプロファイル</label><select id="baseProfile"></select></div>
        <div class="field"><label for="editProfile">いま編集するプロファイル</label><select id="editProfile"></select></div>
        <div class="row">
          <button type="button" id="newProfile">新規作成</button><button type="button" id="duplicateProfile">複製</button><button type="button" id="renameProfile">名前変更</button><button type="button" id="deleteProfile" class="danger">削除</button>
        </div>
      </div>
    </div>
  </section>

  <section class="section">
    <div class="section-head">
      <div><div class="section-title">アプリごとのプロファイル自動切替</div><div class="section-help">前面にあるウィンドウのプロセスをWindows APIで検知し、対応するプロファイルへ切り替えます。</div></div>
      <label class="checkline"><input type="checkbox" id="autoEnabled"> アプリに応じて自動切替する</label>
    </div>
    <div class="section-body">
      <div class="row">
        <div class="field"><label for="autoDebounce">切替待ち時間</label><div class="row"><input id="autoDebounce" type="number" min="50" max="3000" step="50"><span>ms</span><button type="button" id="saveDebounce" class="primary">切替待ち時間を保存</button></div><small>Alt+Tab直後などの一瞬だけ前面になる画面を無視します。</small></div>
        <button type="button" id="recheckAuto">現在の前面アプリで再判定</button>
      </div>
      <div class="metrics">
        <div class="metric"><div class="k">検知方式</div><div class="v" id="monitorStatus">読込中</div></div>
        <div class="metric"><div class="k">現在の前面ウィンドウ</div><div class="v" id="foregroundNow">読込中</div></div>
        <div class="metric"><div class="k">設定画面を開く直前のアプリ</div><div class="v" id="lastExternal">読込中</div></div>
        <div class="metric"><div class="k">自動切替の判定</div><div class="v" id="autoDecision">読込中</div></div>
      </div>
      <div class="statusbox"><div class="main" id="autoDecisionMain">判定情報を読み込んでいます。</div><div class="detail" id="autoDecisionDetail"></div></div>

      <div class="row" style="margin-top:13px">
        <button type="button" id="addCaptured" class="primary large">直前のアプリを新しい自動切替ルールとして追加</button>
        <button type="button" id="addEmpty">空の自動切替ルールを追加</button>
      </div>
      <p class="help">自動切替ルールは「優先順位 1」から順番に判定します。複数のルールが一致した場合は、数字が最も小さいルールを使用します。1つのルールにプロセス名・タイトル・パスを複数入力した場合は、そのすべてに一致する必要があります。</p>

      <div class="split">
        <div>
          <div class="tablewrap"><table>
            <thead><tr><th class="priority">優先順位</th><th class="check">有効</th><th>ルール名</th><th>判定条件</th><th>使用するプロファイル</th><th>直前アプリとの照合</th></tr></thead>
            <tbody id="bindingRows"></tbody>
          </table></div>
          <div class="toolbar"><button type="button" id="bindingTop">最優先へ</button><button type="button" id="bindingUp">1つ上へ</button><button type="button" id="bindingDown">1つ下へ</button><button type="button" id="bindingBottom">最後へ</button><button type="button" id="duplicateBinding">複製</button><button type="button" id="deleteBinding" class="danger">削除</button></div>
        </div>
        <div class="subpanel">
          <div class="subpanel-title">選択中の自動切替ルールを編集</div>
          <div id="bindingEmpty" class="muted">左の一覧から編集するルールを選択してください。</div>
          <div id="bindingEditor" class="editor-grid hidden">
            <label class="checkline span2"><input type="checkbox" id="bindEnabled"> この自動切替ルールを有効にする</label>
            <div class="field span2"><label for="bindName">ルール名</label><input id="bindName" type="text"></div>
            <div class="field span2"><label for="bindProfile">一致したときに使用するプロファイル</label><select id="bindProfile"></select></div>
            <div class="inline-field span2"><div class="field"><label for="bindProcess">プロセス名</label><input id="bindProcess" type="text" placeholder="例: game.exe または Game*-Shipping.exe"><small>通常はこれだけ指定するのが最も安定します。.exeは省略可能です。</small></div><button type="button" data-capture="process">直前値を入力</button></div>
            <div class="inline-field span2"><div class="field"><label for="bindTitle">ウィンドウタイトルに含む文字</label><input id="bindTitle" type="text" placeholder="必要な場合だけ指定"><small>同じプロセス内で画面ごとに分けたい場合に使用します。</small></div><button type="button" data-capture="title">直前値を入力</button></div>
            <div class="inline-field span2"><div class="field"><label for="bindPath">実行ファイルパスに含む文字</label><input id="bindPath" type="text" placeholder="必要な場合だけ指定"><small>同名の実行ファイルをインストール先で区別したい場合に使用します。</small></div><button type="button" data-capture="path">直前値を入力</button></div>
            <div class="statusbox span2"><div class="main" id="bindingMatchMain"></div><div class="detail" id="bindingMatchDetail"></div></div>
            <div class="savebar span2"><span class="muted">このボタンは、この枠内の項目だけを保存します。</span><button type="button" id="saveBinding" class="primary large">選択中の自動切替ルールを保存</button></div>
          </div>
        </div>
      </div>
    </div>
  </section>

  <section class="section">
    <div class="section-head"><div><div class="section-title">マウスボタンの割り当て</div><div class="section-help" id="ruleContext">編集するプロファイルの割り当てを表示しています。</div></div></div>
    <div class="section-body">
      <div class="tablewrap"><table>
        <thead><tr><th class="check">有効</th><th>この操作をしたら</th><th>実行方法</th><th>この操作を実行する</th><th class="check">最後の入力を通常動作させない</th><th class="check">サイド単押しを取り消す</th></tr></thead>
        <tbody id="ruleRows"></tbody>
      </table></div>
      <div class="rule-actions">
        <div class="toolbar"><button type="button" id="addRule">ルールを追加</button><button type="button" id="deleteRule" class="danger">削除</button><button type="button" id="duplicateRule">複製</button><button type="button" id="ruleTop">最上部へ</button><button type="button" id="ruleUp">上へ</button><button type="button" id="ruleDown">下へ</button><button type="button" id="ruleBottom">最下部へ</button></div>
        <div class="record-buttons"><button type="button" id="recordInput" class="record">● 入力を記録</button><button type="button" id="recordOutput" class="record">● 実行内容を記録</button><button type="button" id="stopRecord">記録を中止</button></div>
      </div>

      <div class="subpanel" style="margin-top:12px">
        <div class="subpanel-title">選択中のマウス割り当てを編集</div>
        <div id="ruleEmpty" class="muted">上の一覧から編集する割り当てを選択してください。</div>
        <div id="ruleEditor" class="editor-grid hidden">
          <label class="checkline"><input type="checkbox" id="ruleEnabled"> この割り当てを有効にする</label><span></span>
          <div class="field"><label for="ruleInput">この操作をしたら</label><input id="ruleInput" type="text"></div>
          <div class="field"><label for="ruleOutput">この操作を実行する</label><input id="ruleOutput" type="text"></div>
          <label class="checkline"><input type="checkbox" id="ruleSuppress"> 最後の入力を通常動作させない</label>
          <label class="checkline"><input type="checkbox" id="rulePrefix"> サイド単押しを取り消す</label>
          <div class="span2 row"><button type="button" id="testOutput">実行内容をテスト</button><span class="help">入力・実行内容の記録は、押したボタンやキーをすべて離すと自動終了します。</span></div>
          <div class="savebar span2"><span class="muted" id="ruleSaveHelp">このボタンは、選択中の割り当てだけを保存します。</span><button type="button" id="saveRule" class="primary large">選択中のマウス割り当てを保存</button></div>
        </div>
      </div>
    </div>
  </section>

  <section class="section">
    <div class="section-head"><div><div class="section-title">動作状態と管理</div></div></div>
    <div class="section-body footer-grid">
      <div class="subpanel"><div class="subpanel-title">現在の入力状態</div><div><b>最後に検出した入力:</b> <span id="lastInput">―</span></div><div style="margin-top:7px"><b>フック状態:</b> <span id="hookStatus">―</span></div><div style="margin-top:7px"><b>設定ファイル:</b> <code id="configPath">―</code></div></div>
      <div class="subpanel"><div class="subpanel-title">アプリ操作</div><div class="toolbar"><button type="button" id="toggleRunning">変換を停止</button><button type="button" id="emergency" class="danger">緊急停止</button><button type="button" id="releaseMods">修飾キー解放</button><button type="button" id="reloadConfig">設定をディスクから再読み込み</button><button type="button" id="openConfig">config.jsonを開く</button><button type="button" id="openFolder">設定フォルダー</button><button type="button" id="openLog">ログを開く</button><button type="button" id="exportConfig">すべてエクスポート</button><button type="button" id="quitApp" class="danger">終了</button></div></div>
    </div>
  </section>
</div>
<script>
(function(){
'use strict';
const $=id=>document.getElementById(id);
let state=null,selectedRule=0,selectedBinding=0,dirtyRule=false,dirtyBinding=false,dirtyDebounce=false,busy=0,pollTimer=null;
const esc=s=>String(s==null?'':s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
function setMessage(text,error){const e=$('message');e.textContent=text||'';e.classList.toggle('error',!!error)}
async function api(url,body){busy++;try{const res=await fetch(url,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});let j={};try{j=await res.json()}catch(_e){}if(!res.ok||j.ok===false)throw new Error(j.error||('HTTP '+res.status));return j}finally{busy--}}
async function load(){if(busy)return;try{const res=await fetch('/api/state',{cache:'no-store'});if(!res.ok)throw new Error('HTTP '+res.status);state=await res.json();render(false)}catch(e){setMessage('状態を取得できません: '+e.message,true)}}
function appText(a){if(!a)return '取得前';const p=a.processName||'プロセス名不明';const t=a.title?' / '+a.title:'';return p+t}
function conditionText(b){const a=[];if(b.processName)a.push('プロセス: '+b.processName);if(b.titleContains)a.push('タイトル: '+b.titleContains);if(b.pathContains)a.push('パス: '+b.pathContains);return a.join(' ＋ ')||'条件未設定'}
function profileOptions(selected){return (state.profiles||[]).map(p=>'<option value="'+esc(p.id)+'" '+(p.id===selected?'selected':'')+'>'+esc(p.name)+'</option>').join('')}
function render(full){if(!state)return;$('versionText').textContent='MouseButtonMapper '+state.version;$('runBadge').textContent=state.status;$('runBadge').className='badge '+(state.emergency?'danger':state.enabled?'ok':'warn');$('effectiveBadge').textContent='現在適用中: '+state.profileName;$('autoBadge').textContent='自動切替: '+(state.autoSwitchEnabled?'有効':'無効');$('autoBadge').className='badge '+(state.autoSwitchEnabled?'ok':'warn');$('toggleRunning').textContent=state.enabled&&!state.emergency?'変換を停止':'変換を開始';
renderProfiles();renderAuto();renderRules();$('lastInput').textContent=state.lastInput+(state.lastInputAt?'  '+state.lastInputAt:'');$('hookStatus').textContent=state.hookStatus;$('configPath').textContent=state.configPath;
if(state.recordingMode){const name=state.recordingMode==='input'?'入力':'実行内容';setMessage(name+'を記録中です。押したボタンやキーをすべて離すと、自動で登録して終了します。');$('recordInput').classList.toggle('active',state.recordingMode==='input');$('recordOutput').classList.toggle('active',state.recordingMode==='output')}else{$('recordInput').classList.remove('active');$('recordOutput').classList.remove('active')}
}
function renderProfiles(){const base=$('baseProfile'),edit=$('editProfile');const opts=(state.profiles||[]).map(p=>'<option value="'+p.index+'">'+esc(p.name)+'</option>').join('');base.innerHTML=opts;edit.innerHTML=opts;base.value=String(state.baseProfile);edit.value=String(state.activeProfile)}
function renderAuto(){if(document.activeElement!==$('autoEnabled'))$('autoEnabled').checked=!!state.autoSwitchEnabled;if(!dirtyDebounce&&document.activeElement!==$('autoDebounce'))$('autoDebounce').value=state.autoDebounceMs;$('monitorStatus').textContent=state.autoMonitorStatus||'起動準備中';$('foregroundNow').textContent=appText(state.foregroundApp);$('lastExternal').textContent=appText(state.lastExternalApp);$('autoDecision').textContent=state.autoDecision||'判定待ち';$('autoDecisionMain').textContent=(state.autoDecision||'判定待ち')+(state.autoDecisionAt?'  '+state.autoDecisionAt:'');$('autoDecisionDetail').textContent=state.autoDecisionDetail||'';
if(selectedBinding>=state.autoBindings.length)selectedBinding=Math.max(0,state.autoBindings.length-1);const body=$('bindingRows');body.innerHTML='';state.autoBindings.forEach((b,i)=>{const tr=document.createElement('tr');if(i===selectedBinding)tr.classList.add('selected');tr.innerHTML='<td class="priority">'+(i+1)+'</td><td class="check"><input type="checkbox" '+(b.enabled?'checked':'')+' aria-label="有効"></td><td class="truncate" title="'+esc(b.name)+'">'+esc(b.name)+'</td><td class="truncate" title="'+esc(conditionText(b))+'">'+esc(conditionText(b))+'</td><td>'+esc(b.profileName)+'</td><td class="truncate" title="'+esc(b.matchSummary)+'">'+(b.matchesLastExternal?'一致':'不一致')+(b.matched?' / 適用中':'')+'</td>';tr.addEventListener('click',()=>{selectedBinding=i;dirtyBinding=false;renderAuto()});tr.querySelector('input').addEventListener('click',e=>{e.stopPropagation();toggleBinding(i)});body.appendChild(tr)});fillBindingEditor()}
function fillBindingEditor(){const b=state.autoBindings[selectedBinding];$('bindingEmpty').classList.toggle('hidden',!!b);$('bindingEditor').classList.toggle('hidden',!b);if(!b)return;if(!dirtyBinding){$('bindEnabled').checked=b.enabled;$('bindName').value=b.name;$('bindProfile').innerHTML=profileOptions(b.profileId);$('bindProcess').value=b.processName;$('bindTitle').value=b.titleContains;$('bindPath').value=b.pathContains}$('bindingMatchMain').textContent=(b.matchesLastExternal?'直前のアプリに一致':'直前のアプリには不一致')+(b.matched?'・現在このルールを適用中':'');$('bindingMatchDetail').textContent=b.matchSummary||'判定条件が未設定です。'}
function renderRules(){if(selectedRule>=state.rules.length)selectedRule=Math.max(0,state.rules.length-1);$('ruleContext').textContent='編集対象: '+state.editorProfileName+' / 現在適用中: '+state.profileName+(state.activeProfile===state.effectiveProfile?'（同じプロファイル）':'（別のプロファイル）');const body=$('ruleRows');body.innerHTML='';state.rules.forEach((r,i)=>{const tr=document.createElement('tr');if(i===selectedRule)tr.classList.add('selected');tr.innerHTML='<td class="check"><input type="checkbox" '+(r.enabled?'checked':'')+'></td><td>'+esc(r.input)+'</td><td>'+esc(r.mode)+'</td><td>'+esc(r.output)+'</td><td class="check"><input type="checkbox" '+(r.suppressTrigger?'checked':'')+' '+(r.suppressTriggerEditable?'':'disabled')+'></td><td class="check"><input type="checkbox" '+(r.suppressPrefix?'checked':'')+' '+(r.suppressPrefixEditable?'':'disabled')+'></td>';tr.addEventListener('click',()=>{selectedRule=i;dirtyRule=false;renderRules()});const c=tr.querySelectorAll('input');c[0].addEventListener('click',e=>{e.stopPropagation();toggleRule(i,'enabled')});c[1].addEventListener('click',e=>{e.stopPropagation();toggleRule(i,'suppressTrigger')});c[2].addEventListener('click',e=>{e.stopPropagation();toggleRule(i,'suppressPrefix')});body.appendChild(tr)});fillRuleEditor()}
function fillRuleEditor(){const r=state.rules[selectedRule];$('ruleEmpty').classList.toggle('hidden',!!r);$('ruleEditor').classList.toggle('hidden',!r);if(!r)return;if(!dirtyRule){$('ruleEnabled').checked=r.enabled;$('ruleInput').value=r.input;$('ruleOutput').value=r.output;$('ruleSuppress').checked=r.suppressTrigger;$('rulePrefix').checked=r.suppressPrefix}$('ruleSuppress').disabled=!r.suppressTriggerEditable;$('rulePrefix').disabled=!r.suppressPrefixEditable;$('ruleSaveHelp').textContent=state.activeProfile===state.effectiveProfile?'保存すると、現在の動作へ即時反映されます。':'保存先は「'+state.editorProfileName+'」です。現在適用中のプロファイルは変更しません。'}
async function applyResult(promise){try{const j=await promise;if(j.state)state=j.state;setMessage(j.message||'完了しました。');render(true)}catch(e){setMessage(e.message,true);await load()}}
async function autoEnabledChanged(){const wanted=$('autoEnabled').checked;$('autoEnabled').disabled=true;await applyResult(api('/api/autoswitch',{op:'settings',enabled:wanted,debounceMs:Number($('autoDebounce').value)||state.autoDebounceMs}));$('autoEnabled').disabled=false}
async function saveDebounce(){dirtyDebounce=false;await applyResult(api('/api/autoswitch',{op:'settings',enabled:$('autoEnabled').checked,debounceMs:Number($('autoDebounce').value)}))}
async function autoOp(op,target,delta){const b=state.autoBindings[selectedBinding];if(!['add-captured','add-empty','recheck'].includes(op)&&!b){setMessage('先に自動切替ルールを選択してください。',true);return}if(op==='delete'&&!confirm('選択中の自動切替ルールを削除しますか？'))return;const payload={op:op,index:selectedBinding,target:target==null?-1:target,delta:delta||0,profileId:state.profiles[state.activeProfile]?state.profiles[state.activeProfile].id:''};const j=api('/api/autoswitch',payload);try{const r=await j;if(r.state)state=r.state;if(op==='add-captured'||op==='add-empty')selectedBinding=Math.max(0,state.autoBindings.length-1);if(op==='delete')selectedBinding=Math.max(0,Math.min(selectedBinding,state.autoBindings.length-1));dirtyBinding=false;setMessage(r.message||'完了しました。');render(true)}catch(e){setMessage(e.message,true);await load()}}
async function toggleBinding(index){selectedBinding=index;await applyResult(api('/api/autoswitch',{op:'toggle',index:index}))}
async function saveBinding(){const b=state.autoBindings[selectedBinding];if(!b)return;dirtyBinding=false;await applyResult(api('/api/autoswitch',{op:'save',index:selectedBinding,enabled:$('bindEnabled').checked,name:$('bindName').value,profileId:$('bindProfile').value,processName:$('bindProcess').value,titleContains:$('bindTitle').value,pathContains:$('bindPath').value}))}
function captureField(kind){const a=state.lastExternalApp||{};let value='';if(kind==='process')value=a.processName||'';if(kind==='title')value=a.title||'';if(kind==='path')value=a.path||'';if(!value){setMessage('直前のアプリからこの項目を取得できませんでした。対象アプリを前面に出してから設定画面へ戻ってください。',true);return}const id=kind==='process'?'bindProcess':kind==='title'?'bindTitle':'bindPath';$(id).value=value;dirtyBinding=true;setMessage('直前のアプリの値を入力欄へ反映しました。まだ保存はしていません。内容を確認して「選択中の自動切替ルールを保存」を押してください。')}
async function profileOp(op,index,name){await applyResult(api('/api/profile',{op:op,index:index,name:name||''}))}
async function createProfile(op){let current=state.profiles[state.activeProfile];let def=op==='duplicate'&&current?current.name+' のコピー':op==='rename'&&current?current.name:'';const name=prompt('プロファイル名',def);if(name===null)return;await profileOp(op,state.activeProfile,name)}
async function deleteProfile(){if(!confirm('編集対象のプロファイル「'+state.editorProfileName+'」を削除しますか？'))return;await profileOp('delete',state.activeProfile,'')}
async function ruleOp(op,target,delta){if(op!=='add'&&!state.rules[selectedRule]){setMessage('先にマウス割り当てを選択してください。',true);return}if(op==='delete'&&!confirm('選択中のマウス割り当てを削除しますか？'))return;try{const j=await api('/api/rule',{op:op,index:selectedRule,target:target==null?-1:target,delta:delta||0});state=j.state;if(op==='add')selectedRule=Math.max(0,state.rules.length-1);if(op==='delete')selectedRule=Math.max(0,Math.min(selectedRule,state.rules.length-1));dirtyRule=false;setMessage(j.message||'完了しました。');render(true)}catch(e){setMessage(e.message,true);await load()}}
async function toggleRule(index,field){selectedRule=index;await applyResult(api('/api/rule',{op:'toggle',index:index,field:field}))}
async function saveRule(){if(!state.rules[selectedRule])return;dirtyRule=false;await applyResult(api('/api/rule',{op:'save',index:selectedRule,enabled:$('ruleEnabled').checked,input:$('ruleInput').value,output:$('ruleOutput').value,suppressTrigger:$('ruleSuppress').checked,suppressPrefix:$('rulePrefix').checked}))}
async function action(name,extra){const payload=Object.assign({action:name,index:selectedRule,output:$('ruleOutput').value},extra||{});await applyResult(api('/api/action',payload))}
async function startRecord(kind){if(!state.rules[selectedRule]){setMessage('先に記録先のマウス割り当てを選択してください。',true);return}dirtyRule=false;await action(kind)}
function markRuleDirty(){dirtyRule=true}function markBindingDirty(){dirtyBinding=true}
$('autoEnabled').addEventListener('change',autoEnabledChanged);$('autoDebounce').addEventListener('input',()=>dirtyDebounce=true);$('saveDebounce').addEventListener('click',saveDebounce);$('recheckAuto').addEventListener('click',()=>autoOp('recheck'));$('addCaptured').addEventListener('click',()=>autoOp('add-captured'));$('addEmpty').addEventListener('click',()=>autoOp('add-empty'));$('bindingTop').addEventListener('click',()=>autoOp('move',0));$('bindingUp').addEventListener('click',()=>autoOp('move',null,-1));$('bindingDown').addEventListener('click',()=>autoOp('move',null,1));$('bindingBottom').addEventListener('click',()=>autoOp('move',999999));$('duplicateBinding').addEventListener('click',()=>autoOp('duplicate'));$('deleteBinding').addEventListener('click',()=>autoOp('delete'));$('saveBinding').addEventListener('click',saveBinding);document.querySelectorAll('[data-capture]').forEach(b=>b.addEventListener('click',()=>captureField(b.dataset.capture)));['bindEnabled','bindName','bindProfile','bindProcess','bindTitle','bindPath'].forEach(id=>$(id).addEventListener('input',markBindingDirty));
$('baseProfile').addEventListener('change',e=>profileOp('set-base',Number(e.target.value),''));$('editProfile').addEventListener('change',async e=>{selectedRule=0;dirtyRule=false;await profileOp('edit',Number(e.target.value),'')});$('newProfile').addEventListener('click',()=>createProfile('new'));$('duplicateProfile').addEventListener('click',()=>createProfile('duplicate'));$('renameProfile').addEventListener('click',()=>createProfile('rename'));$('deleteProfile').addEventListener('click',deleteProfile);
$('addRule').addEventListener('click',()=>ruleOp('add'));$('deleteRule').addEventListener('click',()=>ruleOp('delete'));$('duplicateRule').addEventListener('click',()=>ruleOp('duplicate'));$('ruleTop').addEventListener('click',()=>ruleOp('move',0));$('ruleUp').addEventListener('click',()=>ruleOp('move',null,-1));$('ruleDown').addEventListener('click',()=>ruleOp('move',null,1));$('ruleBottom').addEventListener('click',()=>ruleOp('move',999999));$('saveRule').addEventListener('click',saveRule);$('recordInput').addEventListener('click',()=>startRecord('record-input'));$('recordOutput').addEventListener('click',()=>startRecord('record-output'));$('stopRecord').addEventListener('click',()=>action('record-stop'));$('testOutput').addEventListener('click',()=>action('test-output'));['ruleEnabled','ruleInput','ruleOutput','ruleSuppress','rulePrefix'].forEach(id=>$(id).addEventListener('input',markRuleDirty));
$('toggleRunning').addEventListener('click',()=>action('toggle-running'));$('emergency').addEventListener('click',()=>action('emergency'));$('releaseMods').addEventListener('click',()=>action('release'));$('reloadConfig').addEventListener('click',()=>action('reload'));$('openConfig').addEventListener('click',()=>action('open-config'));$('openFolder').addEventListener('click',()=>action('open-folder'));$('openLog').addEventListener('click',()=>action('open-log'));$('exportConfig').addEventListener('click',()=>action('export'));$('quitApp').addEventListener('click',()=>{if(confirm('MouseButtonMapperを終了しますか？'))action('quit')});
load();pollTimer=setInterval(load,750);
})();
</script>
</body>
</html>
`

const defaultConfigJSON = `{
  "Version": 8,
  "SavedBy": "8.2.0-go-default",
  "SavedAt": "2026-07-05T00:00:00+09:00",
  "ActiveProfileId": "default",
  "AutoSwitch": {"Enabled": false, "DebounceMs": 350, "Bindings": []},
  "Profiles": [
    {
      "Id": "default",
      "Name": "既定",
      "Rules": [
        {"Enabled": true, "Input": [{"Kind":"Mouse","Code":"X2"}], "Mode":"Tap", "Output": [{"Kind":"Key","Code":"17"},{"Kind":"Key","Code":"88"},{"Kind":"Key","Code":"67"}], "SuppressTrigger": true, "SuppressPrefix": false},
        {"Enabled": true, "Input": [{"Kind":"Mouse","Code":"X1"}], "Mode":"Tap", "Output": [{"Kind":"Key","Code":"17"},{"Kind":"Key","Code":"86"}], "SuppressTrigger": true, "SuppressPrefix": false},
        {"Enabled": true, "Input": [{"Kind":"Mouse","Code":"X1"},{"Kind":"Mouse","Code":"WheelUp"}], "Mode":"Tap", "Output": [{"Kind":"Key","Code":"17"},{"Kind":"Key","Code":"91"},{"Kind":"Key","Code":"37"}], "SuppressTrigger": true, "SuppressPrefix": true},
        {"Enabled": true, "Input": [{"Kind":"Mouse","Code":"X1"},{"Kind":"Mouse","Code":"WheelDown"}], "Mode":"Tap", "Output": [{"Kind":"Key","Code":"17"},{"Kind":"Key","Code":"91"},{"Kind":"Key","Code":"39"}], "SuppressTrigger": true, "SuppressPrefix": true},
        {"Enabled": true, "Input": [{"Kind":"Mouse","Code":"X2"},{"Kind":"Mouse","Code":"WheelUp"}], "Mode":"Tap", "Output": [{"Kind":"Key","Code":"91"},{"Kind":"Key","Code":"9"}], "SuppressTrigger": true, "SuppressPrefix": true},
        {"Enabled": true, "Input": [{"Kind":"Mouse","Code":"X2"},{"Kind":"Mouse","Code":"WheelDown"}], "Mode":"Tap", "Output": [{"Kind":"Key","Code":"91"},{"Kind":"Key","Code":"68"}], "SuppressTrigger": true, "SuppressPrefix": true},
        {"Enabled": true, "Input": [{"Kind":"Mouse","Code":"Middle"},{"Kind":"Mouse","Code":"WheelUp"}], "Mode":"Tap", "Output": [{"Kind":"Key","Code":"17"},{"Kind":"Key","Code":"65"}], "SuppressTrigger": true, "SuppressPrefix": false},
        {"Enabled": true, "Input": [{"Kind":"Mouse","Code":"Middle"},{"Kind":"Mouse","Code":"WheelDown"}], "Mode":"Tap", "Output": [{"Kind":"Key","Code":"17"},{"Kind":"Key","Code":"16"},{"Kind":"Key","Code":"86"}], "SuppressTrigger": true, "SuppressPrefix": false},
        {"Enabled": true, "Input": [{"Kind":"Mouse","Code":"Right"},{"Kind":"Mouse","Code":"Middle"},{"Kind":"Mouse","Code":"WheelUp"}], "Mode":"Tap", "Output": [{"Kind":"Key","Code":"17"},{"Kind":"Key","Code":"88"}], "SuppressTrigger": true, "SuppressPrefix": false},
        {"Enabled": true, "Input": [{"Kind":"Mouse","Code":"Right"},{"Kind":"Mouse","Code":"Middle"},{"Kind":"Mouse","Code":"WheelDown"}], "Mode":"Tap", "Output": [{"Kind":"Key","Code":"17"},{"Kind":"Key","Code":"86"}], "SuppressTrigger": true, "SuppressPrefix": false},
        {"Enabled": true, "Input": [{"Kind":"Mouse","Code":"X2"},{"Kind":"Mouse","Code":"Left"}], "Mode":"Tap", "Output": [{"Kind":"Key","Code":"44"}], "SuppressTrigger": false, "SuppressPrefix": true},
        {"Enabled": true, "Input": [{"Kind":"Mouse","Code":"Left"},{"Kind":"Mouse","Code":"X2"}], "Mode":"Tap", "Output": [{"Kind":"Key","Code":"44"}], "SuppressTrigger": true, "SuppressPrefix": false},
        {"Enabled": true, "Input": [{"Kind":"Mouse","Code":"X1"},{"Kind":"Mouse","Code":"Left"}], "Mode":"Tap", "Output": [{"Kind":"Key","Code":"16"},{"Kind":"Key","Code":"17"},{"Kind":"Key","Code":"76"}], "SuppressTrigger": false, "SuppressPrefix": false},
        {"Enabled": true, "Input": [{"Kind":"Mouse","Code":"Left"},{"Kind":"Mouse","Code":"X1"}], "Mode":"Tap", "Output": [{"Kind":"Key","Code":"17"},{"Kind":"Key","Code":"16"},{"Kind":"Key","Code":"76"}], "SuppressTrigger": true, "SuppressPrefix": false}
      ]
    }
  ]
}`
