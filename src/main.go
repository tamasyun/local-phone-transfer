package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	DeviceName                 string `json:"deviceName"`
	Port                       int    `json:"port"`
	ReceiveDir                 string `json:"receiveDir"`
	SendDir                    string `json:"sendDir"`
	MaxUploadMB                int64  `json:"maxUploadMB"`
	MinFreeSpaceMB             int64  `json:"minFreeSpaceMB"`
	MaxConcurrentUploads       int    `json:"maxConcurrentUploads"`
	IdleTimeoutMinutes         int    `json:"idleTimeoutMinutes"`
	PreferredInterfaceContains string `json:"preferredInterfaceContains"`
	PreferredIPPrefix          string `json:"preferredIpPrefix"`
	FirewallRemoteSubnet       string `json:"firewallRemoteSubnet"`
	WiFiSSID                   string `json:"wifiSsid"`
	WiFiPassword               string `json:"wifiPassword"`
	ClearSendOnExit            bool   `json:"clearSendOnExit"`
}

type App struct {
	cfg          Config
	exeDir       string
	receiveDir   string
	sendDir      string
	phoneToken   string
	adminToken   string
	servers      []*http.Server
	started      time.Time
	testMode     bool
	transferIP   string
	logger       *log.Logger
	logFile      *os.File
	uploadSlots  chan struct{}
	activityMu   sync.RWMutex
	lastActivity time.Time
	shutdownOnce sync.Once
	done         chan struct{}
}

type NetURL struct { Interface, IP, URL string; Preferred bool }
type fileInfo struct { Name string `json:"name"`; ID string `json:"id"`; Size int64 `json:"size"` }

func main() {
	exe, err := os.Executable(); if err != nil { fatalDialog("起動エラー", err.Error()); return }
	exeDir := filepath.Dir(exe)
	cfg, err := loadConfig(filepath.Join(exeDir, "transfer-config.json")); if err != nil { fatalDialog("設定ファイルエラー", err.Error()); return }
	if v := strings.TrimSpace(os.Getenv("LOCAL_PHONE_TRANSFER_WIFI_SSID")); v != "" { cfg.WiFiSSID = v }
	if v := os.Getenv("LOCAL_PHONE_TRANSFER_WIFI_PASSWORD"); v != "" { cfg.WiFiPassword = v }
	testMode := strings.TrimSpace(os.Getenv("LOCAL_PHONE_TRANSFER_TEST_MODE")) == "1"
	recv, send := resolveDir(exeDir, cfg.ReceiveDir), resolveDir(exeDir, cfg.SendDir)
	if err := os.MkdirAll(recv, 0755); err != nil { fatalDialog("フォルダー作成エラー", err.Error()); return }
	if err := os.MkdirAll(send, 0755); err != nil { fatalDialog("フォルダー作成エラー", err.Error()); return }
	_ = cleanupUploadTemps(recv); _ = cleanupUploadTemps(send)
	logger, logFile := openSessionLogger(exeDir); if logFile != nil { defer logFile.Close() }
	app := &App{cfg:cfg, exeDir:exeDir, receiveDir:recv, sendDir:send, phoneToken:randomToken(12), adminToken:randomToken(16), started:time.Now(), testMode:testMode, logger:logger, logFile:logFile, uploadSlots:make(chan struct{}, cfg.MaxConcurrentUploads), lastActivity:time.Now(), done:make(chan struct{})}
	mux := http.NewServeMux(); app.routes(mux); handler := securityHeaders(mux)
	addresses := []string{fmt.Sprintf("127.0.0.1:%d", cfg.Port)}
	if !testMode {
		ip, err := waitForPreferredIP(cfg.PreferredIPPrefix, 6*time.Second); if err != nil { fatalDialog("ネットワークエラー", err.Error()); return }
		app.transferIP = ip; addresses = append(addresses, net.JoinHostPort(ip, strconv.Itoa(cfg.Port)))
	}
	errCh := make(chan error, len(addresses))
	for _, addr := range addresses {
		srv := &http.Server{Addr:addr, Handler:handler, ReadHeaderTimeout:10*time.Second, IdleTimeout:2*time.Minute, MaxHeaderBytes:32<<10}; app.servers = append(app.servers, srv)
		go func(s *http.Server){ app.logger.Printf("server listen %s test=%v", s.Addr, app.testMode); if err:=s.ListenAndServe(); err!=nil && !errors.Is(err,http.ErrServerClosed){errCh<-err} }(srv)
	}
	go app.idleMonitor(); time.Sleep(250*time.Millisecond)
	adminURL := fmt.Sprintf("http://127.0.0.1:%d/admin/%s/", cfg.Port, app.adminToken); if err:=openBrowser(adminURL);err!=nil{app.logger.Printf("open admin browser: %v",err)}
	if testMode { phoneURL:=fmt.Sprintf("http://127.0.0.1:%d/s/%s/",cfg.Port,app.phoneToken); if err:=openBrowser(phoneURL);err!=nil{app.logger.Printf("open test phone browser: %v",err)} }
	select { case err:=<-errCh: app.logger.Printf("server error: %v",err); fatalDialog("サーバー起動エラー",fmt.Sprintf("ポート %d を使用できません。\n\n%s",cfg.Port,err)); case <-app.done: app.logger.Printf("session ended") }
}

