![Покрытие тестами](.badges/coverage.svg)

# GophKeeper

Клиент-серверный менеджер секретов: надёжно и безопасно хранит логины/пароли, произвольные текстовые и бинарные данные, а также данные банковских карт — с произвольной текстовой метаинформацией к каждой записи. Клиент — CLI/TUI под Windows, Linux и macOS; сервер хранит данные и синхронизирует их между несколькими устройствами одного владельца. Весь обмен идёт по gRPC поверх обязательного TLS.

Проект из курса «Go-разработчик» (Яндекс Практикум), второй выпускной проект.

## Что умеет

- регистрация, аутентификация и авторизация пользователей (JWT: access + refresh);
- хранение четырёх типов данных, каждый — с произвольной метаинформацией:
  - пары логин/пароль,
  - произвольный текст,
  - произвольные бинарные данные (файлы),
  - банковские карты;
- синхронизация данных между несколькими авторизованными клиентами одного владельца;
- выдача приватных данных владельцу по запросу;
- терминальный интерфейс (TUI);
- информация о версии и дате сборки клиента (`--version`).

## Как это устроено

```
┌──────────────────┐     gRPC + TLS (обязательный)     ┌─────────────────┐
│  client (TUI)    │ ────────────────────────────────► │   server         │
│  cmd/client      │                                    │   cmd/server     │
│  локальный кэш    │ ◄──────── синхронизация ──────────│                 │
│  SQLite (pure-Go)│                                    └────────┬────────┘
└──────────────────┘                                             │ пишет в
                                                                  ▼
                                                              PostgreSQL
```

- **Клиент** (`cmd/client`) — интерактивный TUI (bubbletea). Держит локальный кэш в SQLite (чистый Go-драйвер, без CGO) и синхронизирует его с сервером.
- **Сервер** (`cmd/server`) — gRPC-сервис, хранит данные в PostgreSQL.
- **certgen** (`cmd/certgen`) — утилита генерации self-signed TLS-пары. Публичный сертификат вшивается в клиентский бинарь через `go:embed`, приватный ключ остаётся только на сервере.

## Безопасность

- TLS **обязателен** с обеих сторон — plaintext-транспорта нет.
- Сертификат self-signed: клиент пинит публичный сертификат сервера как доверенный корневой (вшит в бинарь), приватный ключ в git и в бинари не попадает.
- Секреты сервера (`JWT_SECRET`, `REFRESH_SECRET`) задаются через env или config-файл, в коде не хранятся.

## Быстрый старт

Требуется: **Go 1.26+**, **Docker** (для PostgreSQL).

```bash
# 1. Сгенерировать TLS-сертификаты (один раз; make build/run сделают это сами)
make certs

# 2. Поднять PostgreSQL в Docker
make db-up

# 3. Запустить сервер (секреты — через config.json или env, см. ниже)
make run-server

# 4. В другом терминале — клиент
make run-client
```

`make run-server` сам зависит от `certs` и `db-up`, так что достаточно одной команды.

## Сборка бинарников

```bash
make build               # сервер и клиент под текущую платформу → bin/server, bin/client
make build-server        # только сервер
make build-client        # только клиент

make build-client-all    # кросс-сборка клиента под все платформы:
                          #   bin/client-windows-amd64.exe
                          #   bin/client-linux-amd64
                          #   bin/client-darwin-amd64
                          #   bin/client-darwin-arm64
```

Кросс-сборка работает «из коробки» с одной машины (`CGO_ENABLED=0`), потому что SQLite-драйвер чистый Go — C-тулчейн не нужен.

Адрес сервера вшивается в клиент на этапе сборки. По умолчанию `localhost:8080`, поменять:

```bash
GRPC_ADDR=example.com:8080 make build-client
```

Версия и дата сборки:

```bash
./bin/client --version
# GophKeeper client
# version: v1.0.0
# build date: 2026-06-22T19:55:26Z
```

## Настройка сервера

Конфигурация читается в порядке приоритета: **дефолты** → **JSON-файл** (`-c path/to/config.json`) → **переменные окружения** → **флаги**.

