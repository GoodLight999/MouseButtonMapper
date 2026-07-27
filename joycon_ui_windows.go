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
      setTimeout(()=>renderRuleMode(state),0);
      if(state&&Number(state.activeProfile)!==lastEditorProfile){
        lastEditorProfile=Number(state.activeProfile);
        joyDirty=false;
        loadJoyCon();
      }
    }).catch(()=>{});
  }
  return response;
};

function injectStyle(){
  if(byId('joyConStyle'))return;
  const style=document.createElement('style');
  style.id='joyConStyle';
  style.textContent='.joy-grid{display:grid;grid-template-columns:minmax(330px,.9fr) minmax(430px,1.1fr);gap:14px}.joy-metrics{display:grid;grid-template-columns:repeat(2,minmax(150px,1fr));gap:8px}.joy-stick{position:relative;width:190px;height:190px;border:1px solid var(--line);border-radius:50%;background:var(--panel);margin:10px auto}.joy-stick:before,.joy-stick:after{content:"";position:absolute;background:var(--line)}.joy-stick:before{left:50%;top:8%;bottom:8%;width:1px}.joy-stick:after{top:50%;left:8%;right:8%;height:1px}.joy-stick-dead{position:absolute;border:1px dashed var(--warn);border-radius:50%;left:35%;top:35%;width:30%;height:30%}.joy-stick-dot{position:absolute;width:18px;height:18px;border-radius:50%;background:var(--accent);left:calc(50% - 9px);top:calc(50% - 9px);transition:left .04s linear,top .04s linear}.joy-settings{display:grid;grid-template-columns:1fr 1fr;gap:10px}.joy-full{grid-column:1/-1}.joy-value{font-family:Consolas,monospace}.joy-buttons{display:flex;gap:8px;flex-wrap:wrap;margin-top:12px}@media(max-width:900px){.joy-grid{grid-template-columns:1fr}}';
  document.head.appendChild(style);
}

