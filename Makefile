COMPOSE ?= docker-compose
POSTGRES_USER ?= aprova
POSTGRES_PASSWORD ?= aprova
POSTGRES_DB ?= aprova
POSTGRES_PORT ?= 5432
DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable
MIGRATE ?= migrate
SQLC ?= sqlc

# A URL vai para o ambiente dos comandos em vez de ser colada na linha do
# shell, para que aspas ou metacaracteres na senha não virem comando.
export DATABASE_URL

.PHONY: up down migrate migrate-up migrate-down migrate-reset sqlc test-banco run-api run-worker test lint keys seed-demo smoke-test verificar-token sandbox-azure consultar-permissao imagem-sandbox-go imagem-sandbox-python imagem-sandbox-java imagem-api imagem-worker

up:
	$(COMPOSE) up -d --wait

down:
	$(COMPOSE) down

migrate: migrate-up

migrate-up:
	$(MIGRATE) -path migrations -database "$$DATABASE_URL" up

migrate-down:
	$(MIGRATE) -path migrations -database "$$DATABASE_URL" down 1

migrate-reset:
	@if [ "$(CONFIRMAR)" != "sim" ]; then \
		echo "migrate-reset apaga todas as tabelas do banco configurado"; \
		echo "para confirmar: make migrate-reset CONFIRMAR=sim"; \
		exit 1; \
	fi
	$(MIGRATE) -path migrations -database "$$DATABASE_URL" down -all

sqlc:
	$(SQLC) generate

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

keys:
	@go run ./scripts/gerar-chaves

seed-demo:
	@go run ./scripts/seed-demo $(if $(INSTALACAO),-instalacao $(INSTALACAO))

smoke-test:
	@scripts/smoke-test.sh $(or $(URL),http://localhost:8080)

verificar-token:
	@go run ./scripts/verificar-token $(INSTALLATION_ID)

sandbox-azure: export REPOSITORIO := $(REPOSITORIO)
sandbox-azure: export SHA := $(SHA)
sandbox-azure: export DIRETORIO := $(DIRETORIO)
sandbox-azure:
	@go run ./scripts/disparar-sandbox-azure "$$REPOSITORIO" "$$SHA" "$$DIRETORIO"

imagem-sandbox-go:
	docker build -f imagens/go/Dockerfile -t aprova-sandbox-go:local .

imagem-api:
	docker build -f imagens/nucleo/Dockerfile --target api -t aprova-api:local .

imagem-worker:
	docker build -f imagens/nucleo/Dockerfile --target worker -t aprova-worker:local .

imagem-sandbox-python:
	docker build -f imagens/python/Dockerfile -t aprova-sandbox-python:local .

imagem-sandbox-java:
	docker build -f imagens/java/Dockerfile -t aprova-sandbox-java:local .

consultar-permissao:
	@go run ./scripts/consultar-permissao $(INSTALLATION_ID) $(REPOSITORIO) $(LOGIN)

test:
	go test ./...

test-banco:
	go test ./tests/integration/postgres/... -race -count=1 -v

lint:
	golangci-lint run

.PHONY: build-web build-api-web
build-web:
	npm --prefix web ci
	npm --prefix web run build

build-api-web: build-web
	go build -o bin/api ./cmd/api
