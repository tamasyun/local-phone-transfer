package main

import "html/template"

func phoneText(ja, en string) string {
	if currentLang() == "en" {
		return en
	}
	return ja
}

var phoneTemplate = template.Must(template.New("phone").Funcs(template.FuncMap{
	"t":    t,
	"tf":   tf,
	"lang": currentLang,
	"x":    phoneText,
}).Parse(`<!doctype html>
<html lang="{{lang}}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover"><meta name="color-scheme" content="light"><title>{{t "phone.title"}}</title>
<style>
:root{--bg:#fff;--text:#222;--sub:#6b6b6b;--line:#e8e8e8;--soft:#f7f7f5;--accent:#2f6fdf;--ok:#2b7a3d;--warn:#8a6500;--danger:#b42318}*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Yu Gothic UI",sans-serif}.wrap{max-width:720px;margin:0 auto;padding:36px 20px 64px}.title{font-size:30px;font-weight:750;letter-spacing:-.02em;margin:0 0 8px}.lead{color:var(--sub);font-size:15px;line-height:1.7;margin:0}.notice{margin-top:22px;padding:12px 14px;background:var(--soft);border-radius:8px;color:#555;font-size:14px;line-height:1.65}.section{padding:28px 0;border-top:1px solid var(--line)}.section:first-of-type{margin-top:26px}.head{display:flex;gap:12px;align-items:flex-start;margin-bottom:18px;min-width:0}.head>div:last-child{min-width:0}.num{font-size:13px;color:var(--sub);font-weight:700;padding-top:4px;flex:0 0 auto}.section h2{font-size:21px;margin:0 0 4px}.section p{margin:0;color:var(--sub);font-size:14px;line-height:1.65}.button,.fileLabel{display:inline-flex;align-items:center;justify-content:center;min-height:44px;border:1px solid #d8d8d8;background:#fff;color:#222;border-radius:7px;padding:11px 15px;font-size:15px;font-weight:650;cursor:pointer;text-decoration:none}.button:disabled{opacity:.55;cursor:default}.primary{background:#222;color:#fff;border-color:#222}.fileLabel input{display:none}.actions{display:flex;flex-wrap:wrap;gap:10px}.progress{display:none;margin-top:14px}.bar{height:5px;background:#ededed;border-radius:99px;overflow:hidden}.bar>div{height:100%;width:0;background:var(--accent)}.status{margin-top:8px;color:var(--sub);font-size:14px;line-height:1.55}.ok{color:var(--ok);font-weight:650}.files{margin-top:12px}.row{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:12px;align-items:center;padding:14px 0;border-bottom:1px solid var(--line)}.row:last-child{border-bottom:0}.fileInfo{min-width:0}.name{font-size:15px;font-weight:650;line-height:1.45;overflow-wrap:anywhere}.size{font-size:12px;color:var(--sub);margin-top:4px}.fileAction{min-width:112px}.fileStatus{grid-column:1/-1;padding:9px 11px;border-radius:7px;background:var(--soft);font-size:13px;line-height:1.55;color:#555}.fileStatus.ok{background:#edf7ef;color:var(--ok)}.fileStatus.error{background:#fff1ef;color:var(--danger)}.empty{color:var(--sub);font-size:14px;padding:8px 0}.foot{border-top:1px solid var(--line);padding-top:18px;color:#999;font-size:12px;line-height:1.6}@media(max-width:540px){.wrap{padding:28px 16px 48px}.title{font-size:26px}.section h2{font-size:20px}.actions{display:block}.actions>*{width:100%}.actions>*+*{margin-top:8px}.row{grid-template-columns:minmax(0,1fr);gap:10px;padding:16px 0}.fileAction{width:100%;min-width:0}.fileStatus{grid-column:1}.name{font-size:16px}.button,.fileLabel{font-size:16px}}
</style></head><body><main class="wrap">
<h1 class="title">{{t "phone.title"}}</h1><p class="lead">{{tf "phone.lead" .DeviceName}}</p>
<div class="notice">{{t "phone.notice"}}</div>
<section class="section"><div class="head"><div class="num">01</div><div><h2>{{t "phone.sendTitle"}}</h2><p>{{tf "phone.sendDescription" .MaxMB}}</p></div></div><div class="actions"><label class="fileLabel primary">{{t "phone.pick"}}<input id="pick" type="file" multiple></label></div><div id="progress" class="progress"><div id="pname" class="status"></div><div class="bar"><div id="pbar"></div></div><div id="ptext" class="status"></div></div><div id="result" class="status"></div></section>
<section class="section"><div class="head"><div class="num">02</div><div><h2>{{t "phone.receiveTitle"}}</h2><p>{{t "phone.receiveDescription"}}</p></div></div><div class="actions"><button id="refreshButton" class="button" type="button">{{t "phone.refresh"}}</button></div><div id="files" class="files"><div class="empty">{{t "phone.checking"}}</div></div></section>
<div class="foot">{{tf "phone.footer" .DeviceName}}</div>
</main><script>
const base='/s/{{.Token}}';
const pick=document.getElementById('pick');
const refreshButton=document.getElementById('refreshButton');
const msg={uploading:{{t "phone.uploading"}},preparing:{{t "phone.preparing"}},sent:{{t "phone.sent"}},sendFailed:{{t "phone.sendFailed"}},networkError:{{t "phone.networkError"}},empty:{{t "phone.empty"}},receive:{{t "phone.receive"}},connectionError:{{t "phone.connectionError"}},share:'{{x "共有" "Share"}}',download:'{{x "ダウンロード" "Download"}}',receiving:'{{x "受け取り中…" "Receiving…"}}',downloadStarted:'{{x "ダウンロードを開始しました。ブラウザのダウンロード一覧または「ファイル」アプリを確認してください。" "Download started. Check your browser downloads or Files app."}}',shared:'{{x "共有画面を開きました。" "Share sheet opened."}}',failed:'{{x "受け取れませんでした。もう一度お試しください。" "Could not receive the file. Please try again."}}'};
const receiveStates=new Map();
let shareFilesSupported=false;

function fmt(n){if(n<1024)return n+' B';if(n<1048576)return(n/1024).toFixed(1)+' KB';if(n<1073741824)return(n/1048576).toFixed(1)+' MB';return(n/1073741824).toFixed(2)+' GB'}
function detectShareSupport(){try{const sample=new File(['x'],'x.txt',{type:'text/plain'});const data={files:[sample]};shareFilesSupported=!!navigator.share&&(!navigator.canShare||navigator.canShare(data))}catch(_){shareFilesSupported=false}}

pick.addEventListener('change',async()=>{for(const f of pick.files){await upload(f)}pick.value='';loadFiles()});
refreshButton.addEventListener('click',loadFiles);

function upload(file){return new Promise(resolve=>{const x=new XMLHttpRequest(),box=document.getElementById('progress'),bar=document.getElementById('pbar'),txt=document.getElementById('ptext'),name=document.getElementById('pname'),res=document.getElementById('result');box.style.display='block';name.textContent=msg.uploading+file.name;bar.style.width='0%';txt.textContent=msg.preparing;res.textContent='';x.upload.onprogress=e=>{if(e.lengthComputable){const p=Math.round(e.loaded/e.total*100);bar.style.width=p+'%';txt.textContent=p+'%  '+fmt(e.loaded)+' / '+fmt(e.total)}};x.onload=()=>{if(x.status>=200&&x.status<300){res.textContent=msg.sent+file.name;res.className='status ok'}else{res.textContent=x.responseText||msg.sendFailed;res.className='status'}resolve()};x.onerror=()=>{res.textContent=msg.networkError;resolve()};x.open('POST',base+'/upload?name='+encodeURIComponent(file.name));x.setRequestHeader('Content-Type','application/octet-stream');x.send(file)})}

function saveBlob(blob,name){const u=URL.createObjectURL(blob);const a=document.createElement('a');a.href=u;a.download=name;document.body.appendChild(a);a.click();a.remove();setTimeout(()=>URL.revokeObjectURL(u),30000)}
function setState(id,text,type){receiveStates.set(id,{text,type});renderStatus(id)}
function renderStatus(id){const el=document.querySelector('[data-status="'+CSS.escape(id)+'"]');if(!el)return;const s=receiveStates.get(id);if(!s){el.hidden=true;el.textContent='';el.className='fileStatus';return}el.hidden=false;el.textContent=s.text;el.className='fileStatus '+(s.type||'')}

async function receiveFile(f,button){const original=button.textContent;button.disabled=true;button.textContent=msg.receiving;setState(f.id,msg.receiving,'');try{const r=await fetch(base+'/download/'+encodeURIComponent(f.id),{cache:'no-store'});if(!r.ok)throw new Error('download failed');const blob=await r.blob();if(shareFilesSupported){try{const file=new File([blob],f.name,{type:blob.type||'application/octet-stream'});await navigator.share({files:[file],title:f.name});setState(f.id,msg.shared,'ok');return}catch(e){if(e&&e.name==='AbortError'){receiveStates.delete(f.id);renderStatus(f.id);return}}}saveBlob(blob,f.name);setState(f.id,msg.downloadStarted,'ok')}catch(e){setState(f.id,msg.failed,'error')}finally{button.disabled=false;button.textContent=original}}

async function loadFiles(){const box=document.getElementById('files');try{const r=await fetch(base+'/api/files',{cache:'no-store'});if(!r.ok)throw new Error('status');const fs=await r.json();if(!fs.length){box.innerHTML='<div class="empty"></div>';box.firstChild.textContent=msg.empty;return}box.innerHTML='';for(const f of fs){const row=document.createElement('div');row.className='row';const info=document.createElement('div');info.className='fileInfo';const name=document.createElement('div');name.className='name';name.textContent=f.name;const size=document.createElement('div');size.className='size';size.textContent=fmt(f.size);info.append(name,size);const b=document.createElement('button');b.className='button fileAction';b.type='button';b.textContent=shareFilesSupported?msg.share:msg.download;b.addEventListener('click',()=>receiveFile(f,b));const st=document.createElement('div');st.className='fileStatus';st.dataset.status=f.id;st.hidden=true;row.append(info,b,st);box.appendChild(row);renderStatus(f.id)}}catch(e){box.innerHTML='<div class="empty"></div>';box.firstChild.textContent=msg.connectionError}}

detectShareSupport();loadFiles();setInterval(loadFiles,5000);
</script></body></html>`))
