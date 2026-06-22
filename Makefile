GREEN := \033[0;32m
RED := \033[0;31m
NC := \033[0m

COVER_EXCLUDE ?= internal/shared/proto|/cmd/|/db/conn/mock|/migrations|/internal/server/model|/internal/client/model

BIN_DIR ?= bin
GRPC_ADDR ?= localhost:8080
LDFLAGS := -X main.grpcServerAddr=$(GRPC_ADDR)

.PHONY: help proto certs build build-server build-client run-server test test-integration test-coverpkg test-short coverage coverage-percent coverage-packages fmt fmt-check vet check clean

help: ## Показать справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

proto: ## Сгенерировать protobuf
	protoc -I api \
		--go_out=. --go_opt=module=gophkeeper --go_opt=default_api_level=API_OPAQUE \
		--go-grpc_out=. --go-grpc_opt=module=gophkeeper \
		$(shell find api -name '*.proto')

certs: ## Сгенерировать self-signed TLS-сертификаты (если их нет)
	@echo "$(GREEN)Ensuring TLS certificates...$(NC)"
	go run ./cmd/certgen

build: build-server build-client ## Собрать сервер и клиент под текущую платформу

build-server: certs ## Собрать бинарь сервера
	go build -o $(BIN_DIR)/server ./cmd/server

build-client: certs ## Собрать бинарь клиента (серт встраивается через go:embed)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/client ./cmd/client

run-server: certs ## Запустить сервер (секреты задаются через config.json или env)
	go run ./cmd/server

test: certs ## Запустить все тесты (без integration) и собрать покрытие
	@echo "$(GREEN)Running tests...$(NC)"
	go test -race -count=1 -coverprofile=coverage.out $$(go list ./... | grep -vE '$(COVER_EXCLUDE)')
	@echo "$(GREEN)Tests passed$(NC)"

test-integration: certs ## Тесты с тегом integration (нужен Docker)
	@echo "$(GREEN)Running tests (integration)...$(NC)"
	go test -race -count=1 -tags=integration -coverprofile=coverage.out $$(go list ./... | grep -vE '$(COVER_EXCLUDE)')
	@echo "$(GREEN)Tests passed$(NC)"

test-coverpkg: certs ## Тесты с -coverpkg по всем тестируемым пакетам
	@PKGS=$$(go list ./... | grep -vE '$(COVER_EXCLUDE)'); \
	COVERPKG=$$(echo "$$PKGS" | tr '\n' ',' | sed 's/,$$//'); \
	go test -race -count=1 -coverpkg="$$COVERPKG" -coverprofile=coverage.out $$PKGS

test-short: certs ## Быстрые тесты
	go test -short ./...

coverage: test ## HTML-отчёт покрытия
	go tool cover -html=coverage.out

coverage-percent: ## Общий процент покрытия
	@if [ ! -f coverage.out ]; then echo "$(RED)coverage.out не найден, запустите make test$(NC)"; exit 1; fi
	@go tool cover -func=coverage.out | grep total | awk '{printf "  Всего: $(GREEN)%s$(NC)\n", $$3}'

coverage-packages: certs ## Процент покрытия по пакетам
	go test -cover $$(go list ./... | grep -vE '$(COVER_EXCLUDE)')

fmt: ## Форматировать код
	gofmt -w $$(go list -f '{{.Dir}}' ./...)

fmt-check: ## Проверить форматирование
	@out=$$(gofmt -l $$(go list -f '{{.Dir}}' ./...)); \
	if [ -n "$$out" ]; then echo "$(RED)not formatted:$(NC)"; echo "$$out"; exit 1; fi

vet: certs ## Запустить go vet
	go vet ./...

check: fmt-check vet test ## Полная проверка: формат + vet + тесты

clean: ## Удалить артефакты сборки, покрытия и сгенерированные сертификаты
	rm -f coverage.out cover.out cover.nogen.out coverage.html
	rm -rf $(BIN_DIR) certs internal/client/tlsclient/cert