function injectPanel(){
  if(byId('joyConSection'))return;
  injectStyle();
  const sections=[...document.querySelectorAll('section.section')];
  const ruleSection=sections.find(s=>{const t=s.querySelector('.section-title');return t&&t.textContent.includes('マウスボタンの割り当て')});
  if(!ruleSection)return;
  const panel=document.createElement('section');
  panel.id='joyConSection';
  panel.className='section';
  panel.innerHTML='<div class="section-head"><div><div class="section-title">Joy-Con（L）</div><div class="section-help">選択中のプロファイルへJoy-Con設定を保存し、既存のマウス・キーボード割り当てと同じルールで使用します。</div></div></div>'+
  '<div class="section-body joy-grid"><div class="subpanel"><div class="subpanel-title">接続と入力状態</div><div class="statusbox"><div class="main" id="joyStatusText">状態を取得中</div><div class="detail" id="joyErrorText"></div></div><div class="joy-metrics" style="margin-top:10px">'+
  '<div class="metric"><div class="k">検出したJoy-Con</div><div class="v" id="joyDeviceText">―</div></div><div class="metric"><div class="k">バッテリー残量</div><div class="v" id="joyBatteryText">―</div></div><div class="metric"><div class="k">最後に検出したJoy-Con入力</div><div class="v" id="joyLastInputText">―</div></div><div class="metric"><div class="k">スティック現在位置</div><div class="v joy-value" id="joyPositionText">X 0.000 / Y 0.000</div></div></div>'+
  '<div class="joy-stick" aria-label="Joy-Conスティック現在位置"><div class="joy-stick-dead" id="joyDeadCircle"></div><div class="joy-stick-dot" id="joyStickDot"></div></div><div class="joy-buttons"><button type="button" id="joyRescan">Joy-Conを接続・再検索</button><button type="button" id="joyRecord" class="record">● Joy-Con入力を記録</button><button type="button" id="joyAssignLast">選択中の割り当てへJoy-Con入力を設定</button></div></div>'+
  '<div class="subpanel"><div class="subpanel-title">選択中のプロファイルのJoy-Con設定</div><div class="muted" id="joyProfileText">―</div><div class="joy-settings" style="margin-top:10px">'+
  '<label class="checkline joy-full"><input type="checkbox" id="joyEnabled"> このプロファイルでJoy-Con（L）を使用する</label><div class="field joy-full"><label for="joyPreferredDevice">優先するJoy-Con識別子</label><input id="joyPreferredDevice" type="text" placeholder="空欄なら最初に検出したJoy-Con（L）"><small>通常は空欄で構いません。複数台を区別する場合だけ指定します。</small></div>'+
  '<label class="checkline"><input type="checkbox" id="joyReconnect"> 切断後に自動再接続する</label><div class="field"><label for="joyReconnectMs">再検索間隔</label><div class="row"><input id="joyReconnectMs" type="number" min="250" max="10000" step="250"><span>ms</span></div></div>'+
  '<div class="field"><label for="joyDeadZone">デッドゾーン</label><input id="joyDeadZone" type="number" min="0.05" max="0.90" step="0.01"></div><div class="field"><label for="joyReleaseZone">解放判定</label><input id="joyReleaseZone" type="number" min="0.01" max="0.89" step="0.01"><small>デッドゾーンより小さくします。</small></div>'+
  '<div class="field"><label for="joyDirectionMode">方向判定</label><select id="joyDirectionMode"><option value="4">4方向</option><option value="8">8方向（斜め入力）</option></select></div><div class="row"><label class="checkline"><input type="checkbox" id="joyInvertX"> X軸反転</label><label class="checkline"><input type="checkbox" id="joyInvertY"> Y軸反転</label></div>'+
  '<div class="statusbox joy-full"><div class="main">キャリブレーション状態</div><div class="detail" id="joyCalibrationText">未実行</div></div></div><div class="joy-buttons"><button type="button" id="joySave" class="primary large">Joy-Con接続・スティック設定を保存</button><button type="button" id="joyCalStart">キャリブレーションを開始</button><button type="button" id="joyCalFinish">キャリブレーション結果を保存</button><button type="button" id="joyCalCancel">キャリブレーションを中止</button></div></div></div>';
  ruleSection.parentNode.insertBefore(panel,ruleSection);

  const title=ruleSection.querySelector('.section-title');
  if(title)title.textContent='マウス・キーボード・Joy-Conの割り当て';
  const editorTitle=[...ruleSection.querySelectorAll('.subpanel-title')].find(e=>e.textContent.includes('マウス割り当て'));
  if(editorTitle)editorTitle.textContent='選択中の割り当てを編集';
  const save=byId('saveRule');
  if(save)save.textContent='選択中の割り当てを保存';

  injectRuleMode();
  bindPanel();
}

function injectRuleMode(){
  if(byId('joyRuleMode'))return;
  const input=byId('ruleInput');
  const output=byId('ruleOutput');
  if(!input||!output)return;
  const field=document.createElement('div');
  field.className='field';
  field.innerHTML='<label for="joyRuleMode">実行方式</label><select id="joyRuleMode"><option value="Tap">押して離したときに1回実行</option><option value="Hold">押している間、出力キーを保持</option></select><small>HoldはJoy-Con単独入力からキーボードキーを保持する場合に使用します。</small>';
  output.parentElement.parentNode.insertBefore(field,output.parentElement);
  byId('joyRuleMode').addEventListener('change',()=>{
    if(byId('joyRuleMode').value==='Hold'){
      const longEnabled=byId('ruleLongEnabled');
      if(longEnabled&&longEnabled.checked){longEnabled.checked=false;longEnabled.dispatchEvent(new Event('change',{bubbles:true}))}
    }
  });
}