func loadConfig(path string)(Config,error){
	cfg:=Config{DeviceName:"共用PC",Port:8765,ReceiveDir:"受信ファイル",SendDir:"送信ファイル",MaxUploadMB:2048,MinFreeSpaceMB:1024,MaxConcurrentUploads:2,IdleTimeoutMinutes:180,PreferredIPPrefix:"192.168.137.",FirewallRemoteSubnet:"192.168.137.0/24",WiFiSSID:"PC-FileTransfer",ClearSendOnExit:true}
	b,err:=os.ReadFile(path);if err!=nil{return cfg,err};if err:=json.Unmarshal(b,&cfg);err!=nil{return cfg,fmt.Errorf("transfer-config.json のJSONが不正です: %w",err)}
	if cfg.Port<1024||cfg.Port>65535{return cfg,fmt.Errorf("port は1024〜65535で指定してください")};if cfg.MaxUploadMB<=0{cfg.MaxUploadMB=2048};if cfg.MinFreeSpaceMB<0{cfg.MinFreeSpaceMB=0};if cfg.MaxConcurrentUploads<1||cfg.MaxConcurrentUploads>8{cfg.MaxConcurrentUploads=2};if cfg.IdleTimeoutMinutes<1{cfg.IdleTimeoutMinutes=180};if strings.TrimSpace(cfg.PreferredIPPrefix)==""{cfg.PreferredIPPrefix="192.168.137."};if strings.TrimSpace(cfg.DeviceName)==""{cfg.DeviceName="共用PC"};return cfg,nil
}
func resolveDir(base,p string)string{if filepath.IsAbs(p){return p};return filepath.Join(base,p)}
func randomToken(n int)string{b:=make([]byte,n);if _,err:=rand.Read(b);err!=nil{return strconv.FormatInt(time.Now().UnixNano(),16)};return hex.EncodeToString(b)}
func openSessionLogger(base string)(*log.Logger,*os.File){dir:=filepath.Join(base,"logs");_=os.MkdirAll(dir,0755);path:=filepath.Join(dir,"app.log");if st,err:=os.Stat(path);err==nil&&st.Size()>2*1024*1024{_=os.Rename(path,filepath.Join(dir,"app.log.1"))};f,err:=os.OpenFile(path,os.O_CREATE|os.O_APPEND|os.O_WRONLY,0644);if err!=nil{return log.New(io.Discard,"",0),nil};return log.New(f,"",log.Ldate|log.Ltime|log.LUTC),f}
func cleanupUploadTemps(dir string)error{ents,err:=os.ReadDir(dir);if err!=nil{return err};for _,e:=range ents{if !e.IsDir()&&strings.HasPrefix(e.Name(),".upload-"){_=os.Remove(filepath.Join(dir,e.Name()))}};return nil}
func waitForPreferredIP(prefix string,timeout time.Duration)(string,error){deadline:=time.Now().Add(timeout);for{if ip:=findPreferredIP(prefix);ip!=""{return ip,nil};if time.Now().After(deadline){return "",fmt.Errorf("転送用ネットワーク（%s*）のIPv4アドレスを確認できませんでした",prefix)};time.Sleep(250*time.Millisecond)}}
func findPreferredIP(prefix string)string{ifaces,_:=net.Interfaces();for _,iface:=range ifaces{if iface.Flags&net.FlagUp==0||iface.Flags&net.FlagLoopback!=0{continue};addrs,_:=iface.Addrs();for _,addr:=range addrs{ipnet,ok:=addr.(*net.IPNet);if !ok{continue};ip:=ipnet.IP.To4();if ip!=nil&&strings.HasPrefix(ip.String(),prefix){return ip.String()}}};return ""}

