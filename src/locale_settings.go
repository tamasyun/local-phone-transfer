package main

import (
    "encoding/json"
    "net/http"
    "os"
    "path/filepath"
    "strings"
)

func (a *App) adminLocale(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    lang := strings.TrimSpace(r.FormValue("locale"))
    if lang != "ja" && lang != "en" {
        http.Error(w, "unsupported locale", http.StatusBadRequest)
        return
    }

    configPath := filepath.Join(a.exeDir, "transfer-config.json")
    b, err := os.ReadFile(configPath)
    if err != nil {
        http.Error(w, "could not read settings", http.StatusInternalServerError)
        return
    }
    raw := map[string]any{}
    if err := json.Unmarshal(b, &raw); err != nil {
        http.Error(w, "settings JSON is invalid", http.StatusInternalServerError)
        return
    }
    raw["locale"] = lang
    out, err := json.MarshalIndent(raw, "", "  ")
    if err != nil {
        http.Error(w, "could not encode settings", http.StatusInternalServerError)
        return
    }
    out = append(out, '\n')
    if err := os.WriteFile(configPath, out, 0644); err != nil {
        http.Error(w, "could not save settings", http.StatusInternalServerError)
        return
    }
    if err := loadLocale(a.exeDir, lang); err != nil {
        http.Error(w, "could not load locale", http.StatusInternalServerError)
        return
    }
    a.touchActivity("admin_locale_change")
    w.WriteHeader(http.StatusNoContent)
}
