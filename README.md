# t-schedule

Telegram-бот, который переносит расписание студента Т-университета/Школы Икс
(`edu.donstu.ru`) в Google Calendar и держит его синхронизированным в фоне.

Это Go-переписывание оригинального Node.js/TypeScript бота (Telegraf +
Firestore), сделанное с рефактором под [go-telegram/bot](https://github.com/go-telegram/bot)
вместо построчного переноса. Функциональность сохранена 1:1:

- `/start` — карточка с ссылками на календарь.
- `/auth <логин> <пароль>` — авторизация на edu.donstu.ru, создание личного календаря.
- `/student <id>` / inline-поиск по фамилии — привязка конкретного студента.
- `/colorize` — раскраска событий через доступ по email.
- Фоновая синхронизация расписания каждые 5 минут (`internal/schedule`).

Хранилище — Google Cloud Firestore (как в оригинале), календарь — Google
Calendar API. Оба используют один и тот же сервис-аккаунт (`credentials.json`).

## Конфигурация

Скопируйте `.env.example` в `.env` и заполните:

| Переменная | Обязательна | Описание |
| - | - | - |
| `BOT_TOKEN` | да | Токен Telegram-бота от @BotFather |
| `CREDENTIALS_PATH` | нет (по умолчанию `credentials.json`) | Путь к JSON-ключу сервис-аккаунта Google (Firestore + Calendar). В Docker-образе используйте `/credentials.json` (см. ниже) |
| `FIRESTORE_PROJECT_ID` | нет | ID проекта GCP; если не задан, определяется автоматически из `credentials.json` |
| `PORT`, `DOMAIN` | нет | Включают webhook-режим вместо long polling |

Сервис-аккаунту нужны роли Firestore (Cloud Datastore User) и доступ к
Google Calendar API (домен-делегирование или обычный service account —
календари создаются от его имени).

## Запуск локально

```bash
task dev            # go run . --debug
```

Или без Task:

```bash
go run . --debug
```

## Сборка бинаря

```bash
task build           # bin/t-schedule
task lint            # go vet + go test + gofmt
```

## Сборка образа и релиз

Образ собирается без Dockerfile — через [ko](https://ko.build/) и
[goreleaser](https://goreleaser.com/) (`.goreleaser.yaml`), как и в
`stack`. CI (`.github/workflows/`):

- **test.yaml** — на каждый push/PR в `main`: `go vet`, `go test`,
  `golangci-lint`, сборка бинаря.
- **release.yaml** — на тег `v*`: прогоняет `task lint`, собирает
  бинарники под linux/amd64+arm64 через goreleaser, публикует
  multi-arch образ через `ko` в `ghcr.io/lentryd/t-schedule-go` (теги
  `latest` и версия), подписывает артефакты `cosign`, прикладывает SBOM
  (`syft`).

Чтобы выпустить релиз: запушить тег `vX.Y.Z` — остальное сделает CI.
Локально нужны установленные `ko`, `goreleaser`, `upx` (см. `taskfile.yaml`).

Локальная тестовая сборка образа (без публикации):

```bash
task docker           # KO_DOCKER_REPO=ko.local/t-schedule ko build . --bare
```

Снапшот полного релиза без публикации:

```bash
task snapshot          # goreleaser --snapshot --clean --skip=validate
```

## Docker Compose

`compose.yml` берёт уже опубликованный образ `ghcr.io/lentryd/t-schedule-go:latest`,
монтирует `./credentials.json` в контейнер как `/credentials.json:ro` и
берёт переменные окружения из `.env`. В `.env` для Docker-запуска укажите:

```env
CREDENTIALS_PATH=/credentials.json
```

Запуск:

```bash
task compose:up         # docker compose up -d
task compose:down       # docker compose down
```

## Структура проекта

```
main.go                    точка входа: flags, graceful shutdown, запуск бота + планировщика
internal/
  config/                   переменные окружения, валидация
  store/                    Firestore: users/sessions/providers, кэш studentList
  pkg/
    eduapi/                 клиент edu.donstu.ru + форматирование расписания
    gcalendar/               обёртка над Google Calendar API
    colorize/                 подбор ближайшего Google Calendar colorId (CIE94)
  schedule/                 фоновая синхронизация (cron */5 * * * *)
  telegram/                 go-telegram/bot: middleware, handlers, сообщения
```