func(a *App)routes(mux *http.ServeMux){mux.HandleFunc("/",a.handleRoot);mux.HandleFunc("/qr.svg",a.handleQR);mux.HandleFunc("/s/",a.handlePhone);mux.HandleFunc("/admin/",a.handleAdmin)}
func securityHeaders(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.Header().Set("X-Content-Type-Options","nosniff");w.Header().Set("X-Frame-Options","DENY");w.Header().Set("Referrer-Policy","no-referrer");w.Header().Set("Cache-Control","no-store");w.Header().Set("Permissions-Policy","camera=(), microphone=(), geolocation=(), usb=(), serial=()");w.Header().Set("Content-Security-Policy","default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'");next.ServeHTTP(w,r)})}
func(a *App)touchActivity(reason string){a.activityMu.Lock();a.lastActivity=time.Now();a.activityMu.Unlock();a.logger.Printf("activity %s",reason)}
func(a *App)idleSeconds()int{a.activityMu.RLock();t:=a.lastActivity;a.activityMu.RUnlock();return int(time.Since(t).Seconds())}
func(a *App)idleMonitor(){ticker:=time.NewTicker(30*time.Second);defer ticker.Stop();limit:=time.Duration(a.cfg.IdleTimeoutMinutes)*time.Minute;for{select{case<-ticker.C:a.activityMu.RLock();idle:=time.Since(a.lastActivity);a.activityMu.RUnlock();if idle>=limit{a.logger.Printf("idle timeout after %s",idle.Round(time.Second));a.shutdown("idle timeout");return};case<-a.done:return}}}
func(a *App)shutdown(reason string){a.shutdownOnce.Do(func(){a.logger.Printf("shutdown reason=%s",reason);if a.cfg.ClearSendOnExit{_=clearFiles(a.sendDir)};ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second);defer cancel();for _,srv:=range a.servers{_=srv.Shutdown(ctx)};close(a.done)})}

