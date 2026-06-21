GREEN := \033[0;32m
RED := \033[0;31m
NC := \033[0m

COVER_EXCLUDE ?= internal/shared/proto|/cmd/|/db/conn/mock|/migrations|/internal/server/model|/internal/client/model

.PHONY: help proto test test-integration test-coverpkg test-short coverage coverage-percent coverage-packages fmt fmt-check vet check clean

help: ## Показать справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

proto: ## Сгенерировать protobuf
	protoc -I api \
		--go_out=. --go_opt=module=gophkeeper --go_opt=default_api_level=API_OPAQUE \
		--go-grpc_out=. --go-grpc_opt=module=gophkeeper \
		$(shell find api -name '*.proto')

test: ## Запустить все тесты (без integration) и собрать покрытие
	@echo "$(GREEN)Running tests...$(NC)"
	go test -race -count=1 -coverprofile=coverage.out $$(go list ./... | grep -vE '$(COVER_EXCLUDE)')
	@echo "$(GREEN)Tests passed$(NC)"

test-integration: ## Тесты с тегом integration (нужен Docker)
	@echo "$(GREEN)Running tests (integration)...$(NC)"
	go test -race -count=1 -tags=integration -coverprofile=coverage.out $$(go list ./... | grep -vE '$(COVER_EXCLUDE)')
	@echo "$(GREEN)Tests passed$(NC)"

test-coverpkg: ## Тесты с -coverpkg по всем тестируемым пакетам
	@PKGS=$$(go list ./... | grep -vE '$(COVER_EXCLUDE)'); \
	COVERPKG=$$(echo "$$PKGS" | tr '\n' ',' | sed 's/,$$//'); \
	go test -race -count=1 -coverpkg="$$COVERPKG" -coverprofile=coverage.out $$PKGS

test-short: ## Быстрые тесты
	go test -short ./...

coverage: test ## HTML-отчёт покрытия
	go tool cover -html=coverage.out

coverage-percent: ## Общий процент покрытия
	@if [ ! -f coverage.out ]; then echo "$(RED)coverage.out не найден, запустите make test$(NC)"; exit 1; fi
	@go tool cover -func=coverage.out | grep total | awk '{printf "  Всего: $(GREEN)%s$(NC)\n", $$3}'

coverage-packages: ## Процент покрытия по пакетам
	go test -cover $$(go list ./... | grep -vE '$(COVER_EXCLUDE)')

fmt: ## Форматировать код
	gofmt -w $$(go list -f '{{.Dir}}' ./...)

fmt-check: ## Проверить форматирование
	@out=$$(gofmt -l $$(go list -f '{{.Dir}}' ./...)); \
	if [ -n "$$out" ]; then echo "$(RED)not formatted:$(NC)"; echo "$$out"; exit 1; fi

vet: ## Запустить go vet
	go vet ./...

check: fmt-check vet test ## Полная проверка: формат + vet + тесты

clean: ## Удалить артефакты покрытия
	rm -f coverage.out cover.out cover.nogen.out coverage.html
