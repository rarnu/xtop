package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// dict holds the loaded translation strings. It is populated once at package
// init time by resolving the system locale and searching the language directories
// in priority order.
var dict map[string]string

func init() {
	dict = loadDict()
}

// T returns the translation for key, or the key itself if no translation exists.
func T(key string) string {
	if v, ok := dict[key]; ok {
		return v
	}
	return key
}

// Tf formats a translated string with fmt.Sprintf.
func Tf(key string, args ...interface{}) string {
	return fmt.Sprintf(T(key), args...)
}

// loadDict resolves the best locale and returns the merged translation map.
// It falls back to en_US when the requested locale is unavailable, and finally
// to a built-in minimal English table if no language file can be loaded.
func loadDict() map[string]string {
	locale := systemLocale()
	paths := langFilePaths(locale)
	for _, p := range paths {
		if d, ok := loadLangFile(p); ok {
			return d
		}
	}

	// Fallback to en_US.
	for _, p := range langFilePaths("en_US") {
		if d, ok := loadLangFile(p); ok {
			return d
		}
	}

	return builtinDict()
}

// systemLocale extracts a locale code like zh_CN or en_US from the environment.
func systemLocale() string {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(k); v != "" {
			return normalizeLocale(v)
		}
	}
	return "en_US"
}

// normalizeLocale converts values such as "zh_CN.UTF-8" or "en_US.utf8" into
// canonical "zh_CN"/"en_US" codes.
func normalizeLocale(s string) string {
	if i := strings.IndexAny(s, ".@"); i >= 0 {
		s = s[:i]
	}
	if s == "C" || s == "POSIX" {
		return "en_US"
	}
	return s
}

// langFilePaths returns candidate language file paths for locale in search order.
func langFilePaths(locale string) []string {
	home, _ := os.UserHomeDir()
	name := locale + ".json"
	return []string{
		filepath.Join("/etc", "xtop", "lang", name),
		filepath.Join(home, ".xtop", "lang", name),
		filepath.Join("lang", name),
	}
}

func loadLangFile(path string) (map[string]string, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var out map[string]string
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, false
	}
	return out, true
}

// builtinDict is the last-resort fallback so the UI never shows raw keys when
// all language files are missing.
func builtinDict() map[string]string {
	return map[string]string{
		"card.cpu":                "CPU",
		"card.mem":                "MEM",
		"card.disk":               "DISK",
		"card.gpu":                "GPU",
		"card.net":                "NETWORK",
		"card.proc":               "PROCESSES",
		"proc.open_manager":       "Open Process Manager",
		"proc.modal.title":        "Process Manager",
		"proc.modal.count":        "%d processes",
		"proc.modal.help":         "↑/↓ select · 1-7/click header sort · k kill · K/f force kill · ESC back",
		"proc.detail.title":       "Process Detail",
		"proc.detail.command":     "Command",
		"proc.detail.user":        "User",
		"proc.detail.status":      "Status",
		"proc.detail.cpu":         "CPU",
		"proc.detail.mem":         "Memory",
		"proc.detail.net":         "Network",
		"proc.detail.disk":        "Disk",
		"proc.detail.gpu":         "GPU Mem",
		"proc.detail.terminate":   "Kill",
		"proc.detail.force_kill":  "Force Kill",
		"confirm.terminate":       "Kill process?",
		"confirm.force_kill":      "Force kill process?",
		"confirm.keys":            "y confirm   n/ESC cancel",
		"no_data.cpu":             "No CPU data",
		"no_data.mem":             "No memory data",
		"no_data.disk":            "No disk data",
		"no_data.gpu":             "No GPU data",
		"no_data.net_procs":       "No process data",
		"footer.help":             "P/Enter process manager · a about · scroll with mouse · q quit",
		"about.github":            "GitHub",
		"about.license":           "License",
		"about.go":                "Go",
		"about.build":             "Build",
		"about.ok":                "OK",
		"label.used":              "Used",
		"label.cached":            "Cached",
		"label.free":              "Free",
		"label.speed":             "Speed",
		"label.total_used":        "Total",
		"label.upload":            "Upload",
		"label.download":          "Download",
		"label.read":              "Read/s",
		"label.write":             "Write/s",
		"label.type":              "Type",
		"label.power":             "Power",
		"label.memory":            "Memory",
		"label.temperature":       "Temp",
		"label.load":              "Load",
		"label.gpu_mem":           "VRAM",
		"unknown":                 "unknown",
		"na":                      "N/A",
		"init":                    "Initializing XTOP...",
	}
}
