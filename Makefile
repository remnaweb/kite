.PHONY: dev panel web build xray tidy install

dev:
	@echo "Start two processes: make panel  and  make web"

panel:
	go run ./cmd/panel

web:
	cd web && npm run dev

xray:
	bash scripts/download-xray.sh

build:
	cd web && npm run build
	rm -rf internal/webui/dist
	mkdir -p internal/webui/dist
	cp -a web/dist/. internal/webui/dist/
	go build -o bin/panel ./cmd/panel

tidy:
	go mod tidy

install:
	bash install.sh
