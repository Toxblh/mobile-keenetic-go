package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"github.com/99designs/keyring"
)

// RouterConfig holds a configured router entry.
// Password is stored in the OS keychain when available,
// falling back to the JSON file (acceptable for a LAN-only tool).
type RouterConfig struct {
	Name      string   `json:"name"`
	Address   string   `json:"address"`
	Login     string   `json:"login"`
	Password  string   `json:"password,omitempty"` // fallback when keyring unavailable
	NetworkIP string   `json:"network_ip,omitempty"`
	KeenDNS   []string `json:"keendns_urls,omitempty"`
}

var globalRing keyring.Keyring

func init() {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: "keenetic-tray",
		AllowedBackends: []keyring.BackendType{
			keyring.KeychainBackend,      // iOS / macOS
			keyring.SecretServiceBackend, // Linux desktop (testing)
		},
	})
	if err == nil {
		globalRing = ring
	}
}

func configDir() string {
	switch runtime.GOOS {
	case "darwin", "ios":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "RouterManager")
	case "android":
		// Fyne sets HOME to the app's private data dir on Android
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "RouterManager")
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".router_manager")
	}
}

func configPath() string {
	dir := configDir()
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "routers.json")
}

func loadRouters() []RouterConfig {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil
	}
	var routers []RouterConfig
	if err := json.Unmarshal(data, &routers); err != nil {
		return nil
	}
	return routers
}

func saveRouters(routers []RouterConfig) error {
	data, err := json.MarshalIndent(routers, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0o644)
}

func getPassword(cfg *RouterConfig) string {
	if globalRing != nil {
		if item, err := globalRing.Get(cfg.Name); err == nil {
			return string(item.Data)
		}
	}
	return cfg.Password
}

func setPassword(cfg *RouterConfig, password string) {
	if globalRing != nil {
		err := globalRing.Set(keyring.Item{
			Key:   cfg.Name,
			Data:  []byte(password),
			Label: "Keenetic Tray — " + cfg.Name,
		})
		if err == nil {
			cfg.Password = "" // don't duplicate into JSON
			return
		}
	}
	cfg.Password = password // keyring unavailable — store in JSON
}

func deletePassword(cfg *RouterConfig) {
	if globalRing != nil {
		_ = globalRing.Remove(cfg.Name)
	}
	cfg.Password = ""
}
