# AGENTS.md — Developer Guide for AI Agents (Mobile)

## Project Overview

Mobile (Android + iOS) companion app for the Keenetic Tray project.
Written in Go using Fyne v2. Single binary per platform, no web views, no JavaScript.

## Key Differences from Desktop (`../app/`)

| Aspect | Desktop | Mobile |
|---|---|---|
| Entry point | System tray (`desktop.App`) | Full-screen Fyne window |
| Device detection | MAC address via `net.Interfaces()` | IP address (MAC restricted on iOS) |
| Tray icon | Dynamic image/draw icon | N/A |
| Gateway detection | `github.com/jackpal/gateway` | Not used (removed) |
| Password storage | Keyring (all backends) | Keychain (iOS) / JSON fallback (Android) |

## File Map

| File | Responsibility |
|---|---|
| `main.go` | Creates Fyne app, single window, calls `newMainUI` |
| `config.go` | `RouterConfig` (with `Password` JSON fallback field), `loadRouters/saveRouters`, `getPassword/setPassword/deletePassword` |
| `router.go` | Full Keenetic HTTP API: login (MD5+SHA256), get clients, get policies, apply policy, block client, `PolicyLabel` helper |
| `network.go` | `getLocalNetworks`, `isIPInNetworks`, `extractHost`, `getLocalIPs` (no gateway), `FindThisDevice` (IP-based match) |
| `ui_main.go` | `MainUI` struct — main screen: status card, device card, policy radio group, apply button, refresh |
| `ui_settings.go` | `showSettingsWindow` (router list), `showRouterForm`, `showRouterFormWithValues` (retry with error) |

## Key Data Flows

### Startup
```
main() → newMainUI() → content() → go refresh()
```

### Refresh
```
refresh() → collectState() → findReachableRouter() → GetOnlineClients() → FindThisDevice() → applyState()
```

### Policy change
```
applyBtn.OnTapped → onApply() → resolve policyKey from selected label
→ go router.SetClientBlock(mac) or router.ApplyPolicy(mac, key) → refresh()
```

### Add router
```
Settings → showRouterForm() → go router.Login() + GetNetworkIP()
→ setPassword() → onConfirm() → saveRouters() → go refresh()
```

## Password Storage Strategy

```
globalRing != nil  →  iOS Keychain / Linux SecretService
globalRing == nil  →  cfg.Password field stored in JSON (Android, or when no keyring available)
```

`setPassword()` always clears `cfg.Password` when keyring succeeds.

## Device Matching

`FindThisDevice(clients []Client)` compares the router's client IP list against
`getLocalIPs()` which uses `net.Interfaces()`. This works on both platforms without
needing MAC addresses (which iOS randomizes since iOS 14).

## Keyring Backends

Only `KeychainBackend` (iOS/macOS) and `SecretServiceBackend` (Linux, for dev) are
listed in `AllowedBackends`. `WinCredBackend` is intentionally excluded (mobile-only module).

## Build Requirements

| Platform | Toolchain |
|---|---|
| Android | Go + Android NDK r27+ (`ANDROID_NDK_HOME`) + Java 17 + `fyne` CLI |
| iOS | Go + macOS + Xcode 15+ + `fyne` CLI |
| Desktop (dev) | Go + GCC (just for UI testing) |

## Common Pitfalls

- **Fyne UI from goroutines** — `applyState` and `setLoading` are called after the goroutine completes. Fyne generally handles cross-goroutine widget updates safely, but avoid calling `window.SetContent` from a goroutine.
- **Android keyring** — `WinCredBackend` / `KWalletBackend` are not listed; `globalRing` will be `nil` on Android → passwords go into JSON. This is intentional.
- **iOS MAC randomization** — Never use `net.Interface.HardwareAddr` for device matching on iOS. Always use IP.
- **`fyne package -os android`** — must be run on Linux or macOS. The CI uses `ubuntu-latest`.
- **`fyne package -os ios`** — must be run on macOS. The CI uses `macos-latest`.
- **Unsigned APK** — Android will warn on install. For distribution, sign with `apksigner`.
- **Unsigned iOS** — Requires sideloading via Xcode or AltStore. App Store requires a paid developer account.
