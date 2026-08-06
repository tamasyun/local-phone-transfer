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
	PreferredInterfaceContains string `json:"preferredInterfaceContains"`
	PreferredIPPrefix          string `json:"preferredIpPrefix"`
	WiFiSSID                   string `json:"wifiSsid"`
	WiFiPassword               string `json:"wifiPassword"`
	ClearSendOnExit            bool   `json:"clearSendOnExit"`
	AutoStopMinutes            int    `json:"autoStopMinutes"`
}

type App struct {
	cfg          Config
	exeDir       string
	receiveDir   string
	sendDir      string
	phoneToken   string
	adminToken   string
	server       *http.Server
	started      time.Time
	mu           sync.RWMutex
	preferredURL string
}

type NetURL struct {
	Interface string
	IP        string
	URL       string
	Preferred bool
}

type fileInfo struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	Size int64  `json:"size"`
}

func main() {
	exe, err := os.Executable()
	if err != nil {
		fatalDialog("起動エラー", err.Error())
		return
	}
	exeDir := filepath.Dir(exe)
	cfgPath := filepath.Join(exeDir, "transfer-config.json")
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		fatalDialog("設定ファイルエラー", err.Error())
		return
	}
	if v := strings.TrimSpace(os.Getenv("LOCAL_PHONE_TRANSFER_WIFI_SSID")); v != "" {
		cfg.WiFiSSID = v
	}
	if v := os.Getenv("LOCAL_PHONE_TRANSFER_WIFI_PASSWORD"); v != "" {
		cfg.WiFiPassword = v
	}
	recv := resolveDir(exeDir, cfg.ReceiveDir)
	send := resolveDir(exeDir, cfg.SendDir)
	if err := os.MkdirAll(recv, 0755); err != nil {
		fatalDialog("フォルダー作成エラー", err.Error())
		return
	}
	if err := os.MkdirAll(send, 0755); err != nil {
		fatalDialog("フォルダー作成エラー", err.Error())
		return
	}

	app := &App{
		cfg:        cfg,
		exeDir:     exeDir,
		receiveDir: recv,
		sendDir:    send,
		phoneToken: randomToken(12),
		adminToken: randomToken(16),
		started:    time.Now(),
	}
	urls := app.networkURLs()
	if len(urls) > 0 {
		app.preferredURL = urls[0].URL
	}

	mux := http.NewServeMux()
	app.routes(mux)
	app.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	errCh := make(chan error, 1)
	go func() {
		err := app.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	time.Sleep(250 * time.Millisecond)
	if cfg.AutoStopMinutes > 0 {
		go func() {
			time.Sleep(time.Duration(cfg.AutoStopMinutes) * time.Minute)
			if cfg.ClearSendOnExit {
				_ = clearFiles(app.sendDir)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = app.server.Shutdown(ctx)
			os.Exit(0)
		}()
	}
	adminURL := fmt.Sprintf("http://127.0.0.1:%d/admin/%s/", cfg.Port, app.adminToken)
	if err := openBrowser(adminURL); err != nil {
		log.Printf("open browser: %v", err)
	}

	select {
	case err := <-errCh:
		fatalDialog("サーバー起動エラー", fmt.Sprintf("ポート %d を使用できません。\n\n%s", cfg.Port, err))
	case <-make(chan struct{}):
	}
}

func loadConfig(path string) (Config, error) {
	cfg := Config{
		DeviceName:                 "共用PC",
		Port:                       8765,
		ReceiveDir:                 "受信ファイル",
		SendDir:                    "送信ファイル",
		MaxUploadMB:                2048,
		PreferredInterfaceContains: "",
		PreferredIPPrefix:          "",
		WiFiSSID:                   "PC-FileTransfer",
		WiFiPassword:               "CHANGE-ME",
		ClearSendOnExit:            true,
		AutoStopMinutes:            60,
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		out, _ := json.MarshalIndent(cfg, "", "  ")
		if werr := os.WriteFile(path, out, 0644); werr != nil {
			return cfg, fmt.Errorf("設定ファイルを作成できません: %w", werr)
		}
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("transfer-config.json のJSONが不正です: %w", err)
	}
	if cfg.Port < 1024 || cfg.Port > 65535 {
		return cfg, fmt.Errorf("port は1024〜65535で指定してください")
	}
	if cfg.MaxUploadMB <= 0 {
		cfg.MaxUploadMB = 2048
	}
	if strings.TrimSpace(cfg.DeviceName) == "" {
		cfg.DeviceName = "共用PC"
	}
	if cfg.AutoStopMinutes < 0 {
		cfg.AutoStopMinutes = 0
	}
	return cfg, nil
}

func resolveDir(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

func (a *App) routes(mux *http.ServeMux) {
	mux.HandleFunc("/", a.handleRoot)
	mux.HandleFunc("/qr.svg", a.handleQR)
	mux.HandleFunc("/s/", a.handlePhone)
	mux.HandleFunc("/admin/", a.handleAdmin)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (a *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, `<!doctype html><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>スマホファイル転送</title><style>body{font-family:system-ui,sans-serif;max-width:680px;margin:60px auto;padding:24px;line-height:1.7}div{border:1px solid #ddd;border-radius:16px;padding:24px}</style><div><h1>スマホファイル転送</h1><p>このURLからは転送画面を開けません。PCのデスクトップからファイル転送を起動してください。</p></div>`)
}

func (a *App) handleQR(w http.ResponseWriter, r *http.Request) {
	text := r.URL.Query().Get("text")
	if err := validateQRPayload(text); err != nil {
		w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, qrSVGError(err.Error()))
		return
	}
	svg, err := qrSVG(text, 6)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	io.WriteString(w, svg)
}

func (a *App) handlePhone(w http.ResponseWriter, r *http.Request) {
	prefix := "/s/" + a.phoneToken
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	switch {
	case rest == "" || rest == "/":
		a.phonePage(w, r)
	case rest == "/api/files":
		a.apiSendFiles(w, r)
	case rest == "/upload" && r.Method == http.MethodPost:
		a.uploadFile(w, r, a.receiveDir)
	case strings.HasPrefix(rest, "/download/") && r.Method == http.MethodGet:
		a.downloadFile(w, r, strings.TrimPrefix(rest, "/download/"))
	default:
		http.NotFound(w, r)
	}
}

func (a *App) phonePage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		DeviceName string
		Token      string
		MaxMB      int64
	}{a.cfg.DeviceName, a.phoneToken, a.cfg.MaxUploadMB}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = phoneTemplate.Execute(w, data)
}

