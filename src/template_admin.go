package main

import (
	"html/template"
	"net/url"
)

func adminText(ja, en string) string {
	if currentLang() == "en" {
		return en
	}
	return ja
}

var adminTemplate = template.Must(template.New("admin").Funcs(template.FuncMap{
	"urlquery": url.QueryEscape,
	"t":        t,
	"tf":       tf,
	"lang":     currentLang,
	"x":        adminText,
}).Parse(`<!doctype html>
<html lang="{{lang}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="light">
<title>{{t "app.title"}}</title>
<style>
:root{--text:#222;--sub:#6b6b6b;--line:#e9e9e7;--soft:#f7f7f5;--danger:#b42318;--ok:#2b7a3d}
*{box-sizing:border-box}body{margin:0;color:var(--text);background:#fff;font-family:"Segoe UI","Yu Gothic UI",sans-serif}.wrap{max-width:1040px;margin:auto;padding:42px 28px 72px}
.top{display:flex;justify-content:space-between;gap:20px;align-items:flex-start;padding-bottom:24px;border-bottom:1px solid var(--line)}h1{font-size:34px;margin:0 0 8px;letter-spacing:-.03em}.lead{margin:0;color:var(--sub);font-size:15px;line-height:1.7}.topActions{display:flex;align-items:flex-end;gap:10px;flex-wrap:wrap}
.languageControl{display:flex;flex-direction:column;gap:4px;font-size:11px;color:var(--sub);font-weight:650}.languageControl select{border:1px solid #d7d7d7;border-radius:7px;background:#fff;padding:7px 28px 7px 9px;color:var(--text)}
.badge{font-size:12px;font-weight:700;background:var(--soft);padding:7px 9px;border-radius:7px;color:#555}.btn,.fileLabel{display:inline-flex;align-items:center;justify-content:center;border:1px solid #d7d7d7;background:#fff;color:#222;border-radius:7px;padding:10px 14px;font-size:14px;font-weight:650;cursor:pointer;text-decoration:none}.btn:disabled{opacity:.55;cursor:default}.primary{background:#222;color:#fff;border-color:#222}.danger{color:var(--danger);border-color:#e5c4c0}.fileLabel input{display:none}
.testHint,.notice{margin-top:14px;padding:12px 14px;border-radius:8px;font-size:14px}.testHint{background:#fff7e6;border:1px solid #efcf8c}.notice{background:var(--soft);color:#555}
.wizardNav{display:grid;grid-template-columns:repeat(4,1fr);gap:8px;margin:24px 0 28px}.wizardNavItem{display:flex;align-items:center;gap:8px;padding:10px 12px;border:1px solid var(--line);border-radius:8px;color:var(--sub);font-size:12px;font-weight:700}.dot{width:24px;height:24px;display:grid;place-items:center;flex:0 0 24px;border-radius:50%;background:var(--soft)}.wizardNavItem.active{border-color:#222;color:#222}.wizardNavItem.active .dot{background:#222;color:#fff}.wizardNavItem.done .dot{background:#eaf6ed;color:#247238}
.wizardStep{display:none}.wizardStep.active{display:block}.wizardCard{max-width:800px;margin:auto;border:1px solid var(--line);border-radius:12px;padding:28px}.wizardCard.wide{max-width:none}.wizardHead{display:flex;gap:14px;align-items:flex-start;margin-bottom:24px}.wizardNo{width:42px;height:42px;display:grid;place-items:center;flex:0 0 42px;border-radius:8px;background:#222;color:#fff;font-size:20px;font-weight:750}.wizardCard h2{font-size:25px;margin:3px 0 7px}.wizardCard p{margin:0;color:var(--sub);font-size:14px;line-height:1.7}.wizardBody{margin-top:20px}.wizardActions{display:flex;justify-content:space-between;align-items:center;gap:12px;margin-top:26px}.rightActions,.actions{display:flex;gap:9px;flex-wrap:wrap}.rightActions{margin-left:auto}
.qrGrid{display:grid;grid-template-columns:220px 1fr;gap:28px;align-items:start}.qr{width:220px;height:220px;border:1px solid #ddd;padding:7px}.label{font-size:12px;color:var(--sub);margin-bottom:4px}.value{font-size:17px;font-weight:650;overflow-wrap:anywhere}.mono{font-family:Consolas,"SFMono-Regular",monospace}.muted{color:var(--sub);font-size:13px;line-height:1.6}.url{font-size:12px;overflow-wrap:anywhere;color:#555}.details{margin-top:12px}.details summary{cursor:pointer;color:#555;font-size:13px}
.waiting{display:flex;align-items:center;gap:9px;margin-top:20px;padding:12px 14px;border-radius:8px;background:var(--soft);color:#555;font-size:14px}.waitingDot{width:9px;height:9px;border-radius:50%;background:#777;animation:pulse 1.2s infinite}.waiting.connected{background:#eef8f0;color:#246b35}.waiting.connected .waitingDot{background:var(--ok);animation:none}@keyframes pulse{0%,100%{opacity:.3}50%{opacity:1}}
.panels{display:grid;grid-template-columns:1fr 1fr;gap:26px;margin-top:8px}.panel{min-width:0;border:1px solid var(--line);border-radius:10px;padding:20px}.panel h3{font-size:18px;margin:0 0 5px}.file{display:flex;gap:12px;padding:10px 0;border-bottom:1px solid var(--line)}.file span:first-child{flex:1;min-width:0;overflow-wrap:anywhere}.panelActions{display:flex;gap:9px;flex-wrap:wrap;margin-top:14px;padding-top:14px;border-top:1px solid var(--line)}.status{font-size:14px;color:var(--sub);margin-top:10px}.ok{color:var(--ok);font-weight:650}
.doneIcon{width:58px;height:58px;display:grid;place-items:center;border-radius:50%;background:#eaf6ed;color:#247238;font-size:30px;font-weight:800;margin-bottom:18px}
.modalBackdrop{display:none;position:fixed;inset:0;z-index:60;background:rgba(0,0,0,.45);padding:20px;align-items:center;justify-content:center}.modalBackdrop.open{display:flex}.modal{width:min(470px,100%);background:#fff;border-radius:12px;padding:24px;box-shadow:0 18px 55px rgba(0,0,0,.2)}.modal h2{font-size:21px;margin:0 0 10px}.modal p{color:var(--sub);font-size:14px;line-height:1.7}.modalActions{display:flex;gap:9px;justify-content:flex-end;margin-top:22px}
@media(max-width:760px){.wrap{padding:28px 18px 52px}.top{flex-direction:column}.wizardNav{grid-template-columns:1fr 1fr}.qrGrid,.panels{grid-template-columns:1fr}.qr{width:210px;height:210px}.wizardActions{align-items:stretch;flex-direction:column}.rightActions{width:100%;margin-left:0}.rightActions .btn,.wizardActions>.btn{width:100%}}
@media(max-width:520px){.wizardNav{grid-template-columns:1fr}.wizardCard{padding:20px}.panelActions{display:grid;grid-template-columns:1fr}.panelActions .btn,.panelActions .fileLabel{width:100%}.modalActions{flex-direction:column-reverse}.modalActions .btn{width:100%}}
</style>
</head>
<body>
<main class="wrap">
<div class="top"><div><h1>{{t "app.title"}}</h1><p class="lead">{{tf "app.deviceLead" .DeviceName}}</p></div><div class="topActions">{{if .TestMode}}<div class="badge">{{t "app.testMode"}}</div>{{end}}<button id="globalStop" class="btn danger" onclick="stopApp()">{{t "finish.button"}}</button><label class="languageControl"><span>Language / 言語</span><select id="languageSelect" data-current="{{lang}}"><option value="ja" {{if eq (lang) "ja"}}selected{{end}}>日本語</option><option value="en" {{if eq (lang) "en"}}selected{{end}}>English</option></select></label></div></div>
{{if .TestMode}}<div class="testHint">{{t "app.testHint"}}</div>{{end}}
<div class="wizardNav"><div class="wizardNavItem" data-nav="1"><span class="dot">1</span><span>{{x "Wi-Fi接続" "Wi-Fi"}}</span></div><div class="wizardNavItem" data-nav="2"><span class="dot">2</span><span>{{x "スマホで開く" "Open on phone"}}</span></div><div class="wizardNavItem" data-nav="3"><span class="dot">3</span><span>{{x "ファイル転送" "Transfer files"}}</span></div><div class="wizardNavItem" data-nav="4"><span class="dot">4</span><span>{{x "完了" "Done"}}</span></div></div>

<section class="wizardStep" data-step="1"><div class="wizardCard"><div class="wizardHead"><div class="wizardNo">1</div><div><h2>{{x "専用Wi-Fiに接続する" "Connect to the dedicated Wi-Fi"}}</h2><p>{{if .TestMode}}{{t "step1.testDescription"}}{{else}}{{x "スマートフォンをこのPCが作成した専用Wi-Fiに接続してください。" "Connect your smartphone to the dedicated Wi-Fi created by this PC."}}{{end}}</p></div></div><div class="wizardBody">{{if .WiFiQR}}<div class="qrGrid"><img class="qr" src="/qr.svg?text={{urlquery .WiFiQR}}" alt="Wi-Fi QR"><div><div class="label">{{t "wifi.name"}}</div><div class="value">{{.WiFiSSID}}</div><div style="height:16px"></div><div class="label">{{t "wifi.password"}}</div><div class="value mono">{{.WiFiPassword}}</div><div class="muted" style="margin-top:12px">{{t "wifi.noInternet"}}</div></div></div>{{else}}<div class="notice">{{t "wifi.notUsed"}}</div>{{end}}</div><div class="wizardActions"><span></span><div class="rightActions"><button class="btn primary" onclick="goToStep(2)">{{x "接続したので次へ" "Connected — Next"}} →</button></div></div></div></section>

<section class="wizardStep" data-step="2"><div class="wizardCard"><div class="wizardHead"><div class="wizardNo">2</div><div><h2>{{x "スマホで転送ページを開く" "Open the transfer page on your phone"}}</h2><p>{{if .TestMode}}{{t "step2.testDescription"}}{{else}}{{t "step2.description"}}{{end}}</p></div></div><div class="wizardBody">{{range $i,$u := .URLs}}{{if eq $i 0}}<div class="qrGrid"><img class="qr" src="/qr.svg?text={{urlquery $u.URL}}" alt="Transfer QR"><div><div class="label">{{t "transfer.page"}}</div><div class="url">{{$u.URL}}</div><details class="details"><summary>{{t "transfer.connectionInfo"}}</summary><div class="muted" style="margin-top:8px">{{$u.Interface}} / {{$u.IP}}</div></details></div></div>{{end}}{{else}}<div class="notice">{{t "transfer.networkMissing"}}</div>{{end}}<div id="phoneWaiting" class="waiting"><span class="waitingDot"></span><span id="phoneWaitingText">{{x "スマートフォンからの接続を待っています…" "Waiting for your smartphone to connect…"}}</span></div></div><div class="wizardActions"><button class="btn" onclick="goToStep(1)">← {{x "戻る" "Back"}}</button><div class="rightActions"><button class="btn" onclick="markPhoneConnected()">{{x "スマホで開けたので次へ" "Opened on phone — Next"}} →</button></div></div></div></section>

<section class="wizardStep" data-step="3"><div class="wizardCard wide"><div class="wizardHead"><div class="wizardNo">3</div><div><h2>{{x "ファイルを転送する" "Transfer files"}}</h2><p>{{t "step3.description"}}</p></div></div><div class="panels"><div class="panel"><h3>{{t "admin.sendHeading"}}</h3><div class="muted">{{t "admin.sendDescription"}}</div><div id="sendList"></div><div class="panelActions"><label class="fileLabel primary">{{x "＋ ファイルを追加" "+ Add files"}}<input id="sharePick" type="file" multiple></label><button class="btn" onclick="openFolder('send')">{{x "フォルダーを開く" "Open folder"}}</button><button class="btn danger" onclick="clearSide('send')">{{t "admin.clearSend"}}</button></div><div id="shareStatus" class="status"></div></div><div class="panel"><h3>{{t "admin.receiveHeading"}}</h3><div class="muted">{{t "admin.receiveDescription"}}</div><div id="recvList"></div><div class="panelActions"><button class="btn" onclick="openFolder('receive')">{{x "フォルダーを開く" "Open folder"}}</button><button class="btn danger" onclick="clearSide('receive')">{{t "admin.clearReceive"}}</button></div></div></div><div class="wizardActions"><button class="btn" onclick="goToStep(2)">← {{x "接続方法を確認" "Check connection"}}</button><div class="rightActions"><button class="btn primary" onclick="stopApp()">{{t "finish.button"}} →</button></div></div></div></section>

<section class="wizardStep" data-step="4"><div class="wizardCard"><div class="doneIcon">✓</div><h2>{{x "ファイル転送を終了しました" "File transfer ended"}}</h2><p id="doneBody">{{x "専用Wi-Fiも停止しました。スマートフォンは普段使っているWi-Fiやモバイル通信に戻してください。" "The dedicated Wi-Fi has also stopped. Reconnect your phone to its usual Wi-Fi or mobile data."}}</p></div></section>
</main>

<div id="idleModal" class="modalBackdrop"><div class="modal"><h2>{{x "まもなく自動終了します" "This session will end soon"}}</h2><p>{{x "しばらく操作がなかったため、この転送セッションはまもなく終了します。" "There has been no activity for a while, so this transfer session will end soon."}}</p><div class="modalActions"><button class="btn danger" onclick="stopApp()">{{x "今すぐ終了" "End now"}}</button><button class="btn primary" onclick="continueSession()">{{x "利用を続ける" "Continue using"}}</button></div></div></div>
<div id="absoluteModal" class="modalBackdrop"><div class="modal"><h2>{{x "まもなく利用可能時間の上限です" "Maximum session time is approaching"}}</h2><p>{{x "この上限は延長できません。必要な転送を完了してください。" "This limit cannot be extended. Complete any remaining transfers."}}</p><div class="modalActions"><button class="btn" onclick="dismissAbsoluteWarning()">OK</button><button class="btn danger" onclick="stopApp()">{{x "今すぐ終了" "End now"}}</button></div></div></div>

<script>
const admin='/admin/{{.AdminToken}}';
const currentLocale=document.getElementById('languageSelect').dataset.current;
const msg={
 phoneConnected:'{{x "スマートフォンが接続されました" "Smartphone connected"}}',
 waiting:'{{x "スマートフォンからの接続を待っています…" "Waiting for your smartphone to connect…"}}',
 emptySend:'{{x "スマホに渡すファイルはまだありません。" "No files are ready for the phone yet."}}',
 emptyReceive:'{{x "スマホから受け取ったファイルはまだありません。" "No files have been received from the phone yet."}}',
 preparing:'{{x "準備中: " "Preparing: "}}',
 shareReady:'{{x "スマホから受け取れる状態になりました。" "Ready for the phone to receive."}}',
 clearSend:'{{x "スマホに渡すファイルの一覧を消去しますか？" "Clear the files for the phone?"}}',
 clearReceive:'{{x "スマホから受け取ったファイルの一覧を消去しますか？" "Clear received files?"}}',
 finish:'{{x "ファイル転送を終了しますか？" "End the file transfer session?"}}',
 ended:'{{x "終了済み" "Ended"}}',
 timeout:'{{x "利用時間が終了しました。" "The session time has ended."}}'
};
let currentStep=Number(sessionStorage.getItem('oftWizardStep')||1);if(currentStep<1||currentStep>3)currentStep=1;
let sessionEnded=false,phoneTransitionPending=false,lastIdle=null,step2EnteredAtIdle=null,idleModalVisible=false,absoluteWarningDismissed=false;

document.getElementById('languageSelect').addEventListener('change',async e=>{const old=e.target.dataset.current;try{const body=new URLSearchParams({locale:e.target.value});const r=await fetch(admin+'/locale',{method:'POST',headers:{'Content-Type':'application/x-www-form-urlencoded'},body});if(!r.ok)throw 0;location.reload()}catch(_){e.target.value=old;alert('Language setting could not be saved. / 言語設定を保存できませんでした。')}});

function resetWaiting(){const w=document.getElementById('phoneWaiting');w.classList.remove('connected');document.getElementById('phoneWaitingText').textContent=msg.waiting}
function goToStep(step){if(sessionEnded)step=4;currentStep=step;if(step<4)sessionStorage.setItem('oftWizardStep',String(step));if(step===2){phoneTransitionPending=false;step2EnteredAtIdle=lastIdle;resetWaiting()}document.querySelectorAll('.wizardStep').forEach(e=>e.classList.toggle('active',Number(e.dataset.step)===step));document.querySelectorAll('.wizardNavItem').forEach(e=>{const n=Number(e.dataset.nav);e.classList.toggle('active',n===step);e.classList.toggle('done',n<step)});const stop=document.getElementById('globalStop');stop.disabled=step===4;if(step===4)stop.textContent=msg.ended;window.scrollTo({top:0,behavior:'smooth'})}
function markPhoneConnected(){if(phoneTransitionPending||sessionEnded)return;phoneTransitionPending=true;const w=document.getElementById('phoneWaiting');w.classList.add('connected');document.getElementById('phoneWaitingText').textContent=msg.phoneConnected;setTimeout(()=>{phoneTransitionPending=false;goToStep(3)},500)}
function fmt(n){if(n<1024)return n+' B';if(n<1048576)return(n/1024).toFixed(1)+' KB';if(n<1073741824)return(n/1048576).toFixed(1)+' MB';return(n/1073741824).toFixed(2)+' GB'}
function render(id,files,empty){const el=document.getElementById(id);el.innerHTML='';if(!files.length){const d=document.createElement('div');d.className='muted';d.style.padding='12px 0';d.textContent=empty;el.appendChild(d);return}for(const f of files){const d=document.createElement('div');d.className='file';const n=document.createElement('span');n.textContent=f.name;const z=document.createElement('span');z.className='muted';z.textContent=fmt(f.size);d.append(n,z);el.appendChild(d)}}

document.getElementById('sharePick').addEventListener('change',async e=>{const st=document.getElementById('shareStatus');for(const f of e.target.files){st.textContent=msg.preparing+f.name;await new Promise(res=>{const x=new XMLHttpRequest();x.open('POST',admin+'/upload?name='+encodeURIComponent(f.name));x.onload=res;x.onerror=res;x.send(f)})}st.textContent=msg.shareReady;st.className='status ok';e.target.value='';loadStatus()});
async function openFolder(which){await fetch(admin+'/open?which='+which,{method:'POST'})}
async function clearSide(which){if(!confirm(which==='send'?msg.clearSend:msg.clearReceive))return;await fetch(admin+'/clear-'+which,{method:'POST'});loadStatus()}
async function continueSession(){try{const body=new URLSearchParams({locale:currentLocale});const r=await fetch(admin+'/locale',{method:'POST',headers:{'Content-Type':'application/x-www-form-urlencoded'},body});if(!r.ok)throw 0;document.getElementById('idleModal').classList.remove('open');idleModalVisible=false;await loadStatus()}catch(_){}}
function dismissAbsoluteWarning(){absoluteWarningDismissed=true;document.getElementById('absoluteModal').classList.remove('open')}
function finishLocally(text){sessionEnded=true;document.getElementById('idleModal').classList.remove('open');document.getElementById('absoluteModal').classList.remove('open');document.getElementById('doneBody').textContent=text;sessionStorage.setItem('oftWizardStep','4');goToStep(4)}
async function stopApp(){if(sessionEnded)return;if(!confirm(msg.finish))return;try{await fetch(admin+'/stop',{method:'POST'})}catch(_){}finishLocally('{{x "専用Wi-Fiも停止しました。スマートフォンは普段の通信に戻してください。" "The dedicated Wi-Fi has stopped. Reconnect your phone to its usual network."}}')}
async function loadStatus(){if(sessionEnded)return;try{const r=await fetch(admin+'/api/status',{cache:'no-store'});if(!r.ok)throw 0;const s=await r.json();render('sendList',s.send||[],msg.emptySend);render('recvList',s.receive||[],msg.emptyReceive);const idle=Number(s.idleRemainingSec||0),absolute=Number(s.absoluteRemainingSec||0);if(currentStep===2&&!phoneTransitionPending&&(s.phoneConnected===true||(lastIdle!==null&&idle>lastIdle+1)||(step2EnteredAtIdle!==null&&idle>step2EnteredAtIdle+1)))markPhoneConnected();lastIdle=idle;if(idle<=0||absolute<=0){finishLocally(msg.timeout);return}if(absolute<=300&&!absoluteWarningDismissed)document.getElementById('absoluteModal').classList.add('open');else if(idle<=300&&!idleModalVisible){idleModalVisible=true;document.getElementById('idleModal').classList.add('open')}}catch(_){}}

goToStep(currentStep);loadStatus();setInterval(loadStatus,2000);
</script>
</body>
</html>`))
