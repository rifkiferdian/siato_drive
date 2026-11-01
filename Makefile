.PHONY: run dev build test vet fmt migrate seed css
run:
	go run ./cmd/server
dev:
	air -c .air.toml
build:
	go build -o bin/cloud-drive ./cmd/server
test:
	go test ./...
vet:
	go vet ./...
fmt:
	gofmt -w cmd internal
migrate:
	go run ./cmd/migrate
seed:
	go run ./cmd/seed
css:
	tailwindcss -i assets/css/input.css -o web/static/css/app.css --minify
