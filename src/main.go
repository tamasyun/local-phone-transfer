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
	TransferSubnet             string `json:"transferSubnet"`
	WiFiSSID                   string `json:"wifiSsid"`
	WiFiPassword               string `json:"wifiPassword"`
	ClearSendOnExit            bool   `json:"clearSendOnExit"`
	IdleStopMinutes            int    `json:"idleStopMinutes"`
	AbsoluteStopMinutes        int    `json:"absoluteStopMinutes"`
	MaxConcurrentUploads       int    `json:"maxConcurrentUploads"`
	MinFreeSpaceMB             int64  `json:"minFreeSpaceMB"`
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
	preferredURL string
	testMode     bool
	transferNet  *net.IPNet
	uploadSlots  chan struct{}
	logger       *log.Logger

	mu           sync.RWMutex
	lastActivity time.Time
	phoneIP      string
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
	cfg, err := loadConfig(filepath.Join(exeDir, "transfer-config.json"))
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
	testMode := strings.TrimSpace(os.Getenv("LOCAL_PHONE_TRANSFER_TEST_MODE")) == "1"

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

	cleanupPartialUploads(recv)
	cleanupPartialUploads(send)
	if cfg.ClearSendOnExit {
		_ = clearFiles(send)
	}

	_, transferNet, err := net.ParseCIDR(cfg.TransferSubnet)
	if err != nil {
		fatalDialog("設定ファイルエラー", "transferSubnet が不正です: "+err.Error())
		return
	}

	logger, closeLog := newSessionLogger(exeDir)
	defer closeLog()
	logger.Printf("session_start test_mode=%t", testMode)

	now := time.Now()
	app := &App{
		cfg:          cfg,
		exeDir:       exeDir,
		receiveDir:   recv,
		sendDir:      send,
		phoneToken:   randomToken(18),
		adminToken:   randomToken(24),
		started:      now,
		lastActivity: now,
		testMode:     testMode,
		transferNet:  transferNet,
		uploadSlots:  make(chan struct{}, cfg.MaxConcurrentUploads),
		logger:       logger,
	}
	urls := app.networkURLs()
	if len(urls) > 0 {
		app.preferredURL = urls[0].URL
	}

	mux := http.NewServeMux()
	app.routes(mux)
	app.server = &http.Server{
		Addr:              listenAddress(cfg.Port, testMode),
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       0,
		WriteTimeout:      0,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    32 << 10,
	}

	errCh := make(chan error, 1)
	go func() {
		err := app.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go app.timeoutMonitor()

	time.Sleep(250 * time.Millisecond)
	adminURL := fmt.Sprintf("http://127.0.0.1:%d/admin/%s/", cfg.Port, app.adminToken)
	if err := openBrowser(adminURL); err != nil {
		logger.Printf("open_admin_browser error=%q", err)
	}
	if testMode {
		time.Sleep(250 * time.Millisecond)
		phoneURL := fmt.Sprintf("http://127.0.0.1:%d/s/%s/", cfg.Port, app.phoneToken)
		if err := openBrowser(phoneURL); err != nil {
			logger.Printf("open_test_phone_browser error=%q", err)
		}
	}

	select {
	case err := <-errCh:
		logger.Printf("server_error error=%q", err)
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
		PreferredIPPrefix:          "192.168.137.",
		TransferSubnet:             "192.168.137.0/24",
		WiFiSSID:                   "PC-FileTransfer",
		WiFiPassword:               "CHANGE-ME",
		ClearSendOnExit:            true,
		IdleStopMinutes:            180,
		AbsoluteStopMinutes:        480,
		MaxConcurrentUploads:       2,
		MinFreeSpaceMB:             1024,
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
	if strings.TrimSpace(cfg.TransferSubnet) == "" {
		cfg.TransferSubnet = "192.168.137.0/24"
	}
	if _, _, err := net.ParseCIDR(cfg.TransferSubnet); err != nil {
		return cfg, fmt.Errorf("transferSubnet が不正です: %w", err)
	}
	if cfg.IdleStopMinutes <= 0 {
		cfg.IdleStopMinutes = 180
	}
	if cfg.AbsoluteStopMinutes <= 0 {
		cfg.AbsoluteStopMinutes = 480
	}
	if cfg.MaxConcurrentUploads < 1 || cfg.MaxConcurrentUploads > 8 {
		cfg.MaxConcurrentUploads = 2
	}
	if cfg.MinFreeSpaceMB < 0 {
		cfg.MinFreeSpaceMB = 1024
	}
	return cfg, nil
}

func listenAddress(port int, testMode bool) string {
	if testMode {
		return fmt.Sprintf("127.0.0.1:%d", port)
	}
	return fmt.Sprintf(":%d", port)
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
		panic("secure random source unavailable: " + err.Error())
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
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), usb=(), serial=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

func remoteIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil
	}
	return net.ParseIP(host)
}

