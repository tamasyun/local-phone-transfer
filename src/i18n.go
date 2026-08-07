package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var localeNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var localization = struct {
	sync.RWMutex
	lang string
	text map[string]string
}{lang: "ja", text: map[string]string{}}

type localeConfig struct {
	Locale string `json:"locale"`
}

func init() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	baseDir := filepath.Dir(exe)
	requested := "ja"
	if b, err := os.ReadFile(filepath.Join(baseDir, "transfer-config.json")); err == nil {
		var cfg localeConfig
		if json.Unmarshal(b, &cfg) == nil && strings.TrimSpace(cfg.Locale) != "" {
			requested = cfg.Locale
		}
	}
	_ = loadLocale(baseDir, requested)
}

func loadLocale(baseDir, requested string) error {
	lang := strings.TrimSpace(requested)
	if lang == "" {
		lang = "ja"
	}
	if !localeNamePattern.MatchString(lang) {
		return fmt.Errorf("invalid locale: %s", lang)
	}

	path := filepath.Join(baseDir, "locales", lang+".json")
	b, err := os.ReadFile(path)
	if err != nil && lang != "ja" {
		lang = "ja"
		path = filepath.Join(baseDir, "locales", "ja.json")
		b, err = os.ReadFile(path)
	}
	if err != nil {
		return fmt.Errorf("locale file could not be loaded: %w", err)
	}

	m := map[string]string{}
	if err := json.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("locale JSON is invalid: %w", err)
	}

	localization.Lock()
	localization.lang = lang
	localization.text = m
	localization.Unlock()
	return nil
}

func t(key string) string {
	localization.RLock()
	v := localization.text[key]
	localization.RUnlock()
	if v == "" {
		return key
	}
	return v
}

func tf(key string, args ...any) string {
	return fmt.Sprintf(t(key), args...)
}

func currentLang() string {
	localization.RLock()
	defer localization.RUnlock()
	return localization.lang
}