func (a *App) apiSendFiles(w http.ResponseWriter, r *http.Request) {
	files, err := listFiles(a.sendDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(files)
}

func (a *App) uploadFile(w http.ResponseWriter, r *http.Request, destDir string) {
	name := cleanFilename(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, "ファイル名がありません", http.StatusBadRequest)
		return
	}
	maxBytes := a.cfg.MaxUploadMB * 1024 * 1024
	if r.ContentLength > maxBytes {
		http.Error(w, "ファイルが大きすぎます", http.StatusRequestEntityTooLarge)
		return
	}
	finalPath := uniquePath(destDir, name)
	tmp, err := os.CreateTemp(destDir, ".upload-*")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	limited := io.LimitReader(r.Body, maxBytes+1)
	n, err := io.Copy(tmp, limited)
	closeErr := tmp.Close()
	if err != nil || closeErr != nil {
		http.Error(w, "保存に失敗しました", http.StatusInternalServerError)
		return
	}
	if n > maxBytes {
		http.Error(w, "ファイルが大きすぎます", http.StatusRequestEntityTooLarge)
		return
	}
	if err := os.Rename(tmpName, finalPath); err != nil {
		http.Error(w, "保存に失敗しました: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "name": filepath.Base(finalPath), "bytes": n})
}

func (a *App) downloadFile(w http.ResponseWriter, r *http.Request, id string) {
	b, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	name := cleanFilename(string(b))
	if name == "" {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(a.sendDir, name)
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}
	ctype := mime.TypeByExtension(filepath.Ext(name))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(name))
	http.ServeFile(w, r, path)
}

func (a *App) handleAdmin(w http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		http.NotFound(w, r)
		return
	}
	prefix := "/admin/" + a.adminToken
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	switch {
	case rest == "" || rest == "/":
		a.adminPage(w, r)
	case rest == "/api/status":
		a.adminStatus(w, r)
	case rest == "/upload" && r.Method == http.MethodPost:
		a.uploadFile(w, r, a.sendDir)
	case rest == "/open" && r.Method == http.MethodPost:
		a.adminOpenFolder(w, r)
	case rest == "/clear-send" && r.Method == http.MethodPost:
		a.adminClearSend(w, r)
	case rest == "/clear-receive" && r.Method == http.MethodPost:
		a.adminClearReceive(w, r)
	case rest == "/stop" && r.Method == http.MethodPost:
		a.adminStop(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (a *App) adminPage(w http.ResponseWriter, r *http.Request) {
	urls := a.networkURLs()
	data := struct {
		DeviceName   string
		AdminToken   string
		PhoneToken   string
		Port         int
		URLs         []NetURL
		WiFiSSID     string
		WiFiPassword string
		WiFiQR       string
		ReceiveDir   string
		SendDir      string
		MaxMB        int64
	}{
		DeviceName: a.cfg.DeviceName, AdminToken: a.adminToken, PhoneToken: a.phoneToken, Port: a.cfg.Port,
		URLs: urls, WiFiSSID: a.cfg.WiFiSSID, WiFiPassword: a.cfg.WiFiPassword,
		WiFiQR: wifiQRText(a.cfg.WiFiSSID, a.cfg.WiFiPassword), ReceiveDir: a.receiveDir, SendDir: a.sendDir, MaxMB: a.cfg.MaxUploadMB,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = adminTemplate.Execute(w, data)
}

func (a *App) adminStatus(w http.ResponseWriter, r *http.Request) {
	recv, _ := listFiles(a.receiveDir)
	send, _ := listFiles(a.sendDir)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"receive": recv, "send": send, "uptimeSec": int(time.Since(a.started).Seconds()), "autoStopMinutes": a.cfg.AutoStopMinutes})
}

func (a *App) adminOpenFolder(w http.ResponseWriter, r *http.Request) {
	which := r.URL.Query().Get("which")
	p := a.receiveDir
	if which == "send" { p = a.sendDir }
	if err := openFolder(p); err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) adminClearSend(w http.ResponseWriter, r *http.Request) { if err := clearFiles(a.sendDir); err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }; w.WriteHeader(http.StatusNoContent) }
func (a *App) adminClearReceive(w http.ResponseWriter, r *http.Request) { if err := clearFiles(a.receiveDir); err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }; w.WriteHeader(http.StatusNoContent) }

