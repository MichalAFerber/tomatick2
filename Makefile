APP := tomatick
PKG := ./cmd/tomatick

# VERSION comes from internal/version/version.go, which is the single source of
# truth. It is a `const`, so `-ldflags -X` CANNOT inject it -- the linker accepts
# the flag, silently does nothing, and the binary reports the old number. The
# version therefore ships by being compiled in, and `make version-check` is what
# stops the constant, FyneApp.toml and the git tag from drifting apart.
#
# Lazy `=`, not `:=`, on purpose: an override (`make VERSION=2.0.0 package-windows`)
# then skips the shell call entirely, which is what lets the Windows runner --
# where make's default shell is cmd.exe, not sh -- use these same targets.
VERSION = $(shell sed -n 's/^const Version = "\(.*\)"$$/\1/p' internal/version/version.go)

.PHONY: test build run tidy version-check package-mac package-linux package-windows

test:
	go test ./...

build:
	go build -o $(APP) $(PKG)

run: build
	./$(APP)

tidy:
	go mod tidy

# Fails when the version constant, FyneApp.toml, and -- when EXPECT_VERSION is
# set, as .github/workflows/release.yml sets it from the pushed tag -- the tag
# itself disagree. Requires a POSIX shell, so it is a release-workflow gate step
# run on Linux rather than a prerequisite of the package-* targets.
version-check:
	@v='$(VERSION)'; \
	 t=$$(sed -n 's/^Version = "\(.*\)"$$/\1/p' cmd/tomatick/FyneApp.toml); \
	 test -n "$$v" || { echo "version-check: no Version const in internal/version/version.go"; exit 1; }; \
	 test "$$v" = "$$t" || { echo "version-check: FyneApp.toml has '$$t', version.go has '$$v'"; exit 1; }; \
	 if [ -n "$(EXPECT_VERSION)" ] && [ "$(EXPECT_VERSION)" != "$$v" ]; then \
	   echo "version-check: tag '$(EXPECT_VERSION)' does not match version.go '$$v'"; exit 1; \
	 fi; \
	 echo "version-check: OK ($$v)"

# fyne package produces a platform-native bundle. Run on the target OS: every GUI
# dependency here is cgo, so there is no cross-compilation. After packaging on
# macOS, LSUIElement is patched so the app lives in the menu bar without a Dock icon.
#
# FyneApp.toml and Icon.png live beside the main package, not at the repo root,
# because fyne resolves BOTH relative to -src: it loads <srcDir>/FyneApp.toml, and
# it rewrites a literal `--icon Icon.png` to <srcDir>/Icon.png. With them at the
# root the metadata was silently ignored and the icon lookup failed outright.
#
# The flags are --appID and --appVersion, NOT --app-id/--app-version: the fyne CLI
# rejects the hyphenated forms with `flag provided but not defined` (run 33951169730).
package-mac: build
	fyne package -os darwin -src cmd/tomatick --name Tomatick --appID us.tomatick --appVersion $(VERSION) --icon Icon.png
	plutil -replace LSUIElement -bool true Tomatick.app/Contents/Info.plist

package-linux:
	fyne package -os linux -src cmd/tomatick --name Tomatick --appID us.tomatick --appVersion $(VERSION) --icon Icon.png

package-windows:
	fyne package -os windows -src cmd/tomatick --name Tomatick --appID us.tomatick --appVersion $(VERSION) --icon Icon.png
