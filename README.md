# tg-spam Corporate Edition

Корпоративная система управления антиспамом для Telegram-каналов. Основана на [tg-spam](https://github.com/umputun/tg-spam) — мощном антиспам-движке с открытым исходным кодом.

## Возможности

- **Веб-панель управления** — React SPA для администрирования через браузер
- **JWT-аутентификация** — три роли: superadmin, admin, moderator
- **Поддержка нескольких каналов** — управление множеством Telegram-групп из одной панели
- **Управление настройками антиспама** — конфигурация модулей детекции через веб-интерфейс
- **Многоуровневая детекция спама** — классификатор Байеса, стоп-слова, CAS, OpenAI, Lua-плагины, мета-проверки
- **SQLite хранилище** — вся информация в одном файле, удобный бэкап и миграция
- **Полная обратная совместимость** с CLI-режимом оригинального tg-spam

## Быстрый старт

### Docker Compose

1. Создайте файл `.env` в директории проекта:

```env
TELEGRAM_TOKEN=ваш_токен_бота
TELEGRAM_GROUP=имя_или_id_группы
ADMIN_GROUP=-123456789
JWT_SECRET=ваш_секретный_ключ
ADMIN_USER=admin
ADMIN_PASSWORD=надежный_пароль
SUPER_USER=ваш_username
```

2. Запустите:

```bash
docker compose up -d
```

3. Откройте веб-панель: `http://localhost:8080/app/`

### Сборка из исходников

```bash
# установка зависимостей фронтенда и сборка
cd frontend && npm ci && npm run build && cd ..

# сборка Go-бинарника
cd app && go build -o ../tg-spam && cd ..

# запуск
./tg-spam --telegram.token=TOKEN --telegram.group=GROUP --server.enabled
```

Или через Make:

```bash
make build   # собирает фронтенд + Go-бинарник
make test    # запуск тестов с покрытием
make docker  # сборка Docker-образа
```

## Настройка

### Обязательные параметры

| Параметр | Переменная окружения | Описание |
|----------|---------------------|----------|
| `--telegram.token` | `TELEGRAM_TOKEN` | Токен Telegram-бота |
| `--telegram.group` | `TELEGRAM_GROUP` | Имя или ID группы |

Бот может работать в режиме только веб-сервера (без Telegram) при `--server.enabled` без указания токена и группы.

### Параметры веб-панели

| Параметр | Переменная окружения | Описание |
|----------|---------------------|----------|
| `--server.enabled` | `SERVER_ENABLED` | Включить веб-сервер |
| `--server.listen` | `SERVER_LISTEN` | Адрес для прослушивания (по умолчанию `:8080`) |

Для v2 API (React SPA) требуется настройка JWT-аутентификации. Параметры `JWT_SECRET`, `ADMIN_USER` и `ADMIN_PASSWORD` задаются через переменные окружения.

### Модули детекции спама

**Анализ сообщений (классификатор Байеса)** — основной модуль. Использует набор спам/хам-сэмплов для классификации. Активен при наличии обоих типов сэмплов.

**Проверка сходства** — сравнивает сообщение с образцами спама. Порог: `--similarity-threshold` (по умолчанию 0.5).

**Стоп-слова** — проверка на наличие запрещённых фраз. Поддерживается точное совпадение (префикс `=`) и подстрочный поиск.

**CAS (Combot Anti-Spam System)** — проверка по внешней антиспам-базе. Включена по умолчанию.

**OpenAI** — интеграция с GPT для анализа сообщений. Включается через `--openai.token`. Поддерживает veto-режим, контекст истории, custom prompts.

**Мета-проверки** — проверки на количество ссылок, эмодзи, наличие изображений/видео/аудио/контактов, пересылок, клавиатуры, символы в имени пользователя, розыгрыши.

**Lua-плагины** — расширение детекции спама через пользовательские Lua-скрипты. Включается через `--lua-plugins.enabled`.

**Детекция дубликатов** — отслеживание повторяющихся сообщений. Настраивается через `--duplicates.threshold` и `--duplicates.window`.

**Аномальные пробелы** — проверка на нестандартное форматирование текста. Включается через `--space.enabled`.

### Роли и доступ

| Роль | Возможности |
|------|------------|
| **superadmin** | Полный доступ: управление пользователями, каналами, настройками, бэкапы |
| **admin** | Управление сэмплами, словарём, настройками каналов |
| **moderator** | Просмотр обнаруженного спама, одобренных пользователей, проверка сообщений |

### Административный чат

Опционально можно указать админский чат/группу через `--admin.group`. Бот отправляет уведомления о спаме в этот чат, позволяя подтверждать или отменять бан через inline-кнопки.

Команды администратора:
- `/spam` — пометить сообщение как спам (обучение бота)
- `/ban` — забанить пользователя без добавления в сэмплы
- `/warn` — предупредить пользователя

### Пользовательские репорты

При включении `--report.enabled` обычные пользователи могут сообщать о спаме командой `/report`. При достижении порога (`--report.threshold`) уведомление отправляется админам.

## API

### v2 API (JWT)

Все эндпоинты (кроме `/api/v2/auth/login` и `/api/v2/auth/refresh`) требуют JWT-токен в заголовке `Authorization: Bearer <token>`.

**Аутентификация:**
- `POST /api/v2/auth/login` — получить access/refresh токены
- `POST /api/v2/auth/refresh` — обновить access токен
- `POST /api/v2/auth/logout` — инвалидировать refresh токен
- `GET /api/v2/auth/me` — информация о текущем пользователе
- `PUT /api/v2/auth/password` — смена пароля

**Спам:**
- `GET /api/v2/spam/detected` — список обнаруженного спама (пагинация, фильтры)
- `POST /api/v2/spam/detected/{id}/add` — добавить спам в сэмплы
- `POST /api/v2/spam/check` — проверить сообщение на спам

**Пользователи:**
- `GET /api/v2/users/approved` — одобренные пользователи
- `POST /api/v2/users/approved` — добавить одобренного пользователя
- `DELETE /api/v2/users/approved/{user_id}` — удалить из одобренных

**Сэмплы (admin+):**
- `GET /api/v2/samples/` — получить спам/хам сэмплы
- `POST /api/v2/samples/spam` — добавить спам-сэмпл
- `POST /api/v2/samples/ham` — добавить хам-сэмпл
- `DELETE /api/v2/samples/spam` — удалить спам-сэмпл
- `DELETE /api/v2/samples/ham` — удалить хам-сэмпл
- `PUT /api/v2/samples/reload` — перезагрузить сэмплы

**Словарь (admin+):**
- `GET /api/v2/dictionary/` — стоп-фразы и игнорируемые слова
- `POST /api/v2/dictionary/` — добавить запись
- `DELETE /api/v2/dictionary/` — удалить запись

**Каналы:**
- `GET /api/v2/channels/` — список каналов
- `POST /api/v2/channels/` — создать канал (superadmin)
- `PUT /api/v2/channels/{id}` — обновить канал (superadmin)
- `DELETE /api/v2/channels/{id}` — удалить канал (superadmin)
- `GET /api/v2/channels/{gid}/settings` — настройки канала
- `PUT /api/v2/channels/{gid}/settings` — обновить настройки (admin+)

**Администрирование (superadmin):**
- `GET /api/v2/admin/users/` — список админов
- `POST /api/v2/admin/users/` — создать админа
- `PUT /api/v2/admin/users/{id}` — обновить админа
- `DELETE /api/v2/admin/users/{id}` — удалить админа
- `PUT /api/v2/admin/users/{id}/password` — сбросить пароль

**Прочее:**
- `GET /api/v2/stats` — статистика
- `GET /api/v2/settings` — текущие настройки
- `GET /api/v2/download/spam` — скачать спам-сэмплы
- `GET /api/v2/download/ham` — скачать хам-сэмплы
- `GET /api/v2/download/detected_spam` — скачать обнаруженный спам (JSONL)
- `GET /api/v2/download/backup` — бэкап базы данных
- `GET /api/v2/download/export-to-postgres` — экспорт SQLite в PostgreSQL

## Разработка

### Структура проекта

```
app/                    # Go backend
├── main.go             # точка входа, CLI-параметры
├── auth/               # JWT-аутентификация, middleware, пароли
├── bot/                # спам-фильтр бот
├── events/             # Telegram listener, channel manager
├── storage/            # хранилище (admin_users, channels, settings, refresh_tokens)
│   └── engine/         # database engine (SQLite)
└── webapi/             # HTTP-хэндлеры (v2 API, SPA serving)
    └── frontend/dist/  # собранный React SPA (go:embed)

frontend/               # React SPA исходники
├── src/
│   ├── api/            # API-клиент (axios)
│   ├── components/     # React-компоненты
│   ├── pages/          # страницы (Login, Dashboard, Spam, Users и т.д.)
│   └── store/          # состояние (zustand)
└── vite.config.ts      # конфигурация Vite

lib/                    # библиотеки спам-детекции
├── approved/           # одобренные пользователи
├── spamcheck/          # типы для проверки спама
└── tgspam/             # детектор спама, чекеры

data/                   # пресет сэмплов для Docker-образа
```

### Сборка и тестирование

```bash
go test -race ./...           # запуск тестов
golangci-lint run             # линтер
go build -o tg-spam ./app     # сборка бинарника
```

### Фронтенд

```bash
cd frontend
npm ci                        # установка зависимостей
npm run dev                   # dev-сервер (порт 3000, проксирует API на 8080)
npm run build                 # production-сборка в app/webapi/frontend/dist/
```

## Все параметры приложения

```
      --instance-id=                    instance id (default: tg-spam) [$INSTANCE_ID]
      --db=                             database URL, if empty uses sqlite (default: tg-spam.db) [$DB]
      --admin.group=                    admin group name, or channel id [$ADMIN_GROUP]
      --disable-admin-spam-forward      disable handling messages forwarded to admin group as spam [$DISABLE_ADMIN_SPAM_FORWARD]
      --testing-id=                     testing ids, allow bot to reply to them [$TESTING_ID]
      --history-duration=               history duration (default: 24h) [$HISTORY_DURATION]
      --history-min-size=               history minimal size to keep (default: 1000) [$HISTORY_MIN_SIZE]
      --storage-timeout=                storage timeout (default: 0s) [$STORAGE_TIMEOUT]
      --super=                          super-users [$SUPER_USER]
      --no-spam-reply                   do not reply to spam messages [$NO_SPAM_REPLY]
      --suppress-join-message           delete join message if user is kicked out [$SUPPRESS_JOIN_MESSAGE]
      --similarity-threshold=           spam threshold (default: 0.5) [$SIMILARITY_THRESHOLD]
      --min-msg-len=                    min message length to check (default: 50) [$MIN_MSG_LEN]
      --max-emoji=                      max emoji count in message, -1 to disable check (default: 2) [$MAX_EMOJI]
      --min-probability=                min spam probability percent to ban (default: 50) [$MIN_PROBABILITY]
      --multi-lang=                     number of words in different languages to consider as spam (default: 0) [$MULTI_LANG]
      --paranoid                        paranoid mode, check all messages [$PARANOID]
      --first-messages-count=           number of first messages to check (default: 1) [$FIRST_MESSAGES_COUNT]
      --aggressive-cleanup              delete all messages from user when banned via /spam command [$AGGRESSIVE_CLEANUP]
      --aggressive-cleanup-limit=       max messages to delete in aggressive cleanup mode (default: 100) [$AGGRESSIVE_CLEANUP_LIMIT]
      --training                        training mode, passive spam detection only [$TRAINING]
      --soft-ban                        soft ban mode, restrict user actions but not ban [$SOFT_BAN]
      --history-size=                   history size (default: 100) [$LAST_MSGS_HISTORY_SIZE]
      --convert=[only|enabled|disabled] convert mode for txt samples and other storage files to DB (default: enabled)
      --max-backups=                    maximum number of backups to keep, set 0 to disable (default: 10) [$MAX_BACKUPS]
      --dry                             dry mode, no bans [$DRY]
      --dbg                             debug mode [$DEBUG]
      --tg-dbg                          telegram debug mode [$TG_DEBUG]

delete:
      --delete.join-messages            delete join messages immediately [$DELETE_JOIN_MESSAGES]
      --delete.leave-messages           delete leave messages immediately [$DELETE_LEAVE_MESSAGES]

telegram:
      --telegram.token=                 telegram bot token [$TELEGRAM_TOKEN]
      --telegram.group=                 group name/id [$TELEGRAM_GROUP]
      --telegram.timeout=               http client timeout for telegram (default: 30s) [$TELEGRAM_TIMEOUT]
      --telegram.idle=                  idle duration (default: 30s) [$TELEGRAM_IDLE]

logger:
      --logger.enabled                  enable spam rotated logs [$LOGGER_ENABLED]
      --logger.file=                    location of spam log (default: tg-spam.log) [$LOGGER_FILE]
      --logger.max-size=                maximum size before it gets rotated (default: 100M) [$LOGGER_MAX_SIZE]
      --logger.max-backups=             maximum number of old log files to retain (default: 10) [$LOGGER_MAX_BACKUPS]

cas:
      --cas.api=                        CAS API (default: https://api.cas.chat) [$CAS_API]
      --cas.timeout=                    CAS timeout (default: 5s) [$CAS_TIMEOUT]
      --cas.user-agent=                 User-Agent header for CAS API requests [$CAS_USER_AGENT]

meta:
      --meta.links-limit=              max links in message, disabled by default (default: -1) [$META_LINKS_LIMIT]
      --meta.mentions-limit=           max mentions in message, disabled by default (default: -1) [$META_MENTIONS_LIMIT]
      --meta.image-only                enable image only check [$META_IMAGE_ONLY]
      --meta.links-only                enable links only check [$META_LINKS_ONLY]
      --meta.video-only                enable video only check [$META_VIDEO_ONLY]
      --meta.audio-only                enable audio only check [$META_AUDIO_ONLY]
      --meta.contact-only              enable contact only check [$META_CONTACT_ONLY]
      --meta.forward                   enable forward check [$META_FORWARD]
      --meta.keyboard                  enable keyboard check [$META_KEYBOARD]
      --meta.username-symbols=         prohibited symbols in username, disabled by default [$META_USERNAME_SYMBOLS]
      --meta.giveaway                  enable giveaway check [$META_GIVEAWAY]

openai:
      --openai.token=                  openai token, disabled if not set [$OPENAI_TOKEN]
      --openai.apibase=                custom openai API base [$OPENAI_API_BASE]
      --openai.veto                    veto mode, confirm detected spam [$OPENAI_VETO]
      --openai.prompt=                 openai system prompt [$OPENAI_PROMPT]
      --openai.custom-prompt=          additional custom prompts [$OPENAI_CUSTOM_PROMPT]
      --openai.model=                  openai model (default: gpt-4o-mini) [$OPENAI_MODEL]
      --openai.max-tokens-response=    max tokens in response (default: 1024) [$OPENAI_MAX_TOKENS_RESPONSE]
      --openai.max-tokens-request=     max tokens in request (default: 2048) [$OPENAI_MAX_TOKENS_REQUEST]
      --openai.max-symbols-request=    max symbols in request (default: 16000) [$OPENAI_MAX_SYMBOLS_REQUEST]
      --openai.retry-count=            retry count (default: 1) [$OPENAI_RETRY_COUNT]
      --openai.history-size=           history size (default: 0) [$OPENAI_HISTORY_SIZE]
      --openai.reasoning-effort=       reasoning effort (default: none) [$OPENAI_REASONING_EFFORT]
      --openai.check-short-messages    check short messages with OpenAI [$OPENAI_CHECK_SHORT_MESSAGES]

lua-plugins:
      --lua-plugins.enabled            enable Lua plugins [$LUA_PLUGINS_ENABLED]
      --lua-plugins.plugins-dir=       directory with Lua plugins [$LUA_PLUGINS_PLUGINS_DIR]
      --lua-plugins.enabled-plugins=   list of enabled plugins [$LUA_PLUGINS_ENABLED_PLUGINS]
      --lua-plugins.dynamic-reload     dynamically reload plugins [$LUA_PLUGINS_DYNAMIC_RELOAD]

space:
      --space.enabled                  enable abnormal spacing check [$SPACE_ENABLED]
      --space.ratio=                   spaces to characters ratio (default: 0.3) [$SPACE_RATIO]
      --space.short-ratio=             short words ratio (default: 0.7) [$SPACE_SHORT_RATIO]
      --space.short-word=              short word length (default: 3) [$SPACE_SHORT_WORD]
      --space.min-words=               min words to check (default: 5) [$SPACE_MIN_WORDS]

duplicates:
      --duplicates.threshold=          duplicate messages threshold (0=disabled) (default: 0) [$DUPLICATES_THRESHOLD]
      --duplicates.window=             time window (default: 1h) [$DUPLICATES_WINDOW]

report:
      --report.enabled                 enable user spam reporting [$REPORT_ENABLED]
      --report.threshold=              reports to trigger notification (default: 2) [$REPORT_THRESHOLD]
      --report.auto-ban-threshold=     auto-ban after N reports (default: 0) [$REPORT_AUTO_BAN_THRESHOLD]
      --report.rate-limit=             max reports per period (default: 10) [$REPORT_RATE_LIMIT]
      --report.rate-period=            rate limit period (default: 1h) [$REPORT_RATE_PERIOD]

files:
      --files.samples=                 samples data path [$FILES_SAMPLES]
      --files.dynamic=                 dynamic data path (default: data) [$FILES_DYNAMIC]

message:
      --message.startup=               startup message [$MESSAGE_STARTUP]
      --message.spam=                  spam message (default: this is spam) [$MESSAGE_SPAM]
      --message.dry=                   dry mode message (default: this is spam (dry mode)) [$MESSAGE_DRY]
      --message.warn=                  warning message [$MESSAGE_WARN]

server:
      --server.enabled                 enable web server [$SERVER_ENABLED]
      --server.listen=                 listen address (default: :8080) [$SERVER_LISTEN]
```

## Лицензия

MIT License. Основан на [tg-spam](https://github.com/umputun/tg-spam) by Umputun.