function bindPanel(){
  const ids=['joyEnabled','joyPreferredDevice','joyReconnect','joyReconnectMs','joyDeadZone','joyReleaseZone','joyDirectionMode','joyInvertX','joyInvertY'];
  ids.forEach(id=>{const e=byId(id);if(e)e.addEventListener('input',()=>{joyDirty=true})});
  byId('joyRescan').addEventListener('click',()=>joyPost('rescan'));
  byId('joySave').addEventListener('click',()=>joyPost('save-stick'));
  byId('joyCalStart').addEventListener('click',()=>joyPost('calibration-start'));
  byId('joyCalFinish').addEventListener('click',()=>joyPost('calibration-finish'));
  byId('joyCalCancel').addEventListener('click',()=>joyPost('calibration-cancel'));
  byId('joyRecord').addEventListener('click',()=>{
    const record=byId('recordInput');
    if(record)record.click();
  });
  byId('joyAssignLast').addEventListener('click',assignLastInput);
}

function selectedRuleIndex(){
  const rows=[...(byId('ruleRows')?byId('ruleRows').querySelectorAll('tr'):[])];
  return rows.findIndex(row=>row.classList.contains('selected'));
}

function renderRuleMode(state){
  injectPanel();
  injectRuleMode();
  const select=byId('joyRuleMode');
  if(!select||!state||!Array.isArray(state.rules))return;
  const index=selectedRuleIndex();
  const rule=index>=0?state.rules[index]:null;
  select.value=rule&&String(rule.mode).toLowerCase()==='hold'?'Hold':'Tap';
  const hasRule=!!rule;
  select.disabled=!hasRule;
}

async function loadJoyCon(){
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
  return {op:op,enabled:byId('joyEnabled').checked,preferredDevice:byId('joyPreferredDevice').value,reconnectEnabled:byId('joyReconnect').checked,reconnectMs:Number(byId('joyReconnectMs').value)||1000,deadZone:Number(byId('joyDeadZone').value),releaseZone:Number(byId('joyReleaseZone').value),directionMode:byId('joyDirectionMode').value,invertX:byId('joyInvertX').checked,invertY:byId('joyInvertY').checked};
}

async function joyPost(op){
  joyBusy++;
  try{
    const response=await originalFetch('/api/joycon',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(joyPayload(op))});
    let result={};
    try{result=await response.json()}catch(_e){}
    if(!response.ok||result.ok===false)throw new Error(result.error||('HTTP '+response.status));
    if(result.joyCon)joyState=result.joyCon;
    if(op==='save-stick')joyDirty=false;
    const message=byId('message');
    if(message){message.textContent=result.message||'Joy-Con操作を完了しました。';message.classList.remove('error')}
    renderJoyCon();
  }catch(e){
    const message=byId('message');
    if(message){message.textContent=e.message;message.classList.add('error')}
  }finally{joyBusy--}
}

function renderJoyCon(){
  injectPanel();
  if(!joyState)return;
  const status=joyState.status||{};
  byId('joyStatusText').textContent=joyState.statusText||'未接続';
  byId('joyErrorText').textContent=status.LastError||'';
  const device=status.Device||{};
  byId('joyDeviceText').textContent=[device.Product,device.Serial&&('Serial '+device.Serial),device.Fingerprint&&('ID '+device.Fingerprint)].filter(Boolean).join(' / ')||'―';
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
    byId('joyPreferredDevice').value=joyState.preferredDevice||'';
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
  const code=joyState&&joyState.status&&joyState.status.LastInput;
  const input=byId('ruleInput');
  if(!code||!input){
    const message=byId('message');
    if(message){message.textContent='割り当てへ設定できるJoy-Con入力がまだありません。Joy-Conを操作してから実行してください。';message.classList.add('error')}
    return;
  }
  if(selectedRuleIndex()<0){
    const message=byId('message');
    if(message){message.textContent='先に割り当てを選択してください。';message.classList.add('error')}
    return;
  }
  const token='Joy-Con '+code;
  const current=input.value.trim();
  input.value=current?current+' + '+token:token;
  input.dispatchEvent(new Event('input',{bubbles:true}));
  const message=byId('message');
  if(message){message.textContent='最後のJoy-Con入力を入力欄へ反映しました。「選択中の割り当てを保存」で確定してください。';message.classList.remove('error')}
}

function start(){
  injectPanel();
  loadJoyCon();
  setInterval(loadJoyCon,300);
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