func (a *App) allowedPhoneRequest(r *http.Request) bool {
	ip := remoteIP(r)
	if ip == nil {
		return false
	}
	if a.testMode {
		return ip.IsLoopback()
	}
	if !a.transferNet.Contains(ip) {
		return false
	}
	key := ip.String()
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.phoneIP == "" {
		a.phoneIP = key
		a.logger.Printf("phone_bound ip=%s", key)
	}
	return a.phoneIP == key
}

func (a *App) touchActivity(kind string) {
	a.mu.Lock()
	a.lastActivity = time.Now()
	a.mu.Unlock()
	if kind != "" {
		a.logger.Printf("activity kind=%s", kind)
	}
}

func (a *App) timeoutMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		a.mu.RLock()
		last := a.lastActivity
		a.mu.RUnlock()
		if a.cfg.IdleStopMinutes > 0 && now.Sub(last) >= time.Duration(a.cfg.IdleStopMinutes)*time.Minute {
			a.shutdown("idle_timeout")
			return
		}
		if a.cfg.AbsoluteStopMinutes > 0 && now.Sub(a.started) >= time.Duration(a.cfg.AbsoluteStopMinutes)*time.Minute {
			a.shutdown("absolute_timeout")
			return
		}
	}
}

func (a *App) shutdown(reason string) {
	a.logger.Printf("session_stop reason=%s", reason)
	if a.cfg.ClearSendOnExit {
		_ = clearFiles(a.sendDir)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = a.server.Shutdown(ctx)
	os.Exit(0)
}

func (a *App) timeoutState() (idleRemaining, absoluteRemaining int64) {
	now := time.Now()
	a.mu.RLock()
	last := a.lastActivity
	a.mu.RUnlock()
	idleRemaining = int64((time.Duration(a.cfg.IdleStopMinutes)*time.Minute - now.Sub(last)).Seconds())
	absoluteRemaining = int64((time.Duration(a.cfg.AbsoluteStopMinutes)*time.Minute - now.Sub(a.started)).Seconds())
	if idleRemaining < 0 {
		idleRemaining = 0
	}
	if absoluteRemaining < 0 {
		absoluteRemaining = 0
	}
	return
}

func (a *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || !remoteIP(r).IsLoopback() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, `<!doctype html><meta charset="utf-8"><title>ファイル転送</title><p>デスクトップの Local Phone Transfer から起動してください。</p>`)
}

func (a *App) handleQR(w http.ResponseWriter, r *http.Request) {
	ip := remoteIP(r)
	if ip == nil || !ip.IsLoopback() {
		http.NotFound(w, r)
		return
	}
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
	if !a.allowedPhoneRequest(r) {
		http.NotFound(w, r)
		return
	}
	prefix := "/s/" + a.phoneToken
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	switch {
	case rest == "" || rest == "/":
		a.touchActivity("phone_open")
		a.phonePage(w)
	case rest == "/api/files" && r.Method == http.MethodGet:
		a.apiSendFiles(w)
	case rest == "/upload" && r.Method == http.MethodPost:
		a.uploadFile(w, r, a.receiveDir, "phone_upload")
	case strings.HasPrefix(rest, "/download/") && r.Method == http.MethodGet:
		a.downloadFile(w, r, strings.TrimPrefix(rest, "/download/"))
	default:
		http.NotFound(w, r)
	}
}

func (a *App) phonePage(w http.ResponseWriter) {
	data := struct {
		DeviceName string
		Token      string
		MaxMB      int64
	}{a.cfg.DeviceName, a.phoneToken, a.cfg.MaxUploadMB}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = phoneTemplate.Execute(w, data)
}

