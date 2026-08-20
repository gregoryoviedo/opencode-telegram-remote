.PHONY: app build icons clean

ROOT := $(shell pwd)
APP_DIR := $(ROOT)/macos/OpenCodeRemote

app: build
	@echo ""
	@echo "✓ App lista en: $(ROOT)/dist/OpenCodeRemote.app"
	@echo ""
	@echo "Para instalar:"
	@echo "    cp -R $(ROOT)/dist/OpenCodeRemote.app /Applications/"
	@echo ""
	@echo "Para abrir (primera vez desbloquea Gatekeeper):"
	@echo "    xattr -dr com.apple.quarantine $(ROOT)/dist/OpenCodeRemote.app"
	@echo "    open $(ROOT)/dist/OpenCodeRemote.app"

build:
	bash $(APP_DIR)/scripts/build.sh

icons:
	bash $(APP_DIR)/scripts/make-icon.sh

clean:
	rm -rf $(APP_DIR)/build $(APP_DIR)/OpenCodeRemote.xcodeproj $(ROOT)/dist
	rm -rf $(APP_DIR)/OpenCodeRemote/Resources/remote-bot