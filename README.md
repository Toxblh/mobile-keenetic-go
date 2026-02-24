# Keenetic Tray — Mobile

Android and iOS companion app for managing Keenetic routers.
Detects the active router on the current Wi-Fi network, shows the current
access policy for this device, and lets you switch policies with a tap.

## Features

- Auto-detects a configured Keenetic router on the current Wi-Fi
- Identifies this device by its IP address (no MAC address required)
- Switch between Default, Blocked, or any named access policy
- Router credentials stored in iOS Keychain (Android: secure local file)
- Pure Go — no React Native, no JavaScript

## Screens

| Main screen | Settings |
|---|---|
| Router status, device info, policy selector | Add / Edit / Delete routers |

## Installation

### Android

Download `keenetic-tray.apk` from [Releases](../../releases).
Enable "Install from unknown sources" in Android settings, then open the APK.

> The APK is unsigned. To install on recent Android versions without warnings,
> you can sign it with your own debug keystore (`apksigner`).

### iOS

iOS distribution without the App Store requires an Apple Developer account
and code signing. The CI produces an unsigned `.app` bundle for sideloading
via Xcode or tools like AltStore.

Download `keenetic-tray-ios.zip` from [Releases](../../releases), unzip, and
install via Xcode → Devices & Simulators → Install App.

## Building from Source

### Requirements

- Go 1.21+
- [Fyne CLI](https://docs.fyne.io/started/): `go install fyne.io/fyne/v2/cmd/fyne@latest`
- **Android**: Android NDK r27+, `ANDROID_NDK_HOME` set
- **iOS**: macOS + Xcode 15+

### Build

```bash
# Android APK
make android

# iOS app bundle (macOS only)
make ios

# Run on desktop for UI development / testing
make desktop
```

## Device Detection

On mobile, MAC addresses are restricted (iOS randomizes them).
This app identifies the current device using its **IP address** instead.

This means the device must have a stable IP (DHCP reservation recommended)
or be online when the app is opened.

## Project Structure

```
mobile/
├── main.go           Entry point
├── config.go         Config (JSON) + keyring/fallback password storage
├── router.go         Keenetic HTTP API client
├── network.go        IP-based device detection (no MAC needed)
├── ui_main.go        Main screen — status, policy selector
├── ui_settings.go    Settings screen — manage routers
├── Makefile
└── .github/workflows/
    ├── build-android.yml
    ├── build-ios.yml
    └── release.yml
```

## CI / CD

- **Every commit** → builds APK (Android) + `.app` (iOS), artifacts for **14 days**
- **Tag `v*`** → GitHub Release with APK and iOS bundle attached

```bash
git tag v1.0.0
git push origin v1.0.0
```

## Tech Stack

| Layer | Library |
|---|---|
| UI | [Fyne v2](https://fyne.io/) |
| HTTP client | `net/http` (stdlib) |
| Password store | [99designs/keyring](https://github.com/99designs/keyring) (iOS Keychain), fallback to JSON |
| Network | `net` stdlib — IP-based detection |

## License

MIT
