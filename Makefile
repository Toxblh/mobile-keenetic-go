APP_NAME  := keenetic-tray
APP_ID    := com.keenetic.tray
BUILD_DIR := bin

# Requires: go install fyne.io/fyne/v2/cmd/fyne@latest

.PHONY: all android ios desktop clean fyne-cli

## Default: build Android APK
all: android

## Install the fyne CLI tool (run once)
fyne-cli:
	go install fyne.io/fyne/v2/cmd/fyne@latest

## Build Android APK
## Requires: Android NDK, ANDROID_NDK_HOME set
android: fyne-cli
	fyne package \
		-os android \
		-appID $(APP_ID) \
		-name "$(APP_NAME)"
	mkdir -p $(BUILD_DIR)
	mv "$(APP_NAME).apk" $(BUILD_DIR)/

## Build iOS app bundle
## Requires: macOS + Xcode + Apple Developer account for device distribution
ios: fyne-cli
	fyne package \
		-os ios \
		-appID $(APP_ID) \
		-name "$(APP_NAME)"
	mkdir -p $(BUILD_DIR)
	mv "$(APP_NAME).app" $(BUILD_DIR)/

## Run on the current desktop OS (for development / testing UI)
desktop:
	CGO_ENABLED=1 go run .

## Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
