# obsidian-auth

Микросервис аутентификации для RTC-приложения. Отвечает за регистрацию, вход, управление сессиями и выдачу JWT-токенов, которые остальные сервисы используют для авторизации.

## Стек

- Go 1.25
- gRPC (контракты — `github.com/Zed3611/obsidian-protos`)
- PostgreSQL (`pgx/v5`) — пользователи и сессии
- Redis (`go-redis/v9`) — blacklist отозванных сессий
- JWT HS256 (`golang-jwt/v5`)
- `golang-migrate/v4` для миграций
- Docker + distroless-образ с `grpc_health_probe`

## gRPC API

Сервис реализует методы из `auth.v1`:

- `Register` — создание пользователя по email/password.
- `Login` — выпуск пары access + refresh-токенов и создание сессии.
- `Logout` — удаление сессии и blacklisting ещё живого access-токена.
- `RefreshToken` — обмен refresh-токена на новую пару токенов.
- `GetSessions` — список активных сессий пользователя.
- `RevokeSession` / `RevokeAllSessions` — отзыв конкретной/всех сессий.

Также регистрируется стандартный gRPC Health Service со статусами для `""`, `liveness` и `readiness`.

### Модель токенов

- **Access-токен** — JWT HS256 с claims `sub` (user id), `email`, `sid` (session id), `iat`, `exp`. Срок жизни — `ACCESS_TOKEN_DURATION_MINUTES` (по умолчанию 5 минут).
- **Refresh-токен** — 32 случайных байта в hex. В БД хранится только его SHA-256-хэш. Срок жизни — `SESSION_DURATION_MINUTES` (по умолчанию 7 дней).
- При logout/revoke запись сессии удаляется из БД, а её `sid` уходит в Redis-blacklist (`blacklist:<sessionID>`) на время жизни access-токена — это закрывает окно, пока ещё валидный JWT не истёк.

## Архитектура

Слоистая, зависимости направлены сверху вниз:

1. **Транспорт** — `pkg/grpc/auth`: gRPC-хендлеры + валидация `validator/v10`.
2. **Бутстрап gRPC** — `pkg/app/grpc`: `grpc.Server` с интерсепторами логирования и recovery.
3. **Сервис** — `pkg/service/auth`: бизнес-логика, JWT, refresh-токены, blacklist.
4. **Хранилище** — `pkg/storage/postgresql`: репозитории пользователей и сессий, sentinel-ошибки в `pkg/storage`.
5. **Кэш** — `pkg/cache/redis`: blacklist + `health.Pinger`.
6. **Health** — `pkg/health`: периодический пинг Postgres и Redis, маппинг в gRPC health-статусы.

Точка сборки зависимостей — `pkg/app/app.go` (`app.New`). `cmd/auth/main.go` отвечает только за чтение env и graceful shutdown.

## Конфигурация

Всё конфигурируется переменными окружения:

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `APP_ENV` | `local` | `local`/`dev`/`prod`, влияет на формат логов slog |
| `GRPC_PORT` | — | Порт gRPC-сервера |
| `PG_CONNECT_STRING` | — | DSN PostgreSQL |
| `JWT_SECRET` | — | Секрет для подписи JWT (HS256) |
| `ACCESS_TOKEN_DURATION_MINUTES` | `5` | Время жизни access-токена |
| `SESSION_DURATION_MINUTES` | `10080` | Время жизни refresh/сессии (7 дней) |
| `REDIS_ADDR` | — | Адрес Redis |
| `REDIS_PASS` | — | Пароль Redis (опционально) |
| `HEALTH_CHECK_INTERVAL_S` | `5` | Интервал health-чекера |
| `HEALTH_CHECK_TIMEOUT_S` | `5` | Таймаут одной проверки |

`cmd/migrator` читает только `PG_CONNECT_STRING`. Индекс Redis-БД захардкожен в `0`.

Шаблон — в [.env.example](./.env.example).

## Запуск

### Локально

```bash
cp .env.example .env
# заполнить JWT_SECRET и при необходимости остальные переменные

make migrate                 # накатить миграции
make migrate ARGS="-seed"    # + засеять тестовых пользователей
go run ./cmd/auth
```

### Docker Compose

Поднимает Postgres, Redis и сам сервис; readiness проверяется через `grpc_health_probe`:

```bash
docker compose up --build
```

Для сборки используется приватный модуль `github.com/Zed3611/*`, поэтому BuildKit пробрасывает SSH-агент (`--mount=type=ssh`). Убедитесь, что `ssh-agent` запущен и ключ добавлен.

## Миграции

- `make migrate` — `Up` до последней версии.
- `make migrate ARGS="-rollback=N"` — откатить N шагов.
- `make migrate ARGS="-seed"` — после `Up` засеять детерминированных тестовых пользователей через `internal/seeder`.

Файлы миграций — `migrations/*.up.sql` / `*.down.sql`, движок — `golang-migrate/v4` (file-source).

## Разработка

```bash
go build ./cmd/auth
go vet ./...
go test ./...
go mod tidy
```

Юнит-тесты пока покрывают только транспортный слой (`pkg/grpc/auth/server_test.go`).