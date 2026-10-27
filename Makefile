.PHONY: run test vet fmt fmt-check build image monitoring-up monitoring-down app-up app-down

GOIMPORTS := go run golang.org/x/tools/cmd/goimports@v0.28.0 -local multithreadredis

run:
	go run ./cmd/server

test:
	go test -race ./...

vet:
	go vet ./...

# Sort, group and clean imports, and gofmt every Go file
fmt:
	$(GOIMPORTS) -w .

# Fail if any Go file is not goimports-formatted
fmt-check:
	@out=$$($(GOIMPORTS) -l .); if [ -n "$$out" ]; then echo "Run 'make fmt' on:"; echo "$$out"; exit 1; fi

build:
	go build -o bin/server ./cmd/server

image:
	docker build -f build/Dockerfile -t multithreaded-redis:local .

# Local monitoring stack: Grafana on http://localhost:3000
monitoring-up:
	docker compose -f deploy/monitoring/compose.yml up -d

monitoring-down:
	docker compose -f deploy/monitoring/compose.yml down

# Local app stack, built from source
app-up:
	docker compose -f deploy/app/compose.yml -f deploy/app/compose.dev.yml up -d --build

app-down:
	docker compose -f deploy/app/compose.yml -f deploy/app/compose.dev.yml down
