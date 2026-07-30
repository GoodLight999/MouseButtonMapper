//go:build windows

package main

import (
	"io"
	"net/http"
)

const joyConUIJS = `(function(){
'use strict';
const byId=id=>document.getElementById(id);
const originalFetch=window.fetch.bind(window);
let joyState=null;
let mainState=null;
let joyDirty=false;
let joyBusy=0;
let controllerBusy=0;
let lastEditorProfile=-1;

window.fetch=async function(input,init){
  const url=typeof input==='string'?input:(input&&input.url)||'';
  let nextInit=init;
  if(url.endsWith('/api/rule')&&init&&String(init.method||'GET').toUpperCase()==='POST'&&init.body){
    try{
      const body=JSON.parse(init.body);
      const mode=byId('joyRuleMode');
      if(body&&body.op==='save'&&mode)body.mode=mode.value;
      nextInit=Object.assign({},init,{body:JSON.stringify(body)});
    }catch(_e){}
  }
  const response=await originalFetch(input,nextInit);
  if(url.endsWith('/api/state')&&(!init||String(init.method||'GET').toUpperCase()==='GET')){
    response.clone().json().then(state=>{
      mainState=state;
      setTimeout(()=>renderControllerFeature(state),0);
      if(state&&state.controllerVisible&&state.controllerEnabled&&Number(state.activeProfile)!==lastEditorProfile){
        lastEditorProfile=Number(state.activeProfile);
        joyDirty=false;
        loadJoyCon();
      }
    }).catch(()=>{});
  }
  return response;
};

function findRuleSection(){
  return [...document.querySelectorAll('section.section')].find(s=>{
    const t=s.querySelector('.section-title');
    return t&&(t.textContent.includes('マウスボタンの割り当て')||t.textContent.includes('マウス・キーボード・ゲームコントローラーの割り当て'));
  });
}

function injectStyle(){
  if(byId('joyConStyle'))return;
  const style=document.createElement('style');
  style.id='joyConStyle';
  style.textContent='.controller-gate{display:flex;align-items:flex-start;gap:12px}.controller-gate-copy{flex:1}.controller-disabled-note{margin-top:8px}.controller-actions{display:flex;gap:8px;flex-wrap:wrap;margin-top:12px}.experimental-restore{font-size:12px;padding:5px 8px}.joy-grid{display:grid;grid-template-columns:minmax(330px,.9fr) minmax(430px,1.1fr);gap:14px}.joy-metrics{display:grid;grid-template-columns:repeat(2,minmax(150px,1fr));gap:8px}.joy-stick{position:relative;width:190px;height:190px;border:1px solid var(--line);border-radius:50%;background:var(--panel);margin:10px auto}.joy-stick:before,.joy-stick:after{content:"";position:absolute;background:var(--line)}.joy-stick:before{left:50%;top:8%;bottom:8%;width:1px}.joy-stick:after{top:50%;left:8%;right:8%;height:1px}.joy-stick-dead{position:absolute;border:1px dashed var(--warn);border-radius:50%;left:35%;top:35%;width:30%;height:30%}.joy-stick-dot{position:absolute;width:18px;height:18px;border-radius:50%;background:var(--accent);left:calc(50% - 9px);top:calc(50% - 9px);transition:left .04s linear,top .04s linear}.joy-settings{display:grid;grid-template-columns:1fr 1fr;gap:10px}.joy-full{grid-column:1/-1}.joy-value{font-family:Consolas,monospace}.joy-buttons{display:flex;gap:8px;flex-wrap:wrap;margin-top:12px}.hid-list{width:100%;min-height:210px;font-family:Consolas,"Yu Gothic UI",sans-serif;font-size:12px}.hid-warning{border-left:4px solid var(--warn);padding:9px 11px;background:var(--panel);margin-top:8px}@media(max-width:900px){.joy-grid{grid-template-columns:1fr}}';
  document.head.appendChild(style);
}

function injectRestoreControl(){
  injectStyle();
  if(byId('experimentalFeatureRestore'))return;
  const sections=[...document.querySelectorAll('section.section')];
  const management=sections.find(s=>{const t=s.querySelector('.section-title');return t&&t.textContent.includes('動作状態と管理')});
  const toolbar=management&&management.querySelector('.toolbar');
  if(!toolbar)return;
  const button=document.createElement('button');
  button.type='button';
  button.id='experimentalFeatureRestore';
  button.className='experimental-restore';
  button.textContent='実験機能を表示';
  button.title='非表示にした実験機能の設定を再表示します';
  button.addEventListener('click',()=>setControllerVisibility(true));
  toolbar.appendChild(button);
}

function restoreRulePresentation(){
  const ruleSection=findRuleSection();
  const title=ruleSection&&ruleSection.querySelector('.section-title');
  if(title)title.textContent='マウスボタンの割り当て';
  const editorTitle=ruleSection&&[...ruleSection.querySelectorAll('.subpanel-title')].find(e=>e.textContent.includes('選択中の割り当て')||e.textContent.includes('マウス割り当て'));
  if(editorTitle)editorTitle.textContent='選択中のマウス割り当てを編集';
  const save=byId('saveRule');
  if(save)save.textContent='選択中のマウス割り当てを保存';
}

function removeControllerUI(){
  ['controllerFeatureSection','joyConSection','joyRuleModeField'].forEach(id=>{const e=byId(id);if(e)e.remove()});
  restoreRulePresentation();
  joyState=null;
  joyDirty=false;
}

function injectVisiblePanels(){
  injectStyle();
  const ruleSection=findRuleSection();
  if(!ruleSection)return;
  if(!byId('controllerFeatureSection')){
    const gate=document.createElement('section');
    gate.id='controllerFeatureSection';
    gate.className='section';
    gate.innerHTML='<div class="section-head"><div><div class="section-title">実験的なゲームコントローラー入力</div><div class="section-help">不要なら機能を停止したうえで、この設定項目そのものを画面から消せます。</div></div></div><div class="section-body"><div class="subpanel controller-gate"><input type="checkbox" id="controllerFeatureEnabled"><div class="controller-gate-copy"><label for="controllerFeatureEnabled"><b>Joy-Con／Raw HID／XInput入力を有効にする</b></label><div class="muted controller-disabled-note" id="controllerFeatureStatus"></div><div class="controller-actions"><button type="button" id="controllerHideUI" class="danger">機能を停止して設定画面から隠す</button></div></div></div></div>';
    ruleSection.parentNode.insertBefore(gate,ruleSection);
    byId('controllerFeatureEnabled').addEventListener('change',toggleControllerFeature);
    byId('controllerHideUI').addEventListener('click',()=>setControllerVisibility(false));
  }
  if(!byId('joyConSection')){
    const panel=document.createElement('section');
    panel.id='joyConSection';
    panel.className='section';
    panel.innerHTML='<div class="section-head"><div><div class="section-title">ゲームコントローラー詳細設定</div><div class="section-help">純正Joy-Con（L）、手動登録したRaw HID、XInput P1〜P4をルールエンジンへ接続します。</div></div></div>'+
    '<div class="section-body joy-grid"><div class="subpanel"><div class="subpanel-title">接続と入力状態</div><div class="statusbox"><div class="main" id="joyStatusText">状態を取得中</div><div class="detail" id="joyErrorText"></div></div><div class="statusbox" style="margin-top:8px"><div class="main" id="xInputStatusText">XInput状態を取得中</div><div class="detail" id="xInputLastText">最後の入力: ―</div></div><div class="joy-metrics" style="margin-top:10px">'+
    '<div class="metric"><div class="k">接続したJoy-Con</div><div class="v" id="joyDeviceText">―</div></div><div class="metric"><div class="k">バッテリー残量</div><div class="v" id="joyBatteryText">―</div></div><div class="metric"><div class="k">最後に検出したJoy-Con入力</div><div class="v" id="joyLastInputText">―</div></div><div class="metric"><div class="k">スティック現在位置</div><div class="v joy-value" id="joyPositionText">X 0.000 / Y 0.000</div></div></div>'+
    '<div class="joy-stick" aria-label="Joy-Conスティック現在位置"><div class="joy-stick-dead" id="joyDeadCircle"></div><div class="joy-stick-dot" id="joyStickDot"></div></div><div class="joy-buttons"><button type="button" id="joyRescan">HID一覧を更新・再接続</button><button type="button" id="joyRecord" class="record">● ゲームコントローラー入力を記録</button><button type="button" id="joyAssignLast">選択中の割り当てへ最後の入力を設定</button></div></div>'+
    '<div class="subpanel"><div class="subpanel-title">選択中のプロファイルのJoy-Con設定</div><div class="muted" id="joyProfileText">―</div><div class="joy-settings" style="margin-top:10px">'+
    '<label class="checkline joy-full"><input type="checkbox" id="joyEnabled"> このプロファイルでJoy-Con（L）入力を使用する</label>'+
    '<div class="field joy-full"><label for="joyCompatibleDevice">互換品を左Joy-Conとして手動登録</label><select id="joyCompatibleDevice" class="hid-list" size="9"><option value="">純正Joy-Conを自動検出（互換品登録なし）</option></select><small>BetterJoyと同様に、Windowsが公開している全HIDインターフェースから対象を明示選択します。VID/PIDだけで推測しません。</small><div class="hid-warning"><b>注意:</b> Steamを完全終了し、対象機器以外を操作しない状態で一覧を更新してください。キーボードやマウスなど別のHIDを選ばないでください。登録した1インターフェースだけを開きます。</div></div>'+
    '<label class="checkline"><input type="checkbox" id="joyReconnect"> 切断後に自動再接続する</label><div class="field"><label for="joyReconnectMs">再検索間隔</label><div class="row"><input id="joyReconnectMs" type="number" min="250" max="10000" step="250"><span>ms</span></div></div>'+
    '<div class="field"><label for="joyDeadZone">デッドゾーン</label><input id="joyDeadZone" type="number" min="0.05" max="0.90" step="0.01"></div><div class="field"><label for="joyReleaseZone">解放判定</label><input id="joyReleaseZone" type="number" min="0.01" max="0.89" step="0.01"><small>デッドゾーンより小さくします。</small></div>'+
    '<div class="field"><label for="joyDirectionMode">方向判定</label><select id="joyDirectionMode"><option value="4">4方向</option><option value="8">8方向（斜め入力）</option></select></div><div class="row"><label class="checkline"><input type="checkbox" id="joyInvertX"> X軸反転</label><label class="checkline"><input type="checkbox" id="joyInvertY"> Y軸反転</label></div>'+
    '<div class="statusbox joy-full"><div class="main">キャリブレーション状態</div><div class="detail" id="joyCalibrationText">未実行</div></div></div><div class="joy-buttons"><button type="button" id="joySave" class="primary large">登録・接続・スティック設定を保存</button><button type="button" id="joyCalStart">キャリブレーションを開始</button><button type="button" id="joyCalFinish">キャリブレーション結果を保存</button><button type="button" id="joyCalCancel">キャリブレーションを中止</button></div></div></div>';
    ruleSection.parentNode.insertBefore(panel,ruleSection);
    bindPanel();
  }
  const title=ruleSection.querySelector('.section-title');
  if(title)title.textContent='マウス・キーボード・ゲームコントローラーの割り当て';
  const editorTitle=[...ruleSection.querySelectorAll('.subpanel-title')].find(e=>e.textContent.includes('マウス割り当て'));
  if(editorTitle)editorTitle.textContent='選択中の割り当てを編集';
  const save=byId('saveRule');
  if(save)save.textContent='選択中の割り当てを保存';
  injectRuleMode();
}

function injectRuleMode(){
  if(byId('joyRuleMode'))return;
  const input=byId('ruleInput');
  const output=byId('ruleOutput');
  if(!input||!output)return;
  const field=document.createElement('div');
  field.id='joyRuleModeField';
  field.className='field';
  field.innerHTML='<label for="joyRuleMode">実行方式</label><select id="joyRuleMode"><option value="Tap">押して離したときに1回実行</option><option value="Hold">押している間、出力キーを保持</option></select><small>HoldはJoy-ConまたはXInputの単独入力からキーボードキーを保持する場合に使用します。</small>';
  output.parentElement.parentNode.insertBefore(field,output.parentElement);
  byId('joyRuleMode').addEventListener('change',()=>{
    if(byId('joyRuleMode').value==='Hold'){
      const longEnabled=byId('ruleLongEnabled');
      if(longEnabled&&longEnabled.checked){longEnabled.checked=false;longEnabled.dispatchEvent(new Event('change',{bubbles:true}))}
    }
  });
}

function bindPanel(){
  const ids=['joyEnabled','joyCompatibleDevice','joyReconnect','joyReconnectMs','joyDeadZone','joyReleaseZone','joyDirectionMode','joyInvertX','joyInvertY'];
  ids.forEach(id=>{const e=byId(id);if(e)e.addEventListener('input',()=>{joyDirty=true})});
  byId('joyRescan').addEventListener('click',()=>joyPost('rescan'));
  byId('joySave').addEventListener('click',()=>joyPost('save-stick'));
  byId('joyCalStart').addEventListener('click',()=>joyPost('calibration-start'));
  byId('joyCalFinish').addEventListener('click',()=>joyPost('calibration-finish'));
  byId('joyCalCancel').addEventListener('click',()=>joyPost('calibration-cancel'));
  byId('joyRecord').addEventListener('click',()=>{const record=byId('recordInput');if(record)record.click()});
  byId('joyAssignLast').addEventListener('click',assignLastInput);
}

function selectedRuleIndex(){
  const rows=[...(byId('ruleRows')?byId('ruleRows').querySelectorAll('tr'):[])];
  return rows.findIndex(row=>row.classList.contains('selected'));
}

function renderRuleMode(state){
  const select=byId('joyRuleMode');
  if(!select||!state||!Array.isArray(state.rules))return;
  const index=selectedRuleIndex();
  const rule=index>=0?state.rules[index]:null;
  select.value=rule&&String(rule.mode).toLowerCase()==='hold'?'Hold':'Tap';
  select.disabled=!state.controllerEnabled||!rule;
}

function renderControllerFeature(state){
  injectRestoreControl();
  if(!state)return;
  const visible=!!state.controllerVisible;
  const restore=byId('experimentalFeatureRestore');
  if(restore)restore.hidden=visible;
  if(!visible){removeControllerUI();return}
  injectVisiblePanels();
  const enabled=!!state.controllerEnabled;
  const toggle=byId('controllerFeatureEnabled');
  if(toggle&&!controllerBusy)toggle.checked=enabled;
  const detail=byId('joyConSection');
  if(detail)detail.hidden=!enabled;
  const status=byId('controllerFeatureStatus');
  if(status)status.textContent=enabled?'有効: Raw HID列挙とXInput監視を実行しています。':'無効: ワーカー・HID列挙・XInput監視・コントローラールール実行を停止中です。';
  const mode=byId('joyRuleModeField');
  if(mode)mode.hidden=!enabled;
  renderRuleMode(state);
}

async function controllerPost(body){
  const response=await originalFetch('/api/controller',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
  let result={};
  try{result=await response.json()}catch(_e){}
  if(!response.ok||result.ok===false)throw new Error(result.error||('HTTP '+response.status));
  if(result.state)mainState=result.state;
  if(result.joyCon)joyState=result.joyCon;
  return result;
}

async function toggleControllerFeature(){
  if(controllerBusy)return;
  const toggle=byId('controllerFeatureEnabled');
  if(!toggle)return;
  controllerBusy++;
  toggle.disabled=true;
  try{
    const result=await controllerPost({op:'set-enabled',enabled:toggle.checked});
    renderControllerFeature(mainState);
    if(mainState&&mainState.controllerEnabled)loadJoyCon();
    setMessageFromResult(result,'コントローラー機能設定を変更しました。');
  }catch(e){
    toggle.checked=!!(mainState&&mainState.controllerEnabled);
    showError(e);
  }finally{controllerBusy--;if(toggle.isConnected)toggle.disabled=false}
}

async function setControllerVisibility(visible){
  if(controllerBusy)return;
  controllerBusy++;
  const hide=byId('controllerHideUI');
  const restore=byId('experimentalFeatureRestore');
  if(hide)hide.disabled=true;
  if(restore)restore.disabled=true;
  try{
    const result=await controllerPost({op:visible?'show-ui':'hide-ui'});
    renderControllerFeature(mainState);
    setMessageFromResult(result,visible?'実験機能を表示しました。':'実験機能を停止して隠しました。');
  }catch(e){showError(e)}finally{
    controllerBusy--;
    if(hide&&hide.isConnected)hide.disabled=false;
    if(restore&&restore.isConnected)restore.disabled=false;
  }
}

function setMessageFromResult(result,fallback){
  const message=byId('message');
  if(message){message.textContent=result.message||fallback;message.classList.remove('error')}
}
function showError(error){
  const message=byId('message');
  if(message){message.textContent=error.message||String(error);message.classList.add('error')}
}

async function loadJoyCon(){
  if(!mainState||!mainState.controllerVisible||!mainState.controllerEnabled)return;
  if(joyBusy)return;
  joyBusy++;
  try{
    const response=await originalFetch('/api/joycon',{cache:'no-store'});
    if(!response.ok)throw new Error('HTTP '+response.status);
    joyState=await response.json();
    renderJoyCon();
  }catch(e){
    const status=byId('joyStatusText');
    if(status)status.textContent='Joy-Con状態を取得できません: '+e.message;
  }finally{joyBusy--}
}

function joyPayload(op){
  return {op:op,enabled:byId('joyEnabled').checked,compatibleDeviceId:byId('joyCompatibleDevice').value,reconnectEnabled:byId('joyReconnect').checked,reconnectMs:Number(byId('joyReconnectMs').value)||1000,deadZone:Number(byId('joyDeadZone').value),releaseZone:Number(byId('joyReleaseZone').value),directionMode:byId('joyDirectionMode').value,invertX:byId('joyInvertX').checked,invertY:byId('joyInvertY').checked};
}

async function joyPost(op){
  if(joyBusy)return;
  joyBusy++;
  try{
    const response=await originalFetch('/api/joycon',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(joyPayload(op))});
    let result={};
    try{result=await response.json()}catch(_e){}
    if(!response.ok||result.ok===false)throw new Error(result.error||('HTTP '+response.status));
    if(result.joyCon)joyState=result.joyCon;
    if(op==='save-stick')joyDirty=false;
    setMessageFromResult(result,'Joy-Con操作を完了しました。');
    renderJoyCon();
  }catch(e){showError(e)}finally{joyBusy--}
}

function hex4(value){return Number(value||0).toString(16).padStart(4,'0')}
function candidateLabel(candidate){
  const kind=(Number(candidate.ControllerType||0)===1)?'Joy-Con(L)候補':(Number(candidate.UsagePage||0)===1&&(Number(candidate.Usage||0)===4||Number(candidate.Usage||0)===5))?'Gamepad/Joystick':'Raw HID';
  const identity=[candidate.Product||'製品名なし',candidate.Manufacturer||'',kind,'VID '+hex4(candidate.VendorId)+' / PID '+hex4(candidate.ProductId),candidate.Serial?('Serial '+candidate.Serial):('ID '+(candidate.Fingerprint||'不明')),'Usage '+hex4(candidate.UsagePage)+':'+hex4(candidate.Usage),'Report IN '+(candidate.InputReportLength||'?')+' / OUT '+(candidate.OutputReportLength||'?')].filter(Boolean);
  if(candidate.InspectError)identity.push('詳細取得失敗');
  return identity.join(' | ');
}

function renderCandidateList(candidates){
  const select=byId('joyCompatibleDevice');
  if(!select)return;
  const selected=joyDirty?select.value:(joyState.compatibleDeviceId||'');
  select.innerHTML='';
  const automatic=document.createElement('option');
  automatic.value='';
  automatic.textContent='純正Joy-Conを自動検出（互換品登録なし）';
  select.appendChild(automatic);
  candidates.forEach(candidate=>{
    if(!candidate.Fingerprint)return;
    const option=document.createElement('option');
    option.value='path:'+String(candidate.Fingerprint).toLowerCase();
    option.textContent=candidateLabel(candidate);
    option.title=option.textContent+(candidate.InspectError?' / '+candidate.InspectError:'');
    select.appendChild(option);
  });
  if(selected&&![...select.options].some(o=>o.value===selected)){
    const saved=joyState.compatibleDevice||{};
    const option=document.createElement('option');
    option.value=selected;
    option.textContent='保存済み・現在未検出 | '+(saved.Product||'HID interface')+' | VID '+hex4(saved.VendorId)+' / PID '+hex4(saved.ProductId)+' | '+(saved.Serial?('Serial '+saved.Serial):('ID '+(saved.Fingerprint||'不明')));
    select.appendChild(option);
  }
  select.value=selected;
}

function renderJoyCon(){
  if(!mainState||!mainState.controllerVisible||!mainState.controllerEnabled||!joyState)return;
  injectVisiblePanels();
  const status=joyState.status||{};
  byId('joyStatusText').textContent='Joy-Con: '+(joyState.statusText||'未接続');
  byId('xInputStatusText').textContent=joyState.xInputStatusText||'XInput未初期化';
  byId('xInputLastText').textContent='最後の入力: '+(joyState.lastControllerText||'―');
  byId('joyErrorText').textContent=status.LastError||'';
  renderCandidateList(Array.isArray(status.Candidates)?status.Candidates:[]);
  const device=status.Device||{};
  byId('joyDeviceText').textContent=[device.Product,device.Manufacturer,device.Serial&&('Serial '+device.Serial),device.Fingerprint&&('ID '+device.Fingerprint)].filter(Boolean).join(' / ')||'―';
  byId('joyBatteryText').textContent=Number(status.BatteryPercent)>=0?(status.BatteryPercent+'%'+(status.Charging?'・充電中':'')):'取得できません';
  byId('joyLastInputText').textContent=joyState.lastInputText||'―';
  const x=Number(status.StickX)||0;
  const y=Number(status.StickY)||0;
  byId('joyPositionText').textContent='X '+x.toFixed(3)+' / Y '+y.toFixed(3)+'  (raw '+(status.RawStickX||0)+', '+(status.RawStickY||0)+')';
  const dot=byId('joyStickDot');
  if(dot){dot.style.left='calc('+(50+x*42)+'% - 9px)';dot.style.top='calc('+(50-y*42)+'% - 9px)'}
  byId('joyProfileText').textContent='保存対象: '+(joyState.profileName||'―');
  byId('joyCalibrationText').textContent=joyState.calibrationText||'未実行';
  byId('joyCalFinish').disabled=!joyState.calibrationActive;
  byId('joyCalCancel').disabled=!joyState.calibrationActive;
  if(!joyDirty){
    byId('joyEnabled').checked=!!joyState.enabled;
    byId('joyReconnect').checked=!!joyState.reconnectEnabled;
    byId('joyReconnectMs').value=joyState.reconnectMs||1000;
    const stick=joyState.stick||{};
    byId('joyDeadZone').value=Number(stick.DeadZone||0.28).toFixed(2);
    byId('joyReleaseZone').value=Number(stick.ReleaseZone||0.20).toFixed(2);
    byId('joyDirectionMode').value=String(stick.DirectionMode||'8');
    byId('joyInvertX').checked=!!stick.InvertX;
    byId('joyInvertY').checked=!!stick.InvertY;
    const dead=Math.max(5,Math.min(90,Number(stick.DeadZone||0.28)*84));
    const circle=byId('joyDeadCircle');
    if(circle){circle.style.width=dead+'%';circle.style.height=dead+'%';circle.style.left=((100-dead)/2)+'%';circle.style.top=((100-dead)/2)+'%'}
  }
}

function assignLastInput(){
  const kind=joyState&&joyState.lastControllerKind;
  const code=joyState&&joyState.lastControllerCode;
  const input=byId('ruleInput');
  if(!kind||!code||!input){showError(new Error('割り当てへ設定できるコントローラー入力がまだありません。'));return}
  if(selectedRuleIndex()<0){showError(new Error('先に割り当てを選択してください。'));return}
  const token=kind==='XInput'?'XInput '+code:'Joy-Con '+code;
  const current=input.value.trim();
  input.value=current?current+' + '+token:token;
  input.dispatchEvent(new Event('input',{bubbles:true}));
  const message=byId('message');
  if(message){message.textContent='最後のコントローラー入力を入力欄へ反映しました。「選択中の割り当てを保存」で確定してください。';message.classList.remove('error')}
}

function start(){
  injectRestoreControl();
  renderControllerFeature(mainState);
  setInterval(()=>{if(mainState&&mainState.controllerVisible&&mainState.controllerEnabled)loadJoyCon()},500);
}
if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',start);else start();
})();`

func (a *App) webJoyConUIJS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, joyConUIJS)
}