func (a *App) apiSendFiles(w http.ResponseWriter) {
	files, err := listFiles(a.sendDir)
	if err != nil {
		http.Error(w, "一覧を取得できません", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(files)
}

func (a *App) acquireUpload(w http.ResponseWriter) bool {
	select {
	case a.uploadSlots <- struct{}{}:
		return true
	default:
		http.Error(w, "ほかの転送処理が進行中です。少し待ってから再試行してください。", http.StatusTooManyRequests)
		return false
	}
}

func (a *App) uploadFile(w http.ResponseWriter, r *http.Request, destDir, activityKind string) {
	if !a.acquireUpload(w) {
		return
	}
	defer func() { <-a.uploadSlots }()

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
	reserve := uint64(a.cfg.MinFreeSpaceMB) * 1024 * 1024
	free, err := diskFreeBytes(destDir)
	if err == nil {
		needed := reserve
		if r.ContentLength > 0 {
			needed += uint64(r.ContentLength)
		} else {
			needed += uint64(maxBytes)
		}
		if free < needed {
			http.Error(w, "PCの空き容量が不足しています", http.StatusInsufficientStorage)
			return
		}
	}

	a.touchActivity(activityKind + "_start")
	finalPath := uniquePath(destDir, name)
	tmp, err := os.CreateTemp(destDir, ".upload-*")
	if err != nil {
		http.Error(w, "一時ファイルを作成できません", http.StatusInternalServerError)
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	limited := io.LimitReader(r.Body, maxBytes+1)
	n, copyErr := io.Copy(tmp, limited)
	closeErr := tmp.Close()
	if copyErr != nil || closeErr != nil {
		http.Error(w, "保存に失敗しました", http.StatusInternalServerError)
		return
	}
	if n > maxBytes {
		http.Error(w, "ファイルが大きすぎます", http.StatusRequestEntityTooLarge)
		return
	}
	if err := os.Rename(tmpName, finalPath); err != nil {
		http.Error(w, "保存に失敗しました", http.StatusInternalServerError)
		return
	}
	a.touchActivity(activityKind + "_complete")
	a.logger.Printf("file_saved kind=%s name=%q bytes=%d", activityKind, filepath.Base(finalPath), n)
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
	a.touchActivity("phone_download")
	a.logger.Printf("file_download name=%q bytes=%d", name, st.Size())
	ctype := mime.TypeByExtension(filepath.Ext(name))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(name))
	http.ServeFile(w, r, path)
}

func (a *App) handleAdmin(w http.ResponseWriter, r *http.Request) {
	ip := remoteIP(r)
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
		a.touchActivity("admin_open")
		a.adminPage(w)
	case rest == "/api/status" && r.Method == http.MethodGet:
		a.adminStatus(w)
	case rest == "/upload" && r.Method == http.MethodPost:
		a.uploadFile(w, r, a.sendDir, "admin_share")
	case rest == "/open" && r.Method == http.MethodPost:
		a.adminOpenFolder(w, r)
	case rest == "/clear-send" && r.Method == http.MethodPost:
		a.adminClear(w, "send")
	case rest == "/clear-receive" && r.Method == http.MethodPost:
		a.adminClear(w, "receive")
	case rest == "/stop" && r.Method == http.MethodPost:
		a.adminStop(w)
	default:
		http.NotFound(w, r)
	}
}

func (a *App) adminPage(w http.ResponseWriter) {
	urls := a.networkURLs()
	wifiSSID, wifiPassword := a.cfg.WiFiSSID, a.cfg.WiFiPassword
	wifiQR := wifiQRText(wifiSSID, wifiPassword)
	if a.testMode {
		wifiSSID, wifiPassword, wifiQR = "テストモード", "Wi-Fiは使用しません", ""
	}
	data := struct {
		DeviceName   string
		AdminToken   string
		URLs         []NetURL
		WiFiSSID     string
		WiFiPassword string
		WiFiQR       string
		MaxMB        int64
		TestMode     bool
	}{a.cfg.DeviceName, a.adminToken, urls, wifiSSID, wifiPassword, wifiQR, a.cfg.MaxUploadMB, a.testMode}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = adminTemplate.Execute(w, data)
}

func (a *App) adminStatus(w http.ResponseWriter) {
	recv, _ := listFiles(a.receiveDir)
	send, _ := listFiles(a.sendDir)
	idleRemaining, absoluteRemaining := a.timeoutState()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"receive": recv, "send": send, "testMode": a.testMode,
		"idleRemainingSec": idleRemaining, "absoluteRemainingSec": absoluteRemaining,
	})
}

