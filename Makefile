.PHONY: serve import test build

DB := progress/go-learn.db

serve:
	go run ./cmd/server

import:
	go run ./cmd/import-content

test:
	go test ./...

build:
	go build -o bin/server ./cmd/server
	go build -o bin/import-content ./cmd/import-content

fresh-import:
	rm -f $(DB)
	$(MAKE) import

# === Cloudflare Workers deployment ===

# Build WASM binary for Workers
worker:
	@mkdir -p build
	GOOS=js GOARCH=wasm go build -o build/app.wasm ./cmd/worker
	@ls -lh build/app.wasm

# Generate wrangler scaffolding from syumai/workers template
worker-init:
	npm create cloudflare@latest -- --template github.com/syumai/workers/_templates/cloudflare/worker-go workers-go -- --accept-defaults

# Deploy to Cloudflare Workers (requires wrangler login)
deploy: worker
	wrangler deploy

# Preview locally
preview: worker
	wrangler dev