func(a *App)handleRoot(w http.ResponseWriter,r *http.Request){if r.URL.Path!="/"{http.NotFound(w,r);return};w.Header().Set("Content-Type","text/html; charset=utf-8");io.WriteString(w,`<!doctype html><meta charset="utf-8"><title>ファイル転送</title><p>デスクトップの Local Phone Transfer から起動してください。</p>`)}
func(a *App)handleQR(w http.ResponseWriter,r *http.Request){text:=r.URL.Query().Get("text");if err:=validateQRPayload(text);err!=nil{http.Error(w,err.Error(),400);return};svg,err:=qrSVG(text,6);if err!=nil{http.Error(w,err.Error(),400);return};w.Header().Set("Content-Type","image/svg+xml; charset=utf-8");io.WriteString(w,svg)}
func(a *App)handlePhone(w http.ResponseWriter,r *http.Request){prefix:="/s/"+a.phoneToken;if !strings.HasPrefix(r.URL.Path,prefix){http.NotFound(w,r);return};rest:=strings.TrimPrefix(r.URL.Path,prefix);switch{case rest==""||rest=="/":a.touchActivity("phone-page");a.phonePage(w,r);case rest=="/api/files"&&r.Method==http.MethodGet:a.apiSendFiles(w,r);case rest=="/upload"&&r.Method==http.MethodPost:a.uploadFile(w,r,a.receiveDir,"phone-upload");case strings.HasPrefix(rest,"/download/")&&r.Method==http.MethodGet:a.touchActivity("phone-download");a.downloadFile(w,r,strings.TrimPrefix(rest,"/download/"));default:http.NotFound(w,r)}}
func(a *App)phonePage(w http.ResponseWriter,r *http.Request){data:=struct{DeviceName,Token string;MaxMB int64}{a.cfg.DeviceName,a.phoneToken,a.cfg.MaxUploadMB};w.Header().Set("Content-Type","text/html; charset=utf-8");_=phoneTemplate.Execute(w,data)}
func(a *App)apiSendFiles(w http.ResponseWriter,r *http.Request){files,err:=listFiles(a.sendDir);if err!=nil{http.Error(w,"一覧を取得できません",500);return};w.Header().Set("Content-Type","application/json; charset=utf-8");_=json.NewEncoder(w).Encode(files)}
func(a *App)uploadFile(w http.ResponseWriter,r *http.Request,destDir,activity string){select{case a.uploadSlots<-struct{}{}:defer func(){<-a.uploadSlots}();default:http.Error(w,"ほかの転送が完了してからもう一度お試しください",http.StatusTooManyRequests);return};name:=cleanFilename(r.URL.Query().Get("name"));if name==""{http.Error(w,"ファイル名がありません",400);return};maxBytes:=a.cfg.MaxUploadMB*1024*1024;if r.ContentLength>maxBytes{http.Error(w,"ファイルが大きすぎます",http.StatusRequestEntityTooLarge);return};free,err:=freeDiskBytes(destDir);reserve:=uint64(a.cfg.MinFreeSpaceMB)*1024*1024;if err==nil{needed:=reserve;if r.ContentLength>0{needed+=uint64(r.ContentLength)};if free<needed{http.Error(w,"PCの空き容量が不足しています",http.StatusInsufficientStorage);return}};a.touchActivity(activity+"-start");finalPath:=uniquePath(destDir,name);tmp,err:=os.CreateTemp(destDir,".upload-*");if err!=nil{http.Error(w,"一時ファイルを作成できません",500);return};tmpName:=tmp.Name();defer os.Remove(tmpName);reader:=&activityReader{r:io.LimitReader(r.Body,maxBytes+1),touch:func(){a.touchActivity(activity+"-progress")},last:time.Now()};n,copyErr:=io.Copy(tmp,reader);closeErr:=tmp.Close();if copyErr!=nil||closeErr!=nil{http.Error(w,"保存に失敗しました",500);return};if n>maxBytes{http.Error(w,"ファイルが大きすぎます",http.StatusRequestEntityTooLarge);return};if err:=os.Rename(tmpName,finalPath);err!=nil{http.Error(w,"保存に失敗しました",500);return};a.touchActivity(activity+"-complete");a.logger.Printf("upload complete bytes=%d destination=%s",n,filepath.Base(destDir));w.Header().Set("Content-Type","application/json; charset=utf-8");_=json.NewEncoder(w).Encode(map[string]any{"ok":true,"bytes":n})}
type activityReader struct{r io.Reader;touch func();last time.Time}
func(r *activityReader)Read(p []byte)(int,error){n,err:=r.r.Read(p);if n>0&&time.Since(r.last)>=30*time.Second{r.touch();r.last=time.Now()};return n,err}
func(a *App)downloadFile(w http.ResponseWriter,r *http.Request,id string){b,err:=base64.RawURLEncoding.DecodeString(id);if err!=nil{http.NotFound(w,r);return};name:=cleanFilename(string(b));if name==""{http.NotFound(w,r);return};path:=filepath.Join(a.sendDir,name);st,err:=os.Stat(path);if err!=nil||st.IsDir(){http.NotFound(w,r);return};ctype:=mime.TypeByExtension(filepath.Ext(name));if ctype==""{ctype="application/octet-stream"};w.Header().Set("Content-Type",ctype);w.Header().Set("Content-Disposition","attachment; filename*=UTF-8''"+url.PathEscape(name));http.ServeFile(w,r,path)}