| Флаг | Переменная | JSON-ключ | По умолчанию | Зачем |
|------|------------|-----------|--------------|-------|
| `-t` | `TRANSPORT` | `transport` | `grpc` | Транспорт (реализован gRPC) |
| `-a` | `ADDRESS` | `address` | `:8080` | Адрес прослушивания `host:port` |
| `-d` | `DATABASE_DSN` | `database_dsn` | — | DSN PostgreSQL |
| `-m` | `APP_MODE` | `app_mode` | `production` | `development` — подробные логи |
| `-l` | `LOG_DIR` | `log_dir` | по умолчанию логгера | Каталог логов |
| — | `JWT_SECRET` | `jwt_secret` | — | Секрет подписи access-токена (обязателен) |
| — | `REFRESH_SECRET` | `refresh_secret` | — | Секрет подписи refresh-токена (обязателен) |
| `-tls-cert` | `TLS_CERT` | `tls_cert` | `./certs/server.crt` | Путь к TLS-сертификату |
| `-tls-key` | `TLS_KEY` | `tls_key` | `./certs/server.key` | Путь к приватному ключу |
| `-c` / `-config` | `CONFIG` | — | — | Путь к JSON-конфигу |

Пример `config.json`:

```json
{
  "address": ":8080",
  "database_dsn": "postgres://gophkeeper:gophkeeper@localhost:5432/gophkeeper?sslmode=disable",
  "app_mode": "development",
  "jwt_secret": "change-me",
  "refresh_secret": "change-me"
}
```

DSN под локальный `make db-up`: `postgres://gophkeeper:gophkeeper@localhost:5432/gophkeeper?sslmode=disable`.

## Что где лежит

```
gophkeeper/
├── cmd/                       # Точки входа
│   ├── client/                # CLI/TUI-клиент
│   ├── server/                # gRPC-сервер
│   └── certgen/               # Генератор self-signed TLS-пары
│
├── internal/
│   ├── client/                # Логика клиента
│   │   ├── app/               # Bootstrap: БД, gRPC-соединение, TUI
│   │   ├── view/              # TUI (bubbletea)
│   │   ├── sync/              # Синхронизация типов данных с сервером
│   │   ├── repository/        # Локальное хранилище (SQLite)
│   │   ├── crypto/ vault/     # Клиентское шифрование секретов
│   │   ├── tlsclient/         # TLS-креды клиента (вшитый серт)
│   │   └── ...
│   ├── server/                # Логика сервера
│   │   ├── app/               # Bootstrap сервера
│   │   ├── transport/grpc/    # gRPC-сервер, перехватчики аутентификации
│   │   ├── service/           # Бизнес-логика
│   │   ├── repository/        # Доступ к PostgreSQL
│   │   ├── config/            # Парсинг флагов/env/JSON
│   │   ├── tlsserver/         # Загрузка TLS-кред сервера
│   │   └── ...
│   └── shared/                # Общий код
│       ├── proto/             # Сгенерированный код gRPC (card/file/note/password/user)
│       ├── certgen/           # Генерация сертификатов (+ config)
│       ├── logger/ migrator/ errors/
│
├── api/                       # Исходные .proto-файлы
├── migrations/                # SQL-миграции (server) и схема клиента
├── pkg/luhn/                  # Валидация номеров банковских карт
├── docker/                    # Dockerfile и compose (используется для PostgreSQL)
└── Makefile
```

Если разбираешься в коде — начни с `cmd/*/main.go`, дальше `internal/server/app` (сервер) или `internal/client/app` (клиент).

## Тестирование

```bash
make test              # все юнит-тесты (-race) + сбор покрытия
make coverage          # HTML-отчёт покрытия
make coverage-percent  # суммарный процент
make test-integration  # интеграционные тесты (нужен Docker)
```

Текущее суммарное покрытие — **90.3%** (требование ТЗ — не менее 70%). Каждый экспортируемый символ снабжён godoc-документацией.

## Зависимости

- **Go 1.26+**
- **PostgreSQL** — хранилище сервера (проще всего через `make db-up`)
- **Docker** — для PostgreSQL и интеграционных тестов
- **protoc** — только если перегенерируешь protobuf (`make proto`)
