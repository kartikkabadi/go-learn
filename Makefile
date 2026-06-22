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