func(a *App)handleAdmin(w http.ResponseWriter,r *http.Request){host,_,_:=net.SplitHostPort(r.RemoteAddr);ip:=net.ParseIP(host);if ip==nil||!ip.IsLoopback(){http.NotFound(w,r);return};prefix:="/admin/"+a.adminToken;if !strings.HasPrefix(r.URL.Path,prefix){http.NotFound(w,r);return};rest:=strings.TrimPrefix(r.URL.Path,prefix);switch{case rest==""||rest=="/":a.adminPage(w,r);case rest=="/api/status"&&r.Method==http.MethodGet:a.adminStatus(w,r);case rest=="/upload"&&r.Method==http.MethodPost:a.uploadFile(w,r,a.sendDir,"admin-upload");case rest=="/open"&&r.Method==http.MethodPost:a.touchActivity("open-folder");a.adminOpenFolder(w,r);case rest=="/clear-send"&&r.Method==http.MethodPost:a.touchActivity("clear-send");a.adminClearSend(w,r);case rest=="/clear-receive"&&r.Method==http.MethodPost:a.touchActivity("clear-receive");a.adminClearReceive(w,r);case rest=="/stop"&&r.Method==http.MethodPost:a.adminStop(w,r);default:http.NotFound(w,r)}}
func(a *App)adminPage(w http.ResponseWriter,r *http.Request){urls:=a.networkURLs();wifiSSID,wifiPassword,wifiQR:=a.cfg.WiFiSSID,a.cfg.WiFiPassword,wifiQRText(a.cfg.WiFiSSID,a.cfg.WiFiPassword);if a.testMode{wifiSSID,wifiPassword,wifiQR="TEST MODE","Wi-Fiは使用しません",""};data:=struct{DeviceName,AdminToken,PhoneToken string;Port int;URLs []NetURL;WiFiSSID,WiFiPassword,WiFiQR,ReceiveDir,SendDir string;MaxMB int64;TestMode bool}{a.cfg.DeviceName,a.adminToken,a.phoneToken,a.cfg.Port,urls,wifiSSID,wifiPassword,wifiQR,a.receiveDir,a.sendDir,a.cfg.MaxUploadMB,a.testMode};w.Header().Set("Content-Type","text/html; charset=utf-8");_=adminTemplate.Execute(w,data)}
func(a *App)adminStatus(w http.ResponseWriter,r *http.Request){recv,_:=listFiles(a.receiveDir);send,_:=listFiles(a.sendDir);remaining:=a.cfg.IdleTimeoutMinutes*60-a.idleSeconds();if remaining<0{remaining=0};w.Header().Set("Content-Type","application/json; charset=utf-8");_=json.NewEncoder(w).Encode(map[string]any{"receive":recv,"send":send,"idleRemainingSec":remaining,"idleTimeoutMinutes":a.cfg.IdleTimeoutMinutes,"testMode":a.testMode})}
func(a *App)adminOpenFolder(w http.ResponseWriter,r *http.Request){p:=a.receiveDir;if r.URL.Query().Get("which")=="send"{p=a.sendDir};if err:=openFolder(p);err!=nil{http.Error(w,"フォルダーを開けません",500);return};w.WriteHeader(204)}
func(a *App)adminClearSend(w http.ResponseWriter,r *http.Request){if err:=clearFiles(a.sendDir);err!=nil{http.Error(w,"削除できません",500);return};w.WriteHeader(204)}
func(a *App)adminClearReceive(w http.ResponseWriter,r *http.Request){if err:=clearFiles(a.receiveDir);err!=nil{http.Error(w,"削除できません",500);return};w.WriteHeader(204)}
func(a *App)adminStop(w http.ResponseWriter,r *http.Request){w.Header().Set("Content-Type","application/json");io.WriteString(w,`{"ok":true}`);go func(){time.Sleep(250*time.Millisecond);a.shutdown("user")}()}

