package main

const webHTML = `<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>MouseButtonMapper</title>
<style>
:root{
  color-scheme:light dark;
  --bg:#f4f5f7;
  --surface:#ffffff;
  --surface-subtle:#f8f9fb;
  --surface-hover:#f0f3f7;
  --text:#131720;
  --text-secondary:#4a5568;
  --muted:#647082;
  --border:#dde2ea;
  --border-strong:#c4cdd9;
  --accent:#1a66e5;
  --accent-hover:#1554c0;
  --accent-soft:#eaf2ff;
  --accent-text:#124db5;
  --ok:#157f43;
  --ok-soft:#e8f7ee;
  --warn:#9a5b00;
  --warn-soft:#fef5e7;
  --danger:#c62828;
  --danger-soft:#fdecec;
  --focus-ring:0 0 0 3px rgba(26,102,229,.22);
  --shadow-sm:0 1px 2px rgba(15,23,42,.05);
  --radius-sm:6px;
  --radius-md:10px;
  --font-sans:"Segoe UI","Yu Gothic UI","Meiryo",system-ui,-apple-system,sans-serif;
  --font-mono:"Cascadia Mono","Consolas",ui-monospace,monospace;
}
@media(prefers-color-scheme:dark){
  :root{
    --bg:#0d1015;
    --surface:#151a22;
    --surface-subtle:#1b212c;
    --surface-hover:#222a37;
    --text:#e6ecf5;
    --text-secondary:#b3bfd1;
    --muted:#8a96a9;
    --border:#283140;
    --border-strong:#39465b;
    --accent:#4c93ff;
    --accent-hover:#6ba7ff;
    --accent-soft:#172844;
    --accent-text:#8ab9ff;
    --ok:#4cd37b;
    --ok-soft:#132d1e;
    --warn:#f0b44c;
    --warn-soft:#302511;
    --danger:#ff6b6b;
    --danger-soft:#35181a;
    --focus-ring:0 0 0 3px rgba(76,147,255,.28);
    --shadow-sm:0 1px 2px rgba(0,0,0,.28);
  }
}
*,*::before,*::after{box-sizing:border-box}
::selection{background:var(--accent-soft);color:var(--text)}
body{
  margin:0;
  background:var(--bg);
  color:var(--text);
  font-family:var(--font-sans);
  font-size:13.5px;
  line-height:1.5;
  -webkit-font-smoothing:antialiased;
}
::-webkit-scrollbar{width:9px;height:9px}
::-webkit-scrollbar-track{background:transparent}
::-webkit-scrollbar-thumb{background:var(--border-strong);border-radius:999px;border:2px solid transparent;background-clip:content-box}
::-webkit-scrollbar-thumb:hover{background:var(--muted);border:2px solid transparent;background-clip:content-box}

button,input,select{font:inherit;color:inherit}
button{
  display:inline-flex;align-items:center;justify-content:center;gap:6px;
  border:1px solid var(--border-strong);
  background:var(--surface);
  color:var(--text);
  border-radius:var(--radius-sm);
  padding:6px 12px;
  min-height:32px;
  font-weight:500;
  cursor:pointer;
  transition:background-color .15s ease,border-color .15s ease,color .15s ease,box-shadow .15s ease;
  white-space:nowrap;
}
button:hover:not(:disabled){background:var(--surface-hover);border-color:var(--muted)}
button:active:not(:disabled){transform:translateY(.5px)}
button:focus-visible,input:focus-visible,select:focus-visible{outline:none;box-shadow:var(--focus-ring);border-color:var(--accent)}
button:disabled{opacity:.45;cursor:not-allowed}
button.primary{background:var(--accent);border-color:var(--accent);color:#fff;font-weight:600}
button.primary:hover:not(:disabled){background:var(--accent-hover);border-color:var(--accent-hover)}
button.danger{color:var(--danger);border-color:color-mix(in srgb,var(--danger) 35%,var(--border))}
button.danger:hover:not(:disabled){background:var(--danger-soft);border-color:var(--danger)}
button.large{padding:8px 16px;min-height:36px;font-size:13.5px}
button.ghost{border-color:transparent;background:transparent}
button.ghost:hover:not(:disabled){background:var(--surface-hover);border-color:var(--border)}
button.small{padding:4px 9px;min-height:28px;font-size:12.5px}

input[type=text],input[type=number],input[type=search],select{
  border:1px solid var(--border-strong);
  background:var(--surface);
  color:var(--text);
  border-radius:var(--radius-sm);
  padding:6px 10px;
  min-height:34px;
  transition:border-color .15s ease,box-shadow .15s ease;
}
input[type=text],input[type=search]{width:100%}
input[type=number]{width:104px;font-variant-numeric:tabular-nums}
input[type=color]{width:42px;height:30px;padding:2px;border:1px solid var(--border-strong);border-radius:var(--radius-sm);background:var(--surface);cursor:pointer}
select{min-width:160px}
input[type=checkbox]{width:16px;height:16px;margin:0;accent-color:var(--accent);cursor:pointer;flex-shrink:0}
code{font-family:var(--font-mono);font-size:12px;word-break:break-all}

.app{max-width:1440px;margin:0 auto;padding:16px 20px 28px}
.topbar{
  display:flex;align-items:center;justify-content:space-between;gap:14px;flex-wrap:wrap;
  padding-bottom:12px;border-bottom:1px solid var(--border);
}
.brand{display:flex;align-items:baseline;gap:10px;flex-wrap:wrap}
.title{font-size:19px;font-weight:700;letter-spacing:-.015em;margin:0}
.subtitle{color:var(--muted);font-size:12.5px;font-variant-numeric:tabular-nums}
.badges{display:flex;align-items:center;flex-wrap:wrap;gap:7px}
.badge,.pill{
  display:inline-flex;align-items:center;gap:5px;
  border:1px solid var(--border);
  border-radius:999px;
  padding:3px 10px;
  font-size:12px;font-weight:600;
  background:var(--surface);
  color:var(--text-secondary);
  white-space:nowrap;
}
.badge.ok,.pill.ok{color:var(--ok);background:var(--ok-soft);border-color:color-mix(in srgb,var(--ok) 30%,transparent)}
.badge.warn,.pill.warn{color:var(--warn);background:var(--warn-soft);border-color:color-mix(in srgb,var(--warn) 30%,transparent)}
.badge.danger,.pill.danger{color:var(--danger);background:var(--danger-soft);border-color:color-mix(in srgb,var(--danger) 30%,transparent)}

.profile-bar{
  display:flex;align-items:flex-end;justify-content:space-between;gap:14px;flex-wrap:wrap;
  margin-top:12px;padding:11px 14px;
  background:var(--surface);border:1px solid var(--border);border-radius:var(--radius-md);
  box-shadow:var(--shadow-sm);
}
.profile-controls{display:flex;align-items:flex-end;gap:12px;flex-wrap:wrap}
.nav-tabs{
  display:flex;align-items:center;gap:4px;flex-wrap:wrap;
  margin-top:12px;padding:4px;
  background:var(--surface-subtle);border:1px solid var(--border);border-radius:var(--radius-md);
}
.nav-tab{
  border:1px solid transparent;background:transparent;color:var(--text-secondary);
  padding:7px 14px;min-height:34px;border-radius:7px;font-weight:600;
}
.nav-tab:hover:not(:disabled){color:var(--text);background:var(--surface-hover);border-color:transparent}
.nav-tab.active{
  background:var(--surface);color:var(--accent-text);border-color:var(--border);
  box-shadow:var(--shadow-sm);
}
.tab-count{
  font-size:11px;font-variant-numeric:tabular-nums;
  padding:1px 6px;border-radius:999px;background:var(--surface-subtle);color:var(--muted);
}
.nav-tab.active .tab-count{background:var(--accent-soft);color:var(--accent-text)}

.message{
  margin-top:10px;padding:9px 13px;min-height:38px;
  border:1px solid var(--border);border-radius:var(--radius-sm);
  background:var(--surface);color:var(--text-secondary);
  display:flex;align-items:center;gap:8px;
}
.message.error{border-color:var(--danger);background:var(--danger-soft);color:var(--danger);font-weight:600}

.callout{
  margin-top:10px;padding:10px 13px;
  border:1px solid var(--border);border-radius:var(--radius-sm);
  background:var(--surface-subtle);
}
.callout.warn{border-color:color-mix(in srgb,var(--warn) 45%,var(--border));background:var(--warn-soft);color:var(--warn)}
.callout.warn .muted{color:color-mix(in srgb,var(--warn) 85%,var(--text))}

.workspace{margin-top:12px}
.section{
  background:var(--surface);
  border:1px solid var(--border);
  border-radius:var(--radius-md);
  margin-bottom:14px;
  box-shadow:var(--shadow-sm);
}
.section.tab-hidden{display:none}
.section-head{
  padding:13px 16px;border-bottom:1px solid var(--border);
  display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap;
}
.section-title{font-size:15.5px;font-weight:700;letter-spacing:-.01em}
.section-help{color:var(--muted);font-size:12.5px;margin-top:2px}
.section-body{padding:15px 16px}

.row{display:flex;align-items:center;gap:10px;flex-wrap:wrap}
.row+.row{margin-top:10px}
.field{display:flex;flex-direction:column;gap:4px;min-width:155px}
.field.grow{flex:1;min-width:220px}
.field label{font-size:12.5px;font-weight:600;color:var(--text-secondary)}
.field small,.help{color:var(--muted);font-size:12px}
.checkline{display:inline-flex;align-items:center;gap:7px;font-weight:600;cursor:pointer;user-select:none}
.toolbar{display:flex;flex-wrap:wrap;align-items:center;gap:7px}
.toolbar.spaced{margin-top:10px}

.split{display:grid;grid-template-columns:minmax(500px,1.25fr) minmax(370px,.85fr);gap:16px;align-items:start}
.sw-split{display:grid;grid-template-columns:minmax(480px,1.15fr) minmax(380px,.85fr);gap:16px;align-items:start;margin-top:14px}
@media(max-width:1040px){.split,.sw-split{grid-template-columns:1fr}.app{padding:12px}}

.subpanel{
  border:1px solid var(--border);border-radius:var(--radius-sm);
  background:var(--surface-subtle);padding:13px 14px;
}
.subpanel-title{font-weight:700;font-size:14px;margin-bottom:10px}

.metrics,.sw-stats{display:grid;grid-template-columns:repeat(4,minmax(160px,1fr));gap:8px;margin:10px 0}
@media(max-width:860px){.metrics,.sw-stats{grid-template-columns:repeat(2,1fr)}}
@media(max-width:520px){.metrics,.sw-stats{grid-template-columns:1fr}}
.metric{
  border:1px solid var(--border);border-radius:var(--radius-sm);
  background:var(--surface-subtle);padding:9px 11px;
}
.metric .k{color:var(--muted);font-size:11.5px;font-weight:600}
.metric .v{font-weight:700;margin-top:3px;word-break:break-word;font-variant-numeric:tabular-nums}

.tablewrap{
  border:1px solid var(--border);border-radius:var(--radius-sm);
  overflow:auto;max-height:420px;background:var(--surface);
}
table{border-collapse:collapse;width:100%;min-width:680px;font-variant-numeric:tabular-nums}
th,td{border-bottom:1px solid var(--border);padding:7px 9px;text-align:left;vertical-align:middle}
th{position:sticky;top:0;background:var(--surface-subtle);z-index:1;font-size:12px;font-weight:700;color:var(--text-secondary)}
tbody tr{cursor:pointer;transition:background-color .1s ease}
tbody tr:hover{background:var(--surface-hover)}
tbody tr.selected{background:var(--accent-soft)}
td.check,th.check{text-align:center;width:48px}
td.priority,th.priority,td.num,th.num{width:54px;text-align:center}
.truncate{max-width:260px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}

.editor-grid{display:grid;grid-template-columns:1fr 1fr;gap:11px}
@media(max-width:640px){.editor-grid{grid-template-columns:1fr}}
.span2{grid-column:1/-1}
.inline-field{display:flex;gap:7px;align-items:flex-end}
.inline-field .field{flex:1}

.statusbox{
  border:1px solid var(--border);border-radius:var(--radius-sm);
  background:var(--surface-subtle);padding:10px 12px;margin-top:10px;
}
.statusbox .main{font-weight:700}
.statusbox .detail{color:var(--muted);font-size:12.5px;margin-top:3px;white-space:pre-wrap}

.rule-actions{display:flex;justify-content:space-between;align-items:center;gap:10px;flex-wrap:wrap;margin-top:10px}
.record-buttons{display:flex;gap:8px;flex-wrap:wrap}
.record{
  font-weight:700;border:1.5px solid var(--accent);color:var(--accent-text);
  background:var(--accent-soft);padding:7px 14px;min-height:36px;
}
.record:hover:not(:disabled){background:color-mix(in srgb,var(--accent) 18%,var(--surface))}
.record.active{background:var(--warn);color:#111;border-color:var(--warn)}

.longpress-box{
  grid-column:1/-1;
  border:1px solid color-mix(in srgb,var(--accent) 40%,var(--border));
  border-radius:var(--radius-sm);
  background:color-mix(in srgb,var(--accent-soft) 35%,var(--surface));
  padding:12px;
}
.longpress-grid{display:grid;grid-template-columns:190px 1fr;gap:10px;margin-top:10px}
@media(max-width:680px){.longpress-grid{grid-template-columns:1fr}}

.savebar{
  display:flex;align-items:center;justify-content:space-between;gap:10px;flex-wrap:wrap;
  border-top:1px solid var(--border);margin-top:12px;padding-top:12px;
}
.footer-grid{display:grid;grid-template-columns:1fr 1fr;gap:14px}
@media(max-width:820px){.footer-grid{grid-template-columns:1fr}}

.picker-box{
  margin-top:12px;padding:12px;
  border:1px solid var(--border);border-radius:var(--radius-sm);
  background:var(--surface-subtle);
}
.picker-list{
  margin-top:8px;max-height:220px;overflow:auto;
  border:1px solid var(--border);border-radius:var(--radius-sm);
  background:var(--surface);
}
.picker-item{
  display:flex;align-items:center;justify-content:space-between;gap:10px;
  padding:7px 10px;border-bottom:1px solid var(--border);
}
.picker-item:last-child{border-bottom:none}
.picker-item:hover{background:var(--surface-hover)}
.picker-meta{min-width:0;flex:1}
.picker-proc{font-weight:700;display:flex;align-items:center;gap:6px}
.picker-title{color:var(--muted);font-size:12px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}

.plugin-list{display:grid;grid-template-columns:repeat(auto-fill,minmax(320px,1fr));gap:10px;margin-bottom:14px}
.plugin-item{
  border:1px solid var(--border);border-radius:var(--radius-sm);
  background:var(--surface-subtle);padding:12px;
  display:flex;flex-direction:column;gap:8px;
}
.plugin-item-head{display:flex;align-items:center;justify-content:space-between;gap:8px}
.sw-panel{border-top:1px solid var(--border);padding-top:14px;margin-top:6px}
.sw-header{display:flex;align-items:flex-start;justify-content:space-between;gap:12px;flex-wrap:wrap;margin-bottom:10px}
.sw-title{margin:0;font-size:15.5px;font-weight:700}
.sw-sub{margin:2px 0 0;font-size:12.5px}
.sw-block{margin-top:14px;padding-top:12px;border-top:1px solid var(--border)}
.sw-block-head{display:flex;align-items:center;justify-content:space-between;gap:10px;flex-wrap:wrap;margin-bottom:9px}
.sw-key-wrap{display:flex;align-items:center;gap:7px;flex-wrap:wrap}
.sw-mod{display:inline-flex;align-items:center;gap:3px;font-size:12px;color:var(--text-secondary)}
.muted{color:var(--muted)}
.hidden{display:none!important}
</style>
</head>
<body>
<div class="app">
  <header class="topbar">
    <div class="brand">
      <h1 class="title">MouseButtonMapper</h1>
      <span class="subtitle" id="versionText">マウスとキーボードの割り当て</span>
    </div>
    <div class="badges">
      <span class="badge" id="runBadge">読込中</span>
      <span class="badge" id="effectiveBadge">現在適用中: 読込中</span>
      <span class="badge" id="autoBadge">自動切替: 読込中</span>
      <button type="button" id="toggleRunning" class="small">変換を停止</button>
      <button type="button" id="emergency" class="danger small">緊急停止</button>
    </div>
  </header>

  <div class="profile-bar">
    <div class="profile-controls">
      <div class="field">
        <label for="baseProfile">通常時に使用するプロファイル</label>
        <select id="baseProfile"></select>
      </div>
      <div class="field">
        <label for="editProfile">いま編集するプロファイル</label>
        <select id="editProfile"></select>
      </div>
    </div>
    <div class="toolbar">
      <button type="button" id="newProfile">新規作成</button>
      <button type="button" id="duplicateProfile">複製</button>
      <button type="button" id="renameProfile">名前変更</button>
      <button type="button" id="deleteProfile" class="danger">削除</button>
    </div>
  </div>

  <nav class="nav-tabs" role="tablist" aria-label="設定セクション">
    <button type="button" class="nav-tab active" data-tab="rules" role="tab" aria-selected="true">割り当て <span class="tab-count" id="tabCountRules">0</span></button>
    <button type="button" class="nav-tab" data-tab="autoswitch" role="tab" aria-selected="false">アプリ自動切替 <span class="tab-count" id="tabCountAuto">0</span></button>
    <button type="button" class="nav-tab" data-tab="plugins" role="tab" aria-selected="false">マウス設定・プラグイン <span class="tab-count" id="tabCountPlugins">1</span></button>
    <button type="button" class="nav-tab" data-tab="system" role="tab" aria-selected="false">動作状態と管理</button>
  </nav>

  <div class="message" id="message" aria-live="polite">設定を読み込んでいます。</div>
  <div class="callout warn hidden" id="elevationWarning">
    <b>管理者権限アプリ（UIPI 制限）を検知しました</b>
    <div class="muted" id="elevationWarningDetail"></div>
  </div>

  <main class="workspace">
    <section class="section" data-tab-panel="rules">
      <div class="section-head">
        <div>
          <div class="section-title">マウス・キーボードの割り当て</div>
          <div class="section-help" id="ruleContext">編集するプロファイルの割り当てを表示しています。</div>
        </div>
        <div class="record-buttons">
          <button type="button" id="recordInput" class="record">● 入力を記録</button>
          <button type="button" id="recordOutput" class="record">● 短押し時の実行内容を記録</button>
          <button type="button" id="stopRecord">記録を中止</button>
        </div>
      </div>
      <div class="section-body">
        <div class="tablewrap">
          <table>
            <thead>
              <tr>
                <th class="check">有効</th>
                <th>この操作をしたら</th>
                <th>実行内容</th>
                <th>長押し / 方式</th>
                <th class="check">通常動作を抑止</th>
                <th class="check">サイド単押し取消</th>
              </tr>
            </thead>
            <tbody id="ruleRows"></tbody>
          </table>
        </div>
        <div class="rule-actions">
          <div class="toolbar">
            <button type="button" id="addRule" class="primary">ルールを追加</button>
            <button type="button" id="duplicateRule">複製</button>
            <button type="button" id="ruleTop">最上部へ</button>
            <button type="button" id="ruleUp">上へ</button>
            <button type="button" id="ruleDown">下へ</button>
            <button type="button" id="ruleBottom">最下部へ</button>
            <button type="button" id="deleteRule" class="danger">削除</button>
          </div>
          <span class="help">ヒント: Ray Pawn+ の右サイドをプラグインで F13/F14 にすると、通常キーと衝突せず記録できます。</span>
        </div>

        <div class="subpanel" style="margin-top:14px">
          <div class="subpanel-title">選択中の割り当てを編集</div>
          <div id="ruleEmpty" class="muted">上の一覧から編集する割り当てを選択するか、「ルールを追加」を押してください。</div>
          <div id="ruleEditor" class="editor-grid hidden">
            <label class="checkline"><input type="checkbox" id="ruleEnabled"> この割り当てを有効にする</label>
            <div class="field">
              <label for="ruleMode">実行方式</label>
              <select id="ruleMode">
                <option value="Tap">タップ / 長押し分岐（既定）</option>
                <option value="Hold">そのまま長押し連動（押している間ずっと押し続ける）</option>
              </select>
            </div>
            <div class="field">
              <label for="ruleInput">この操作をしたら</label>
              <input id="ruleInput" type="text" placeholder="例: 左サイド2 / 右サイド1 / DPI+ボタン">
              <div class="toolbar" id="inputQuickButtons" style="margin-top:4px" aria-label="マウスボタンを入力へ追加"></div>
              <small>「入力を記録」で実際のボタン・キー・ホイールを押して離すと自動反映されます。</small>
            </div>
            <div class="field">
              <label for="ruleOutput" id="ruleOutputLabel">短押し時の実行内容</label>
              <input id="ruleOutput" type="text" placeholder="例: Ctrl + C（長押し専用なら空欄）">
              <div class="toolbar" style="margin-top:4px" aria-label="マウス操作を追加">
                <span class="muted" style="font-size:12px">マウス操作:</span>
                <button type="button" class="small" data-output-target="ruleOutput" data-output-token="左クリック">左クリック</button>
                <button type="button" class="small" data-output-target="ruleOutput" data-output-token="右クリック">右クリック</button>
                <button type="button" class="small" data-output-target="ruleOutput" data-output-token="中クリック">中クリック</button>
                <button type="button" class="small" data-output-target="ruleOutput" data-output-token="左サイド1">左サイド1</button>
                <button type="button" class="small" data-output-target="ruleOutput" data-output-token="左サイド2">左サイド2</button>
                <button type="button" class="small" data-output-target="ruleOutput" data-output-token="ホイール上">ホイール↑</button>
                <button type="button" class="small" data-output-target="ruleOutput" data-output-token="ホイール下">ホイール↓</button>
              </div>
            </div>
            <label class="checkline"><input type="checkbox" id="ruleSuppress"> 最後の入力を通常動作させない</label>
            <label class="checkline"><input type="checkbox" id="rulePrefix"> サイド単押しを取り消す</label>
            <div class="span2 row">
              <button type="button" id="testOutput">実行内容をテスト</button>
              <span class="help" id="ruleModeHelp">記録モードは押したキー・ボタンをすべて離した瞬間に自動終了します。</span>
            </div>
            <div class="longpress-box" id="longPressBox">
              <label class="checkline"><input type="checkbox" id="ruleLongEnabled"> この割り当てで長押し判定を使う</label>
              <div id="longPressOptions" class="longpress-grid hidden">
                <div class="field">
                  <label for="ruleLongMs">長押しと判定する時間</label>
                  <div class="row"><input id="ruleLongMs" type="number" min="100" max="5000" step="50"><span>ms</span></div>
                  <small>100〜5000ms（既定 500ms）</small>
                </div>
                <div class="field">
                  <label for="ruleLongAction">長押ししたとき</label>
                  <select id="ruleLongAction">
                    <option value="Execute">別の実行内容を実行する</option>
                    <option value="Cancel">短押し時の実行をキャンセルする</option>
                  </select>
                </div>
                <div id="longOutputBlock" class="field span2">
                  <label for="ruleLongOutput">長押し時の実行内容</label>
                  <input id="ruleLongOutput" type="text">
                  <div class="toolbar" style="margin-top:4px" aria-label="長押し出力へマウス操作を追加">
                    <span class="muted" style="font-size:12px">マウス操作:</span>
                    <button type="button" class="small" data-output-target="ruleLongOutput" data-output-token="左クリック">左クリック</button>
                    <button type="button" class="small" data-output-target="ruleLongOutput" data-output-token="右クリック">右クリック</button>
                    <button type="button" class="small" data-output-target="ruleLongOutput" data-output-token="中クリック">中クリック</button>
                    <button type="button" class="small" data-output-target="ruleLongOutput" data-output-token="左サイド1">左サイド1</button>
                    <button type="button" class="small" data-output-target="ruleLongOutput" data-output-token="左サイド2">左サイド2</button>
                    <button type="button" class="small" data-output-target="ruleLongOutput" data-output-token="ホイール上">ホイール↑</button>
                    <button type="button" class="small" data-output-target="ruleLongOutput" data-output-token="ホイール下">ホイール↓</button>
                  </div>
                  <div class="record-buttons" style="margin-top:8px">
                    <button type="button" id="recordLongOutput" class="record">● 長押し時の実行内容を記録</button>
                    <button type="button" id="testLongOutput">長押し時の実行内容をテスト</button>
                  </div>
                </div>
                <div class="statusbox span2">
                  <div class="main">長押し判定の動作</div>
                  <div class="detail" id="longPressHelp">短く離すと短押し時の内容を実行し、指定時間まで押し続けると長押し時の内容を実行します。</div>
                </div>
              </div>
            </div>
            <div class="savebar span2">
              <span class="muted" id="ruleSaveHelp">このボタンは、選択中の割り当てだけを保存します。</span>
              <button type="button" id="saveRule" class="primary large">選択中の割り当てを保存</button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="section tab-hidden" data-tab-panel="autoswitch">
      <div class="section-head">
        <div>
          <div class="section-title">アプリごとのプロファイル自動切替</div>
          <div class="section-help">前面ウィンドウを自動検知し、ゲームやソフトに応じたプロファイル（＋マウス本体設定）へ切り替えます。</div>
        </div>
        <label class="checkline"><input type="checkbox" id="autoEnabled"> アプリに応じて自動切替する</label>
      </div>
      <div class="section-body">
        <div class="row">
          <div class="field">
            <label for="autoDebounce">切替待ち時間</label>
            <div class="row">
              <input id="autoDebounce" type="number" min="50" max="3000" step="50"><span>ms</span>
              <button type="button" id="saveDebounce" class="primary">切替待ち時間を保存</button>
            </div>
            <small>Alt+Tab直後などの一瞬の切替を無視します。</small>
          </div>
          <button type="button" id="recheckAuto">現在の前面アプリで再判定</button>
          <button type="button" id="toggleRunningApps">起動中のアプリ一覧から追加</button>
        </div>

        <div id="runningAppsBox" class="picker-box hidden">
          <div class="row" style="justify-content:space-between">
            <b>起動中のアプリ・ゲーム一覧</b>
            <div class="toolbar">
              <input type="search" id="runningAppsFilter" placeholder="プロセス名・タイトルで絞り込み" style="width:240px">
              <button type="button" id="refreshRunningApps" class="small">一覧を更新</button>
            </div>
          </div>
          <div class="picker-list" id="runningAppsList"></div>
        </div>

        <div class="metrics">
          <div class="metric"><div class="k">検知方式</div><div class="v" id="monitorStatus">読込中</div></div>
          <div class="metric"><div class="k">現在の前面ウィンドウ</div><div class="v" id="foregroundNow">読込中</div></div>
          <div class="metric"><div class="k">設定画面を開く直前のアプリ</div><div class="v" id="lastExternal">読込中</div></div>
          <div class="metric"><div class="k">自動切替の判定</div><div class="v" id="autoDecision">読込中</div></div>
        </div>
        <div class="statusbox">
          <div class="main" id="autoDecisionMain">判定情報を読み込んでいます。</div>
          <div class="detail" id="autoDecisionDetail"></div>
        </div>

        <div class="toolbar spaced">
          <button type="button" id="addCaptured" class="primary">直前のアプリを新しい自動切替ルールとして追加</button>
          <button type="button" id="addEmpty">空の自動切替ルールを追加</button>
        </div>
        <p class="help">優先順位 1 から順に判定し、最初に一致したルールを適用します。複数条件を入力した場合は AND（すべて一致）です。</p>

        <div class="split">
          <div>
            <div class="tablewrap">
              <table>
                <thead>
                  <tr>
                    <th class="priority">優先</th>
                    <th class="check">有効</th>
                    <th>ルール名</th>
                    <th>判定条件</th>
                    <th>プロファイル</th>
                    <th>照合結果</th>
                  </tr>
                </thead>
                <tbody id="bindingRows"></tbody>
              </table>
            </div>
            <div class="toolbar spaced">
              <button type="button" id="bindingTop">最優先へ</button>
              <button type="button" id="bindingUp">1つ上へ</button>
              <button type="button" id="bindingDown">1つ下へ</button>
              <button type="button" id="bindingBottom">最後へ</button>
              <button type="button" id="duplicateBinding">複製</button>
              <button type="button" id="deleteBinding" class="danger">削除</button>
            </div>
          </div>
          <div class="subpanel">
            <div class="subpanel-title">選択中の自動切替ルールを編集</div>
            <div id="bindingEmpty" class="muted">左の一覧から編集するルールを選択してください。</div>
            <div id="bindingEditor" class="editor-grid hidden">
              <label class="checkline span2"><input type="checkbox" id="bindEnabled"> この自動切替ルールを有効にする</label>
              <div class="field span2"><label for="bindName">ルール名</label><input id="bindName" type="text"></div>
              <div class="field span2"><label for="bindProfile">一致したときに使用するプロファイル</label><select id="bindProfile"></select></div>
              <div class="inline-field span2">
                <div class="field">
                  <label for="bindProcess">プロセス名</label>
                  <input id="bindProcess" type="text" placeholder="例: Client-Win64-Shipping.exe または game*.exe">
                  <small>最も安定する指定方法です。.exe 省略やワイルドカード(*)に対応。</small>
                </div>
                <button type="button" data-capture="process">直前値を入力</button>
              </div>
              <div class="inline-field span2">
                <div class="field">
                  <label for="bindTitle">ウィンドウタイトルに含む文字</label>
                  <input id="bindTitle" type="text" placeholder="必要な場合だけ指定">
                </div>
                <button type="button" data-capture="title">直前値を入力</button>
              </div>
              <div class="inline-field span2">
                <div class="field">
                  <label for="bindPath">実行ファイルパスに含む文字</label>
                  <input id="bindPath" type="text" placeholder="必要な場合だけ指定">
                </div>
                <button type="button" data-capture="path">直前値を入力</button>
              </div>
              <div class="statusbox span2">
                <div class="main" id="bindingMatchMain"></div>
                <div class="detail" id="bindingMatchDetail"></div>
              </div>
              <div class="savebar span2">
                <span class="muted">この枠内の自動切替ルールだけを保存します。</span>
                <button type="button" id="saveBinding" class="primary large">選択中の自動切替ルールを保存</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="section tab-hidden" data-tab-panel="plugins">
      <div class="section-head">
        <div>
          <div class="section-title">マウス設定・プラグイン拡張</div>
          <div class="section-help">中核の入力フックと分離されたハードウェアプラグインを管理します。不要なプラグインはオフにできます。</div>
        </div>
        <button type="button" id="refreshPlugins">プラグイン状態を更新</button>
      </div>
      <div class="section-body">
        <div class="plugin-list" id="pluginList"></div>
        <div id="sinoWealthPluginHost"></div>
      </div>
    </section>

    <section class="section tab-hidden" data-tab-panel="system">
      <div class="section-head">
        <div>
          <div class="section-title">動作状態と管理</div>
          <div class="section-help">フック健全性、権限状態、設定ファイル、バックアップ、実験機能の管理を行います。</div>
        </div>
      </div>
      <div class="section-body footer-grid">
        <div class="subpanel">
          <div class="subpanel-title">現在の動作・診断情報</div>
          <div><b>最後に検出した入力:</b> <span id="lastInput">―</span></div>
          <div style="margin-top:7px"><b>フック状態:</b> <span id="hookStatus">―</span></div>
          <div style="margin-top:7px"><b>実行権限:</b> <span id="elevationStatus">―</span></div>
          <div style="margin-top:7px"><b>設定ファイル:</b> <code id="configPath">―</code></div>
        </div>
        <div class="subpanel">
          <div class="subpanel-title">アプリ操作</div>
          <div class="toolbar">
            <button type="button" id="releaseMods">修飾キー解放</button>
            <button type="button" id="reloadConfig">設定を再読み込み</button>
            <button type="button" id="openConfig">config.jsonを開く</button>
            <button type="button" id="openFolder">設定フォルダー</button>
            <button type="button" id="openLog">ログを開く</button>
            <button type="button" id="exportConfig">すべてエクスポート</button>
            <button type="button" id="quitApp" class="danger">終了</button>
          </div>
        </div>
      </div>
    </section>
  </main>
</div>

<script>
(function(){
'use strict';
const $=id=>document.getElementById(id);
let state=null,selectedRule=0,selectedBinding=0,dirtyRule=false,dirtyBinding=false,dirtyDebounce=false,busy=0;
let activeTab=(function(){try{return localStorage.getItem('mbm_tab')||'rules'}catch(_e){return 'rules'}})();
let lastProfilesSig='',lastRulesSig='',lastBindingsSig='',lastMouseButtonsSig='',runningAppsCache=[];
const esc=s=>String(s==null?'':s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));

function setMessage(text,error){const e=$('message');e.textContent=text||'';e.classList.toggle('error',!!error)}

function switchTab(tab){
  activeTab=tab;
  try{localStorage.setItem('mbm_tab',tab)}catch(_e){}
  document.querySelectorAll('.nav-tab').forEach(btn=>{
    const on=btn.dataset.tab===tab;
    btn.classList.toggle('active',on);
    btn.setAttribute('aria-selected',on?'true':'false');
  });
  document.querySelectorAll('section[data-tab-panel]').forEach(sec=>{
    sec.classList.toggle('tab-hidden',sec.dataset.tabPanel!==tab);
  });
  // Keep dynamically injected controller sections aligned with the Rules tab.
  ['controllerFeatureSection','joyConSection'].forEach(id=>{
    const el=$(id);
    if(el) el.classList.toggle('tab-hidden',tab!=='rules');
  });
  if(tab==='plugins'){
    loadPlugins();
    window.dispatchEvent(new CustomEvent('mbm:plugin-tab-opened'));
  }
}

async function api(url,body){
  busy++;
  try{
    const res=await fetch(url,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
    let j={};try{j=await res.json()}catch(_e){}
    if(!res.ok||j.ok===false)throw new Error(j.error||('HTTP '+res.status));
    return j;
  }finally{busy--}
}

async function load(){
  if(busy)return;
  try{
    const res=await fetch('/api/state',{cache:'no-store'});
    if(!res.ok)throw new Error('HTTP '+res.status);
    state=await res.json();
    render(false);
  }catch(e){setMessage('状態を取得できません: '+e.message,true)}
}

function appText(a){
  if(!a||(!a.processName&&!a.title))return '取得前';
  const p=a.processName||'プロセス名不明';
  const t=a.title?' / '+a.title:'';
  return p+t+(a.elevated?' [管理者]':'');
}
function conditionText(b){
  const a=[];
  if(b.processName)a.push('プロセス: '+b.processName);
  if(b.titleContains)a.push('タイトル: '+b.titleContains);
  if(b.pathContains)a.push('パス: '+b.pathContains);
  return a.join(' ＋ ')||'条件未設定';
}
function profileOptions(selected){
  return (state.profiles||[]).map(p=>'<option value="'+esc(p.id)+'" '+(p.id===selected?'selected':'')+'>'+esc(p.name)+'</option>').join('');
}

function render(force){
  if(!state)return;
  $('versionText').textContent='v'+state.version+' · マウス・キーボード割り当て';
  $('runBadge').textContent=state.status;
  $('runBadge').className='badge '+(state.emergency?'danger':state.enabled?'ok':'warn');
  $('effectiveBadge').textContent='適用中: '+state.profileName;
  $('autoBadge').textContent='自動切替: '+(state.autoSwitchEnabled?'有効':'無効');
  $('autoBadge').className='badge '+(state.autoSwitchEnabled?'ok':'warn');
  $('toggleRunning').textContent=state.enabled&&!state.emergency?'変換を停止':'変換を開始';
  $('tabCountRules').textContent=String((state.rules||[]).length);
  $('tabCountAuto').textContent=String((state.autoBindings||[]).length);

  const extElev=state.lastExternalApp&&state.lastExternalApp.elevated;
  const showWarn=!!extElev&&!state.selfElevated;
  $('elevationWarning').classList.toggle('hidden',!showWarn);
  if(showWarn){
    $('elevationWarningDetail').textContent='直前のアプリ（'+(state.lastExternalApp.processName||'不明')+'）は管理者権限で実行されています。現在の MouseButtonMapper は通常権限のため、ゲーム内でボタン変換が届かない場合は MouseButtonMapper を管理者として実行してください。';
  }
  $('elevationStatus').textContent=state.selfElevated?'管理者権限（昇格済み・管理者ゲーム対応）':'通常ユーザー権限';

  renderProfiles(force);
  renderAuto(force);
  renderRules(force);
  $('lastInput').textContent=state.lastInput+(state.lastInputAt?'  ('+state.lastInputAt+')':'');
  $('hookStatus').textContent=state.hookStatus;
  $('configPath').textContent=state.configPath;

  // Keep injected controller sections synchronized with current tab.
  ['controllerFeatureSection','joyConSection'].forEach(id=>{
    const el=$(id);
    if(el) el.classList.toggle('tab-hidden',activeTab!=='rules');
  });

  if(state.recordingMode){
    const names={input:'入力',output:'短押し時の実行内容','long-output':'長押し時の実行内容'};
    const name=names[state.recordingMode]||'実行内容';
    setMessage(name+'を記録中です。ボタン・キー・ホイールを操作し、すべて離すと自動で確定します。');
    $('recordInput').classList.toggle('active',state.recordingMode==='input');
    $('recordOutput').classList.toggle('active',state.recordingMode==='output');
    $('recordLongOutput').classList.toggle('active',state.recordingMode==='long-output');
    if(state.recordedText){
      if(state.recordingMode==='input')$('ruleInput').value=state.recordedText;
      if(state.recordingMode==='output')$('ruleOutput').value=state.recordedText;
      if(state.recordingMode==='long-output')$('ruleLongOutput').value=state.recordedText;
    }
  }else{
    $('recordInput').classList.remove('active');
    $('recordOutput').classList.remove('active');
    $('recordLongOutput').classList.remove('active');
  }
}

function renderProfiles(force){
  const base=$('baseProfile'),edit=$('editProfile');
  const sig=JSON.stringify((state.profiles||[]).map(p=>[p.index,p.id,p.name]));
  if(force||sig!==lastProfilesSig){
    lastProfilesSig=sig;
    const opts=(state.profiles||[]).map(p=>'<option value="'+p.index+'">'+esc(p.name)+'</option>').join('');
    base.innerHTML=opts;
    edit.innerHTML=opts;
  }
  if(document.activeElement!==base)base.value=String(state.baseProfile);
  if(document.activeElement!==edit)edit.value=String(state.activeProfile);
}

function renderAuto(force){
  if(document.activeElement!==$('autoEnabled'))$('autoEnabled').checked=!!state.autoSwitchEnabled;
  if(!dirtyDebounce&&document.activeElement!==$('autoDebounce'))$('autoDebounce').value=state.autoDebounceMs;
  $('monitorStatus').textContent=state.autoMonitorStatus||'起動準備中';
  $('foregroundNow').textContent=appText(state.foregroundApp);
  $('lastExternal').textContent=appText(state.lastExternalApp);
  $('autoDecision').textContent=state.autoDecision||'判定待ち';
  $('autoDecisionMain').textContent=(state.autoDecision||'判定待ち')+(state.autoDecisionAt?'  ('+state.autoDecisionAt+')':'');
  $('autoDecisionDetail').textContent=state.autoDecisionDetail||'';

  if(selectedBinding>=state.autoBindings.length)selectedBinding=Math.max(0,state.autoBindings.length-1);
  const sig=JSON.stringify({sel:selectedBinding,list:state.autoBindings});
  if(force||sig!==lastBindingsSig){
    lastBindingsSig=sig;
    const body=$('bindingRows');
    const prevScroll=body.parentElement?body.parentElement.parentElement.scrollTop:0;
    body.innerHTML='';
    state.autoBindings.forEach((b,i)=>{
      const tr=document.createElement('tr');
      if(i===selectedBinding)tr.classList.add('selected');
      const badge=b.matched?'<span class="pill ok">適用中</span>':(b.matchesLastExternal?'<span class="pill ok">一致</span>':'<span class="muted">不一致</span>');
      tr.innerHTML='<td class="priority">'+(i+1)+'</td><td class="check"><input type="checkbox" '+(b.enabled?'checked':'')+' aria-label="有効"></td><td class="truncate" title="'+esc(b.name)+'"><b>'+esc(b.name)+'</b></td><td class="truncate" title="'+esc(conditionText(b))+'">'+esc(conditionText(b))+'</td><td>'+esc(b.profileName)+'</td><td title="'+esc(b.matchSummary)+'">'+badge+'</td>';
      tr.addEventListener('click',()=>{selectedBinding=i;dirtyBinding=false;renderAuto(true)});
      tr.querySelector('input').addEventListener('click',e=>{e.stopPropagation();toggleBinding(i)});
      body.appendChild(tr);
    });
    if(body.parentElement&&body.parentElement.parentElement)body.parentElement.parentElement.scrollTop=prevScroll;
  }
  fillBindingEditor();
}

function fillBindingEditor(){
  const b=state.autoBindings[selectedBinding];
  $('bindingEmpty').classList.toggle('hidden',!!b);
  $('bindingEditor').classList.toggle('hidden',!b);
  if(!b)return;
  if(!dirtyBinding){
    $('bindEnabled').checked=b.enabled;
    $('bindName').value=b.name;
    $('bindProfile').innerHTML=profileOptions(b.profileId);
    $('bindProcess').value=b.processName;
    $('bindTitle').value=b.titleContains;
    $('bindPath').value=b.pathContains;
  }
  $('bindingMatchMain').textContent=(b.matchesLastExternal?'直前のアプリに一致':'直前のアプリには不一致')+(b.matched?' ・ 現在このルールを適用中':'');
  $('bindingMatchDetail').textContent=b.matchSummary||'判定条件が未設定です。';
}

function renderInputQuickButtons(){
  const wrap=$('inputQuickButtons');
  if(!wrap)return;
  const logical=(state&&state.mouseButtons)||[];
  const sig=JSON.stringify(logical);
  if(sig===lastMouseButtonsSig)return;
  lastMouseButtonsSig=sig;
  const base=['左サイド1','左サイド2','中クリック','ホイール上','ホイール下'];
  const tokens=base.concat(logical.map(b=>b.name));
  wrap.innerHTML='<span class="muted" style="font-size:12px">マウスボタン:</span>'+tokens.map(t=>'<button type="button" class="small" data-input-token="'+esc(t)+'">'+esc(t)+'</button>').join('');
  wrap.querySelectorAll('button[data-input-token]').forEach(btn=>{
    btn.addEventListener('click',()=>appendOutputToken('ruleInput',btn.dataset.inputToken));
  });
}

function renderRules(force){
  renderInputQuickButtons();
  if(selectedRule>=state.rules.length)selectedRule=Math.max(0,state.rules.length-1);
  $('ruleContext').textContent='編集対象: '+state.editorProfileName+' / 現在適用中: '+state.profileName+(state.activeProfile===state.effectiveProfile?'（同じプロファイル）':'（別のプロファイル）');
  const sig=JSON.stringify({sel:selectedRule,list:state.rules});
  if(force||sig!==lastRulesSig){
    lastRulesSig=sig;
    const body=$('ruleRows');
    const prevScroll=body.parentElement?body.parentElement.parentElement.scrollTop:0;
    body.innerHTML='';
    state.rules.forEach((r,i)=>{
      const tr=document.createElement('tr');
      if(i===selectedRule)tr.classList.add('selected');
      const modeCell=r.mode==='Hold'?'<span class="pill ok">そのまま長押し (Hold)</span>':esc(r.longPressSummary);
      tr.innerHTML='<td class="check"><input type="checkbox" '+(r.enabled?'checked':'')+'></td><td><b>'+esc(r.input||'（未設定）')+'</b></td><td>'+esc(r.output||'（なし）')+'</td><td>'+modeCell+'</td><td class="check"><input type="checkbox" '+(r.suppressTrigger?'checked':'')+' '+(r.suppressTriggerEditable?'':'disabled')+'></td><td class="check"><input type="checkbox" '+(r.suppressPrefix?'checked':'')+' '+(r.suppressPrefixEditable?'':'disabled')+'></td>';
      tr.addEventListener('click',()=>{selectedRule=i;dirtyRule=false;renderRules(true)});
      const c=tr.querySelectorAll('input');
      c[0].addEventListener('click',e=>{e.stopPropagation();toggleRule(i,'enabled')});
      c[1].addEventListener('click',e=>{e.stopPropagation();toggleRule(i,'suppressTrigger')});
      c[2].addEventListener('click',e=>{e.stopPropagation();toggleRule(i,'suppressPrefix')});
      body.appendChild(tr);
    });
    if(body.parentElement&&body.parentElement.parentElement)body.parentElement.parentElement.scrollTop=prevScroll;
  }
  fillRuleEditor();
}

function fillRuleEditor(){
  const r=state.rules[selectedRule];
  $('ruleEmpty').classList.toggle('hidden',!!r);
  $('ruleEditor').classList.toggle('hidden',!r);
  if(!r)return;
  if(!dirtyRule){
    $('ruleEnabled').checked=r.enabled;
    $('ruleMode').value=(r.mode==='Hold')?'Hold':'Tap';
    $('ruleInput').value=r.input;
    $('ruleOutput').value=r.output;
    $('ruleSuppress').checked=r.suppressTrigger;
    $('rulePrefix').checked=r.suppressPrefix;
    $('ruleLongEnabled').checked=r.longPressEnabled;
    $('ruleLongMs').value=r.longPressMs||500;
    $('ruleLongAction').value=r.longPressAction||'Execute';
    $('ruleLongOutput').value=r.longPressOutput||'';
  }
  $('ruleSuppress').disabled=!r.suppressTriggerEditable;
  $('rulePrefix').disabled=!r.suppressPrefixEditable;
  updateLongPressEditor();
  $('ruleSaveHelp').textContent=state.activeProfile===state.effectiveProfile?'保存すると、現在の動作へ即時反映されます。':'保存先は「'+state.editorProfileName+'」です。現在適用中のプロファイルは変更しません。';
}

function updateLongPressEditor(){
  const isHold=$('ruleMode').value==='Hold';
  $('longPressBox').classList.toggle('hidden',isHold);
  $('ruleOutputLabel').textContent=isHold?'押している間ずっと押し続ける内容（Hold）':'短押し時の実行内容';
  $('ruleModeHelp').textContent=isHold?'そのまま長押し連動モード: マウスボタンを押した瞬間にDOWN、離した瞬間にUPを送るため、ゲーム側の短押し・長押しチャージの両方に自然に対応します。':'記録モードは押したキー・ボタンをすべて離した瞬間に自動終了します。';
  if(isHold&&$('ruleLongEnabled').checked)$('ruleLongEnabled').checked=false;
  const enabled=!isHold&&$('ruleLongEnabled').checked;
  const cancel=$('ruleLongAction').value==='Cancel';
  $('longPressOptions').classList.toggle('hidden',!enabled);
  $('longOutputBlock').classList.toggle('hidden',cancel);
  $('longPressHelp').textContent=cancel?'指定時間以上押し続けた場合、短押し時の実行内容を発動せずに終了します。':'指定時間より前に離すと短押し時の内容を実行し、指定時間まで押し続けると長押し時の内容を1回だけ実行します。';
}

async function loadRunningApps(){
  try{
    const res=await fetch('/api/running-apps',{cache:'no-store'});
    const j=await res.json();
    if(!res.ok||j.ok===false)throw new Error(j.error||('HTTP '+res.status));
    runningAppsCache=j.apps||[];
    renderRunningApps();
  }catch(e){setMessage('起動中アプリ一覧を取得できません: '+e.message,true)}
}

function renderRunningApps(){
  const list=$('runningAppsList');
  const q=($('runningAppsFilter').value||'').trim().toLowerCase();
  const filtered=runningAppsCache.filter(a=>!q||a.processName.toLowerCase().includes(q)||a.title.toLowerCase().includes(q));
  if(!filtered.length){
    list.innerHTML='<div class="picker-item"><span class="muted">一致する起動中アプリがありません。</span></div>';
    return;
  }
  list.innerHTML='';
  filtered.forEach(a=>{
    const row=document.createElement('div');
    row.className='picker-item';
    row.innerHTML='<div class="picker-meta"><div class="picker-proc">'+esc(a.processName)+(a.elevated?' <span class="pill warn">管理者</span>':'')+'</div><div class="picker-title" title="'+esc(a.title)+'">'+esc(a.title)+'</div></div><button type="button" class="small primary">このアプリを追加</button>';
    row.querySelector('button').addEventListener('click',async()=>{
      const profId=state&&state.profiles[state.activeProfile]?state.profiles[state.activeProfile].id:'';
      try{
        const r=await api('/api/autoswitch',{op:'add-from-app',profileId:profId,name:a.processName.replace(/\.exe$/i,''),processName:a.processName,pathContains:''});
        if(r.state)state=r.state;
        selectedBinding=Math.max(0,state.autoBindings.length-1);
        dirtyBinding=false;
        setMessage(r.message||'自動切替ルールを追加しました。');
        render(true);
      }catch(e){setMessage(e.message,true)}
    });
    list.appendChild(row);
  });
}

async function loadPlugins(){
  try{
    const res=await fetch('/api/plugins',{cache:'no-store'});
    const j=await res.json();
    if(!res.ok||j.ok===false)return;
    const list=j.plugins||[];
    $('tabCountPlugins').textContent=String(list.length);
    const wrap=$('pluginList');
    wrap.innerHTML='';
    list.forEach(p=>{
      const card=document.createElement('div');
      card.className='plugin-item';
      card.innerHTML=
        '<div class="plugin-item-head"><b>'+esc(p.name)+'</b><span class="pill '+(p.detected?'ok':'')+'">'+(p.detected?'検出済み':'未検出')+'</span></div>'+
        '<div class="muted" style="font-size:12.5px">'+esc(p.description)+'</div>'+
        '<div style="font-size:12px">'+esc(p.summary)+'</div>'+
        '<label class="checkline"><input type="checkbox" data-plugin-id="'+esc(p.id)+'"'+(p.enabled?' checked':'')+'> このプラグインを有効にする</label>';
      card.querySelector('input').addEventListener('change',async e=>{
        try{
          const r=await api('/api/plugins',{op:'set-enabled',id:p.id,enabled:e.target.checked});
          if(r.message)setMessage(r.message);
          loadPlugins();
          window.dispatchEvent(new CustomEvent('mbm:plugin-tab-opened'));
        }catch(err){setMessage(err.message,true);loadPlugins()}
      });
      wrap.appendChild(card);
    });
  }catch(_e){}
}

async function applyResult(promise){
  try{
    const j=await promise;
    if(j.state)state=j.state;
    setMessage(j.message||'完了しました。');
    render(true);
  }catch(e){setMessage(e.message,true);await load()}
}
async function autoEnabledChanged(){
  const wanted=$('autoEnabled').checked;
  $('autoEnabled').disabled=true;
  await applyResult(api('/api/autoswitch',{op:'settings',enabled:wanted,debounceMs:Number($('autoDebounce').value)||state.autoDebounceMs}));
  $('autoEnabled').disabled=false;
}
async function saveDebounce(){dirtyDebounce=false;await applyResult(api('/api/autoswitch',{op:'settings',enabled:$('autoEnabled').checked,debounceMs:Number($('autoDebounce').value)}))}
async function autoOp(op,target,delta){
  const b=state.autoBindings[selectedBinding];
  if(!['add-captured','add-empty','recheck'].includes(op)&&!b){setMessage('先に自動切替ルールを選択してください。',true);return}
  if(op==='delete'&&!confirm('選択中の自動切替ルールを削除しますか？'))return;
  const payload={op:op,index:selectedBinding,target:target==null?-1:target,delta:delta||0,profileId:state.profiles[state.activeProfile]?state.profiles[state.activeProfile].id:''};
  try{
    const r=await api('/api/autoswitch',payload);
    if(r.state)state=r.state;
    if(op==='add-captured'||op==='add-empty')selectedBinding=Math.max(0,state.autoBindings.length-1);
    if(op==='delete')selectedBinding=Math.max(0,Math.min(selectedBinding,state.autoBindings.length-1));
    dirtyBinding=false;
    setMessage(r.message||'完了しました。');
    render(true);
  }catch(e){setMessage(e.message,true);await load()}
}
async function toggleBinding(index){selectedBinding=index;await applyResult(api('/api/autoswitch',{op:'toggle',index:index}))}
async function saveBinding(){
  const b=state.autoBindings[selectedBinding];
  if(!b)return;
  dirtyBinding=false;
  await applyResult(api('/api/autoswitch',{op:'save',index:selectedBinding,enabled:$('bindEnabled').checked,name:$('bindName').value,profileId:$('bindProfile').value,processName:$('bindProcess').value,titleContains:$('bindTitle').value,pathContains:$('bindPath').value}));
}
function captureField(kind){
  const a=state.lastExternalApp||{};
  let value='';
  if(kind==='process')value=a.processName||'';
  if(kind==='title')value=a.title||'';
  if(kind==='path')value=a.path||'';
  if(!value){setMessage('直前のアプリからこの項目を取得できませんでした。対象アプリを一度前面に出してください。',true);return}
  const id=kind==='process'?'bindProcess':kind==='title'?'bindTitle':'bindPath';
  $(id).value=value;
  dirtyBinding=true;
  setMessage('直前のアプリの値を入力欄へ反映しました。「選択中の自動切替ルールを保存」で確定してください。');
}
async function profileOp(op,index,name){
  await applyResult(api('/api/profile',{op:op,index:index,name:name||''}));
  window.dispatchEvent(new CustomEvent('mbm:profile-changed'));
}
async function createProfile(op){
  const current=state.profiles[state.activeProfile];
  const def=op==='duplicate'&&current?current.name+' のコピー':op==='rename'&&current?current.name:'';
  const name=prompt('プロファイル名',def);
  if(name===null)return;
  await profileOp(op,state.activeProfile,name);
}
async function deleteProfile(){
  if(!confirm('編集対象のプロファイル「'+state.editorProfileName+'」を削除しますか？'))return;
  await profileOp('delete',state.activeProfile,'');
}
async function ruleOp(op,target,delta){
  if(op!=='add'&&!state.rules[selectedRule]){setMessage('先に割り当てを選択してください。',true);return}
  if(op==='delete'&&!confirm('選択中の割り当てを削除しますか？'))return;
  try{
    const j=await api('/api/rule',{op:op,index:selectedRule,target:target==null?-1:target,delta:delta||0});
    state=j.state;
    if(op==='add')selectedRule=Math.max(0,state.rules.length-1);
    if(op==='delete')selectedRule=Math.max(0,Math.min(selectedRule,state.rules.length-1));
    dirtyRule=false;
    setMessage(j.message||'完了しました。');
    render(true);
  }catch(e){setMessage(e.message,true);await load()}
}
async function toggleRule(index,field){selectedRule=index;await applyResult(api('/api/rule',{op:'toggle',index:index,field:field}))}
function appendOutputToken(targetId,token){
  const input=$(targetId);
  if(!input)return;
  const parts=input.value.split('+').map(v=>v.trim()).filter(Boolean);
  if(!parts.includes(token))parts.push(token);
  input.value=parts.join(' + ');
  input.dispatchEvent(new Event('input',{bubbles:true}));
  input.focus();
  setMessage(token+' を実行内容へ追加しました。');
}
document.querySelectorAll('[data-output-target]').forEach(button=>button.addEventListener('click',()=>appendOutputToken(button.dataset.outputTarget,button.dataset.outputToken)));
async function saveRule(){
  if(!state.rules[selectedRule])return;
  dirtyRule=false;
  const mode=$('ruleMode').value==='Hold'?'Hold':'Tap';
  await applyResult(api('/api/rule',{op:'save',index:selectedRule,enabled:$('ruleEnabled').checked,mode:mode,input:$('ruleInput').value,output:$('ruleOutput').value,suppressTrigger:$('ruleSuppress').checked,suppressPrefix:$('rulePrefix').checked,longPressEnabled:mode==='Hold'?false:$('ruleLongEnabled').checked,longPressMs:Number($('ruleLongMs').value)||500,longPressAction:$('ruleLongAction').value,longPressOutput:$('ruleLongOutput').value}));
}
async function action(name,extra){
  const payload=Object.assign({action:name,index:selectedRule,output:$('ruleOutput').value,longOutput:$('ruleLongOutput').value},extra||{});
  await applyResult(api('/api/action',payload));
}
async function startRecord(kind){
  if(!state.rules[selectedRule]){setMessage('先に記録先の割り当てを選択してください。',true);return}
  dirtyRule=false;
  await action(kind);
}
function markRuleDirty(){dirtyRule=true}
function markBindingDirty(){dirtyBinding=true}

document.querySelectorAll('.nav-tab').forEach(btn=>btn.addEventListener('click',()=>switchTab(btn.dataset.tab)));
$('toggleRunningApps').addEventListener('click',()=>{
  const box=$('runningAppsBox');
  const open=box.classList.contains('hidden');
  box.classList.toggle('hidden',!open);
  if(open)loadRunningApps();
});
$('refreshRunningApps').addEventListener('click',loadRunningApps);
$('runningAppsFilter').addEventListener('input',renderRunningApps);
$('refreshPlugins').addEventListener('click',()=>{loadPlugins();window.dispatchEvent(new CustomEvent('mbm:plugin-tab-opened'))});

$('autoEnabled').addEventListener('change',autoEnabledChanged);
$('autoDebounce').addEventListener('input',()=>{dirtyDebounce=true});
$('saveDebounce').addEventListener('click',saveDebounce);
$('recheckAuto').addEventListener('click',()=>autoOp('recheck'));
$('addCaptured').addEventListener('click',()=>autoOp('add-captured'));
$('addEmpty').addEventListener('click',()=>autoOp('add-empty'));
$('bindingTop').addEventListener('click',()=>autoOp('move',0));
$('bindingUp').addEventListener('click',()=>autoOp('move',null,-1));
$('bindingDown').addEventListener('click',()=>autoOp('move',null,1));
$('bindingBottom').addEventListener('click',()=>autoOp('move',999999));
$('duplicateBinding').addEventListener('click',()=>autoOp('duplicate'));
$('deleteBinding').addEventListener('click',()=>autoOp('delete'));
$('saveBinding').addEventListener('click',saveBinding);
document.querySelectorAll('[data-capture]').forEach(b=>b.addEventListener('click',()=>captureField(b.dataset.capture)));
['bindEnabled','bindName','bindProfile','bindProcess','bindTitle','bindPath'].forEach(id=>{
  $(id).addEventListener('input',markBindingDirty);
  $(id).addEventListener('change',markBindingDirty);
});

$('baseProfile').addEventListener('change',e=>profileOp('set-base',Number(e.target.value),''));
$('editProfile').addEventListener('change',async e=>{selectedRule=0;dirtyRule=false;await profileOp('edit',Number(e.target.value),'')});
$('newProfile').addEventListener('click',()=>createProfile('new'));
$('duplicateProfile').addEventListener('click',()=>createProfile('duplicate'));
$('renameProfile').addEventListener('click',()=>createProfile('rename'));
$('deleteProfile').addEventListener('click',deleteProfile);

$('addRule').addEventListener('click',()=>ruleOp('add'));
$('deleteRule').addEventListener('click',()=>ruleOp('delete'));
$('duplicateRule').addEventListener('click',()=>ruleOp('duplicate'));
$('ruleTop').addEventListener('click',()=>ruleOp('move',0));
$('ruleUp').addEventListener('click',()=>ruleOp('move',null,-1));
$('ruleDown').addEventListener('click',()=>ruleOp('move',null,1));
$('ruleBottom').addEventListener('click',()=>ruleOp('move',999999));
$('saveRule').addEventListener('click',saveRule);
$('recordInput').addEventListener('click',()=>startRecord('record-input'));
$('recordOutput').addEventListener('click',()=>startRecord('record-output'));
$('recordLongOutput').addEventListener('click',()=>startRecord('record-long-output'));
$('stopRecord').addEventListener('click',()=>action('record-stop'));
$('testOutput').addEventListener('click',()=>action('test-output'));
$('testLongOutput').addEventListener('click',()=>action('test-long-output'));
$('ruleMode').addEventListener('change',()=>{markRuleDirty();updateLongPressEditor()});
$('ruleLongEnabled').addEventListener('change',()=>{markRuleDirty();updateLongPressEditor()});
$('ruleLongAction').addEventListener('change',()=>{markRuleDirty();updateLongPressEditor()});
['ruleEnabled','ruleInput','ruleOutput','ruleSuppress','rulePrefix','ruleLongMs','ruleLongOutput'].forEach(id=>{
  $(id).addEventListener('input',markRuleDirty);
  $(id).addEventListener('change',markRuleDirty);
});

$('toggleRunning').addEventListener('click',()=>action('toggle-running'));
$('emergency').addEventListener('click',()=>action('emergency'));
$('releaseMods').addEventListener('click',()=>action('release'));
$('reloadConfig').addEventListener('click',()=>action('reload'));
$('openConfig').addEventListener('click',()=>action('open-config'));
$('openFolder').addEventListener('click',()=>action('open-folder'));
$('openLog').addEventListener('click',()=>action('open-log'));
$('exportConfig').addEventListener('click',()=>action('export'));
$('quitApp').addEventListener('click',()=>{if(confirm('MouseButtonMapperを終了しますか？'))action('quit')});

switchTab(activeTab);
load();
loadPlugins();
setInterval(load,750);
})();
</script>
<script src="/joycon-ui.js"></script>
<script src="/plugins/sinowealth-ui.js"></script>
</body>
</html>
`

const defaultConfigJSON = `{
  "Version": 11,
  "SavedBy": "8.5.0-dev-go-default",
  "SavedAt": "2026-07-05T00:00:00+09:00",
  "ActiveProfileId": "default",
  "AutoSwitch": {"Enabled": false, "DebounceMs": 350, "Bindings": []},
  "Controller": {"Enabled": false, "Visible": false},
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