func (a *App) adminOpenFolder(w http.ResponseWriter, r *http.Request) {
	which := r.URL.Query().Get("which")
	p := a.receiveDir
	if which == "send" {
		p = a.sendDir
	}
	if err := openFolder(p); err != nil {
		http.Error(w, "フォルダーを開けません", http.StatusInternalServerError)
		return
	}
	a.touchActivity("admin_open_folder")
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) adminClear(w http.ResponseWriter, which string) {
	p := a.receiveDir
	if which == "send" {
		p = a.sendDir
	}
	if err := clearFiles(p); err != nil {
		http.Error(w, "削除できません", http.StatusInternalServerError)
		return
	}
	a.touchActivity("admin_clear_" + which)
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) adminStop(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"ok":true}`)
	go func() {
		time.Sleep(300 * time.Millisecond)
		a.shutdown("user_stop")
	}()
}

func listFiles(dir string) ([]fileInfo, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]fileInfo, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".upload-") {
			continue
		}
		st, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, fileInfo{Name: e.Name(), ID: base64.RawURLEncoding.EncodeToString([]byte(e.Name())), Size: st.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func clearFiles(dir string) error {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func cleanupPartialUploads(dir string) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range ents {
		if !e.IsDir() && strings.HasPrefix(e.Name(), ".upload-") {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

func cleanFilename(s string) string {
	s = strings.TrimSpace(s)
	s = filepath.Base(s)
	s = strings.ReplaceAll(s, "\x00", "")
	if s == "." || s == ".." || s == "" {
		return ""
	}
	repl := strings.NewReplacer("<", "_", ">", "_", ":", "_", `"`, "_", "/", "_", `\`, "_", "|", "_", "?", "_", "*", "_")
	s = repl.Replace(s)
	s = strings.TrimRight(s, " .")
	if len([]rune(s)) > 180 {
		ext := filepath.Ext(s)
		base := strings.TrimSuffix(s, ext)
		br := []rune(base)
		if len(br) > 140 {
			base = string(br[:140])
		}
		s = base + ext
	}
	return s
}

func uniquePath(dir, name string) string {
	p := filepath.Join(dir, name)
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		return p
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 2; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
}

func (a *App) networkURLs() []NetURL {
	if a.testMode {
		return []NetURL{{Interface: "localhost", IP: "127.0.0.1", URL: fmt.Sprintf("http://127.0.0.1:%d/s/%s/", a.cfg.Port, a.phoneToken), Preferred: true}}
	}
	ifaces, _ := net.Interfaces()
	var out []NetURL
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipnet.IP.To4()
			if ip == nil || !a.transferNet.Contains(ip) {
				continue
			}
			s := ip.String()
			u := fmt.Sprintf("http://%s:%d/s/%s/", s, a.cfg.Port, a.phoneToken)
			preferred := strings.HasPrefix(s, a.cfg.PreferredIPPrefix)
			if a.cfg.PreferredInterfaceContains != "" && strings.Contains(strings.ToLower(iface.Name), strings.ToLower(a.cfg.PreferredInterfaceContains)) {
				preferred = true
			}
			out = append(out, NetURL{Interface: iface.Name, IP: s, URL: u, Preferred: preferred})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Preferred != out[j].Preferred {
			return out[i].Preferred
		}
		return out[i].IP < out[j].IP
	})
	return out
}

func wifiQRText(ssid, password string) string {
	esc := func(s string) string {
		r := strings.NewReplacer(`\`, `\\`, `;`, `\;`, `,`, `\,`, `:`, `\:`)
		return r.Replace(s)
	}
	if strings.TrimSpace(ssid) == "" {
		return ""
	}
	return fmt.Sprintf("WIFI:T:WPA;S:%s;P:%s;;", esc(ssid), esc(password))
}

func newSessionLogger(exeDir string) (*log.Logger, func()) {
	logDir := filepath.Join(exeDir, "logs")
	_ = os.MkdirAll(logDir, 0755)
	cleanupOldLogs(logDir, 14*24*time.Hour)
	name := "session-" + time.Now().Format("20060102-150405") + ".log"
	f, err := os.OpenFile(filepath.Join(logDir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return log.New(io.Discard, "", log.LstdFlags), func() {}
	}
	return log.New(f, "", log.LstdFlags), func() { _ = f.Close() }
}

func cleanupOldLogs(dir string, maxAge time.Duration) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, e := range ents {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "session-") || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		if st, err := e.Info(); err == nil && st.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

func openBrowser(u string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	case "darwin":
		return exec.Command("open", u).Start()
	default:
		return exec.Command("xdg-open", u).Start()
	}
}

func openFolder(path string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer.exe", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func fatalDialog(title, message string) {
	if runtime.GOOS == "windows" {
		script := fmt.Sprintf(`Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show(%q,%q)`, message, title)
		_ = exec.Command("powershell.exe", "-NoProfile", "-Command", script).Run()
		return
	}
	log.Printf("%s: %s", title, message)
}

var _ = strconv.Itoa