func listFiles(dir string)([]fileInfo,error){ents,err:=os.ReadDir(dir);if err!=nil{return nil,err};out:=[]fileInfo{};for _,e:=range ents{if e.IsDir()||strings.HasPrefix(e.Name(),".upload-"){continue};st,err:=e.Info();if err!=nil{continue};out=append(out,fileInfo{Name:e.Name(),ID:base64.RawURLEncoding.EncodeToString([]byte(e.Name())),Size:st.Size()})};sort.Slice(out,func(i,j int)bool{return out[i].Name<out[j].Name});return out,nil}
func clearFiles(dir string)error{ents,err:=os.ReadDir(dir);if err!=nil{return err};for _,e:=range ents{if e.IsDir(){continue};if err:=os.Remove(filepath.Join(dir,e.Name()));err!=nil{return err}};return nil}
func cleanFilename(s string)string{s=strings.TrimSpace(filepath.Base(strings.ReplaceAll(s,"\x00","")));if s=="."||s==".."||s==""{return ""};repl:=strings.NewReplacer("<","_",">","_",":","_",`"`,"_","/","_",`\\`,"_","|","_","?","_","*","_");s=strings.TrimRight(repl.Replace(s)," .");if len([]rune(s))>180{ext:=filepath.Ext(s);base:=strings.TrimSuffix(s,ext);br:=[]rune(base);if len(br)>140{base=string(br[:140])};s=base+ext};return s}
func uniquePath(dir,name string)string{p:=filepath.Join(dir,name);if _,err:=os.Stat(p);errors.Is(err,os.ErrNotExist){return p};ext:=filepath.Ext(name);base:=strings.TrimSuffix(name,ext);for i:=2;;i++{c:=filepath.Join(dir,fmt.Sprintf("%s (%d)%s",base,i,ext));if _,err:=os.Stat(c);errors.Is(err,os.ErrNotExist){return c}}}
func(a *App)networkURLs()[]NetURL{if a.testMode{return []NetURL{{Interface:"localhost (test mode)",IP:"127.0.0.1",URL:fmt.Sprintf("http://127.0.0.1:%d/s/%s/",a.cfg.Port,a.phoneToken),Preferred:true}}};if a.transferIP==""{return nil};return []NetURL{{Interface:"transfer Wi-Fi",IP:a.transferIP,URL:fmt.Sprintf("http://%s:%d/s/%s/",a.transferIP,a.cfg.Port,a.phoneToken),Preferred:true}}}
func wifiQRText(ssid,password string)string{esc:=func(s string)string{r:=strings.NewReplacer(`\\`,`\\\\`,`;`,`\\;`,`,`,`\\,`,`:`,`\\:`);return r.Replace(s)};if strings.TrimSpace(ssid)==""{return ""};return fmt.Sprintf("WIFI:T:WPA;S:%s;P:%s;;",esc(ssid),esc(password))}
func openBrowser(u string)error{switch runtime.GOOS{case"windows":return exec.Command("rundll32","url.dll,FileProtocolHandler",u).Start();case"darwin":return exec.Command("open",u).Start();default:return exec.Command("xdg-open",u).Start()}}
func openFolder(path string)error{switch runtime.GOOS{case"windows":return exec.Command("explorer.exe",path).Start();case"darwin":return exec.Command("open",path).Start();default:return exec.Command("xdg-open",path).Start()}}
func fatalDialog(title,message string){if runtime.GOOS=="windows"{script:=fmt.Sprintf(`Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show(%q,%q)`,message,title);_=exec.Command("powershell.exe","-NoProfile","-Command",script).Run();return};log.Printf("%s: %s",title,message)}
