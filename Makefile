APP := tomatick
PKG := ./cmd/tomatick

.PHONY: test build run tidy package-mac package-linux package-windows

test:
	go test ./...

build:
	go build -o $(APP) $(PKG)

run: build
	./$(APP)

tidy:
	go mod tidy

# fyne package produces a platform-native bundle. Run on the target OS
# (or use fyne-cross). After packaging on macOS, LSUIElement is patched
# so the app lives in the menu bar without a Dock icon.
package-mac: build
	fyne package -os darwin -src cmd/tomatick --name Tomatick --app-id us.tomatick --app-version 0.5.0 --icon Icon.png
	plutil -replace LSUIElement -bool true Tomatick.app/Contents/Info.plist

package-linux:
	fyne package -os linux -src cmd/tomatick --name Tomatick --app-id us.tomatick --app-version 0.5.0 --icon Icon.png

package-windows:
	fyne package -os windows -src cmd/tomatick --name Tomatick --app-id us.tomatick --app-version 0.5.0 --icon Icon.png