func (a *App) adminStop(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"ok":true}`)
	go func() {
		time.Sleep(400 * time.Millisecond)
		if a.cfg.ClearSendOnExit { _ = clearFiles(a.sendDir) }
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second); defer cancel()
		_ = a.server.Shutdown(ctx)
		os.Exit(0)
	}()
}

func listFiles(dir string) ([]fileInfo, error) {
	ents, err := os.ReadDir(dir); if err != nil { return nil, err }
	out := make([]fileInfo, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".upload-") { continue }
		st, err := e.Info(); if err != nil { continue }
		out = append(out, fileInfo{Name: e.Name(), ID: base64.RawURLEncoding.EncodeToString([]byte(e.Name())), Size: st.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func clearFiles(dir string) error {
	ents, err := os.ReadDir(dir); if err != nil { return err }
	for _, e := range ents { if e.IsDir() { continue }; if err := os.Remove(filepath.Join(dir, e.Name())); err != nil { return err } }
	return nil
}

func cleanFilename(s string) string {
	s = strings.TrimSpace(s); s = filepath.Base(s); s = strings.ReplaceAll(s, "\x00", "")
	if s == "." || s == ".." || s == "" { return "" }
	repl := strings.NewReplacer("<", "_", ">", "_", ":", "_", `"`, "_", "/", "_", `\\`, "_", "|", "_", "?", "_", "*", "_")
	s = repl.Replace(s); s = strings.TrimRight(s, " .")
	if len([]rune(s)) > 180 { ext := filepath.Ext(s); base := strings.TrimSuffix(s, ext); br := []rune(base); if len(br) > 140 { base = string(br[:140]) }; s = base + ext }
	return s
}

func uniquePath(dir, name string) string {
	p := filepath.Join(dir, name); if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) { return p }
	ext := filepath.Ext(name); base := strings.TrimSuffix(name, ext)
	for i := 2; ; i++ { candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext)); if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) { return candidate } }
}

func (a *App) networkURLs() []NetURL {
	ifaces, _ := net.Interfaces(); var out []NetURL
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 { continue }
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet); if !ok { continue }
			ip := ipnet.IP.To4(); if ip == nil || ip.IsLoopback() || ip.IsUnspecified() { continue }
			s := ip.String(); if strings.HasPrefix(s, "169.254.") { continue }
			u := fmt.Sprintf("http://%s:%d/s/%s/", s, a.cfg.Port, a.phoneToken)
			preferred := false
			if a.cfg.PreferredIPPrefix != "" && strings.HasPrefix(s, a.cfg.PreferredIPPrefix) { preferred = true }
			if a.cfg.PreferredInterfaceContains != "" && strings.Contains(strings.ToLower(iface.Name), strings.ToLower(a.cfg.PreferredInterfaceContains)) { preferred = true }
			out = append(out, NetURL{Interface: iface.Name, IP: s, URL: u, Preferred: preferred})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { if out[i].Preferred != out[j].Preferred { return out[i].Preferred }; return privateRank(out[i].IP) > privateRank(out[j].IP) })
	return out
}

func privateRank(ipstr string) int { ip := net.ParseIP(ipstr).To4(); if ip == nil { return 0 }; if ip[0] == 192 && ip[1] == 168 { return 3 }; if ip[0] == 10 { return 2 }; if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 { return 2 }; return 1 }

func wifiQRText(ssid, password string) string {
	esc := func(s string) string { r := strings.NewReplacer(`\\`, `\\\\`, `;`, `\\;`, `,`, `\\,`, `:`, `\\:`); return r.Replace(s) }
	if strings.TrimSpace(ssid) == "" { return "" }
	return fmt.Sprintf("WIFI:T:WPA;S:%s;P:%s;;", esc(ssid), esc(password))
}

func openBrowser(u string) error { switch runtime.GOOS { case "windows": return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start(); case "darwin": return exec.Command("open", u).Start(); default: return exec.Command("xdg-open", u).Start() } }
func openFolder(path string) error { switch runtime.GOOS { case "windows": return exec.Command("explorer.exe", path).Start(); case "darwin": return exec.Command("open", path).Start(); default: return exec.Command("xdg-open", path).Start() } }
func fatalDialog(title, message string) { if runtime.GOOS == "windows" { script := fmt.Sprintf(`Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show(%q,%q)`, message, title); _ = exec.Command("powershell.exe", "-NoProfile", "-Command", script).Run(); return }; log.Printf("%s: %s", title, message) }
