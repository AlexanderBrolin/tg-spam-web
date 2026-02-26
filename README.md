# tg-spam Corporate Edition

Корпоративная система управления антиспамом для Telegram-каналов. Основана на [tg-spam](https://github.com/umputun/tg-spam) — мощном антиспам-движке с открытым исходным кодом.

## Возможности

- **Веб-панель управления** — React SPA для администрирования через браузер
- **JWT-аутентификация** — три роли: superadmin, admin, moderator
- **Управление ботами** — добавление и валидация Telegram-ботов через веб-интерфейс
- **Поддержка нескольких каналов** — управление множеством Telegram-групп из одной панели
- **Per-channel настройки** — каждый канал имеет собственную конфигурацию антиспама, OpenAI, мета-проверок
- **Динамическое управление** — каналы запускаются/останавливаются без перезапуска сервера
- **Многоуровневая детекция спама** — классификатор Байеса, стоп-слова, CAS, OpenAI, Lua-плагины, мета-проверки
- **SQLite хранилище** — вся информация в одном файле, удобный бэкап и миграция
- **Корпоративный режим** — работа без привязки к CLI, всё управляется через веб-панель
- **Полная обратная совместимость** с CLI-режимом оригинального tg-spam

## Быстрый старт

### Корпоративный режим (рекомендуется)

В корпоративном режиме боты и каналы управляются целиком через веб-панель. Telegram-токен в CLI не нужен.

1. Создайте файл `.env`:

```env
JWT_SECRET=ваш_секретный_ключ_минимум_32_символа
ADMIN_USER=admin
ADMIN_PASSWORD=надежный_пароль
SERVER_ENABLED=true
```

2. Запустите:

```bash
docker compose up -d
```

3. Откройте веб-панель: `http://localhost:8080/app/`

4. Войдите как `admin` с указанным паролем.

5. Через веб-панель:
   - Добавьте бота (раздел **Боты** — введите имя и токен)
   - Создайте канал (раздел **Каналы** — укажите GID, имя, выберите бота)
   - Настройте антиспам (раздел **Настройки канала** — пороги, OpenAI, мета-проверки)

### Legacy-режим (совместимость с оригинальным tg-spam)

```env
TELEGRAM_TOKEN=ваш_токен_бота
TELEGRAM_GROUP=имя_или_id_группы
ADMIN_GROUP=-123456789
JWT_SECRET=ваш_секретный_ключ
ADMIN_USER=admin
ADMIN_PASSWORD=надежный_пароль
SUPER_USER=ваш_username
SERVER_ENABLED=true
```

В legacy-режиме бот подключается через CLI-параметры, а веб-панель работает параллельно для мониторинга и управления сэмплами.

### Сборка из исходников

```bash
# установка зависимостей фронтенда и сборка
cd frontend && npm ci && npm run build && cd ..

# сборка Go-бинарника
cd app && go build -o ../tg-spam && cd ..

# запуск в корпоративном режиме
./tg-spam --server.enabled --auth.jwt-secret=SECRET

# запуск в legacy-режиме
./tg-spam --telegram.token=TOKEN --telegram.group=GROUP --server.enabled --auth.jwt-secret=SECRET
```

Или через Make:

```bash
make build   # собирает фронтенд + Go-бинарник
make test    # запуск тестов с покрытием
make docker  # сборка Docker-образа
```

## Веб-панель управления

Веб-панель доступна по адресу `http://host:port/app/` и предоставляет полный контроль над системой антиспама.

### Аутентификация и авторизация

Панель использует JWT-аутентификацию с парой access/refresh токенов. При первом запуске с `JWT_SECRET` автоматически создаётся суперадмин с указанными `ADMIN_USER`/`ADMIN_PASSWORD` (если пароль не задан — генерируется случайный и выводится в лог).

| Параметр | Переменная окружения | Описание | По умолчанию |
|----------|---------------------|----------|-------------|
| `--auth.jwt-secret` | `AUTH_JWT_SECRET` | Секрет для подписи JWT-токенов | — |
| `--auth.admin-user` | `AUTH_ADMIN_USER` | Логин начального суперадмина | `admin` |
| `--auth.admin-password` | `AUTH_ADMIN_PASSWORD` | Пароль начального суперадмина | авто-генерация |

### Роли и права доступа

Система поддерживает три роли с иерархическим доступом:

| Роль | Описание | Доступные разделы |
|------|----------|-------------------|
| **superadmin** | Полный доступ ко всем функциям | Все разделы, управление пользователями, ботами, каналами, бэкапы |
| **admin** | Управление антиспамом | Дашборд, спам, сэмплы, словарь, проверка сообщений, настройки каналов |
| **moderator** | Мониторинг | Дашборд, обнаруженный спам, одобренные пользователи |

Детальная таблица доступа по разделам:

| Раздел | superadmin | admin | moderator |
|--------|:----------:|:-----:|:---------:|
| Дашборд | + | + | + |
| Обнаруженный спам | + | + | + |
| Одобренные пользователи | + | + | + |
| Проверка сообщений | + | + | — |
| Сэмплы | + | + | — |
| Словарь | + | + | — |
| Настройки каналов | + | + | — |
| Системные настройки | + | + | — |
| Боты | + | — | — |
| Каналы | + | — | — |
| Пользователи админки | + | — | — |
| Бэкап/Экспорт | + | — | — |
| Смена пароля | + | + | + |

### Разделы панели

#### Дашборд

Главная страница с обзорной статистикой: количество обнаруженного спама, одобренных пользователей, активных каналов. Включает визуализацию: круговая диаграмма по типам спама и временной график обнаружений. Данные фильтруются по выбранному каналу.

#### Управление ботами

Раздел для регистрации Telegram-ботов, используемых для мониторинга каналов. Каждый бот — это отдельный токен, полученный через @BotFather.

Возможности:
- Добавление бота по имени и токену
- Валидация токена через Telegram API (проверка getMe)
- Автоматическое определение username бота
- Активация/деактивация ботов
- Удаление ботов
- Маскировка токенов в интерфейсе (отображается только ID бота)

Один бот может обслуживать несколько каналов.

#### Управление каналами

Раздел для добавления Telegram-групп/каналов, которые бот будет защищать от спама.

При создании канала указываются:
- **GID** — уникальный текстовый идентификатор канала (произвольный, используется внутри системы для привязки настроек, одобренных пользователей и т.д.). Рекомендуется использовать осмысленное имя, например `my-chat`, `dev-group`, `main-channel`. GID нельзя изменить после создания.
- **Telegram ID** — числовой ID чата в Telegram (отрицательное число для групп, например `-1001234567890`). Как узнать:
  - Добавьте бота @RawDataBot в группу — он отправит JSON с `chat.id`
  - Или используйте @userinfobot — перешлите ему сообщение из группы
  - Или в web-версии Telegram: откройте группу, в URL будет число после `#-` (добавьте `-100` перед ним)
  - Для супергрупп ID всегда начинается с `-100`
- **Имя** — отображаемое имя канала в панели управления
- **Username** — @username канала (опционально, если есть публичная ссылка)
- **Бот** — выбор из зарегистрированных ботов. Бот должен быть добавлен в группу как администратор с правами на удаление сообщений и бан пользователей

При создании канала автоматически создаётся запись с настройками по умолчанию и запускается listener (если бот назначен). При удалении канала listener останавливается, настройки удаляются.

#### Настройки канала

Каждый канал имеет собственную конфигурацию антиспама. Настройки сгруппированы по разделам:

**Канал:**
- Admin Group — ID Telegram-чата для уведомлений администраторов

**Классификатор:**
- Similarity Threshold — порог сходства с образцами спама (0–1)
- Min Message Length — минимальная длина сообщения для проверки
- Max Emoji — максимальное количество эмодзи (-1 для отключения)
- Min Spam Probability — минимальная вероятность спама для бана (%)
- First Messages Count — количество первых сообщений для проверки
- Paranoid Mode — проверять все сообщения, не только первые

**Интеграции:**
- CAS Enabled — проверка по базе Combot Anti-Spam
- OpenAI Enabled — использование GPT для анализа
- OpenAI Token — API-ключ (per-channel, позволяет использовать разные ключи для разных каналов)
- OpenAI API Base — кастомный эндпоинт API (для прокси или альтернативных провайдеров)
- OpenAI Model — модель GPT (gpt-4o-mini, gpt-4o и т.д.)
- OpenAI Veto Mode — GPT подтверждает или отменяет решение основного детектора

**Мета-проверки:**
- Links Limit — максимум ссылок в сообщении (-1 для отключения)
- Links Only — спам при наличии только ссылок
- Image Only — спам при наличии только изображений
- Video Only — спам при наличии только видео
- Audio Only — спам при наличии только аудио
- Contact Only — спам при наличии контактов
- Forward Detection — спам при пересылке
- Keyboard Detection — спам при наличии inline-клавиатуры
- Username Symbols — проверка запрещённых символов в имени
- Giveaway Detection — детекция розыгрышей

**Детекция дубликатов:**
- Threshold — порог срабатывания (0 для отключения)
- Window — временное окно (например, `1h`, `30m`)

**Поведение:**
- **Training Mode** — пассивный режим обучения. Бот анализирует все сообщения и логирует результаты проверок, но не выполняет никаких действий: не банит, не удаляет сообщения, не отправляет уведомления в чат. Используйте для наблюдения за тем, как бот классифицирует сообщения, прежде чем включать активную защиту. Обнаруженный спам будет виден в разделе «Detected Spam» веб-панели.
- **Dry Mode** — режим сухого запуска. Бот обнаруживает спам и отправляет уведомление в чат (с пометкой «dry mode»), но не банит пользователя и не удаляет сообщение. Полезно для тестирования настроек — вы видите, что бот считает спамом, но пользователи не страдают от ложных срабатываний.
- **Soft Ban** — мягкий бан. Вместо полного бана (kick + ban) бот ограничивает пользователя: запрещает отправку сообщений, медиа, стикеров и т.д. Пользователь остаётся в группе, но не может писать. Администратор может вручную снять ограничения.
- **No Spam Reply** — не отправлять сообщение в чат при обнаружении спама. По умолчанию бот пишет в чат кто был забанен и за что. При включении этой опции бот молча удаляет сообщение и банит пользователя без публичного уведомления. Уведомление в админ-группу отправляется в любом случае (если настроена Admin Group).
- **Aggressive Cleanup** — агрессивная очистка сообщений. При бане пользователя бот удаляет не только спам-сообщение, но и все предыдущие сообщения этого пользователя (до лимита). Работает при бане через автодетекцию и через команду `/spam` в админ-группе.
- **Cleanup Limit** — максимальное количество сообщений для удаления при агрессивной очистке (по умолчанию 100). Telegram API имеет ограничение — можно удалить только сообщения не старше 48 часов.
- **Suppress Join Message** — удалять системное сообщение «Пользователь присоединился к группе», но только когда пользователь забанен за спам. Если пользователь не является спамером, сообщение о входе остаётся.
- **Delete Join Messages** — удалять ВСЕ системные сообщения о входе пользователей в группу, независимо от того, является ли пользователь спамером. Полезно для чистоты чата в больших группах с частым входом/выходом участников.
- **Delete Leave Messages** — удалять ВСЕ системные сообщения о выходе пользователей из группы. Аналогично Delete Join Messages, но для сообщений о выходе.

При сохранении настроек listener канала автоматически перезапускается с новой конфигурацией.

#### Обнаруженный спам

Список сообщений, распознанных как спам. Для каждого сообщения отображается:
- Текст сообщения
- Информация об отправителе (ID, имя)
- Время обнаружения
- Результаты каждой проверки (название чекера, статус, детали)

Для каждой записи доступны три независимых действия:
- **Add to samples** — добавить текст сообщения в спам-сэмплы для дообучения классификатора Байеса. Чем больше реальных примеров — тем точнее детекция.
- **Unban** — разбанить пользователя в Telegram. Снимает бан, но не добавляет пользователя в белый список — при следующем спам-сообщении пользователь будет забанен снова.
- **Add to approved** — добавить пользователя в список одобренных (белый список) для текущего канала. Сообщения одобренных пользователей не проверяются на спам.

Все три действия независимы — можно выполнить любую комбинацию. Например, разбанить пользователя и добавить в одобренные (если бан был ложным срабатыванием), или добавить текст в сэмплы но не разбанивать (если бан корректный и нужно обучить классификатор).

#### Проверка сообщений

Инструмент для ручной проверки текста на спам. Ввод текста в поле и немедленный анализ всеми активными чекерами. Показывает результат каждого чекера отдельно.

#### Одобренные пользователи

Белый список пользователей, сообщения которых не проверяются на спам. Список привязан к конкретному каналу — при выборе канала в сайдбаре отображаются только пользователи этого канала.

Для добавления пользователя:
1. Выберите канал в сайдбаре
2. Нажмите «Add User»
3. Введите **User ID** — числовой Telegram ID пользователя (можно узнать через @userinfobot или @RawDataBot)
4. Введите **Username** (опционально) — @username пользователя для удобства идентификации
5. Нажмите «Add»

Один и тот же пользователь может быть одобрен в одном канале и не одобрен в другом.

#### Сэмплы

Управление обучающими данными классификатора Байеса. Сэмплы общие для всех каналов — изменения влияют на детекцию спама во всех группах.

Два типа сэмплов:

- **Спам-сэмплы (Spam Samples)** — примеры спам-сообщений. Классификатор учится распознавать похожие тексты как спам. Чем больше разнообразных примеров — тем точнее детекция. Рекомендуется добавлять реальные спам-сообщения из раздела «Обнаруженный спам».

- **Хам-сэмплы (Ham Samples)** — примеры легитимных сообщений. Нужны чтобы классификатор отличал спам от нормального общения. Добавляйте примеры типичных сообщений вашего сообщества — вопросы, обсуждения, приветствия.

Классификатор Байеса работает только при наличии обоих типов сэмплов. После добавления/удаления сэмплов нажмите «Reload» для переобучения классификатора (или это произойдёт автоматически).

Также из раздела «Обнаруженный спам» можно добавить сообщение прямо в спам-сэмплы одной кнопкой — это удобный способ дообучать классификатор на реальных примерах.

#### Словарь

Управление двумя списками, общими для всех каналов:

- **Стоп-фразы (Stop Phrases)** — слова и выражения, при наличии которых в сообщении оно считается спамом. Поддерживается два режима:
  - **Подстрочный поиск** (по умолчанию) — фраза ищется как подстрока. Например, стоп-фраза `казино` сработает на сообщение «лучшее казино онлайн».
  - **Точное совпадение** (префикс `=`) — фраза должна совпадать целиком как отдельное слово. Например, `=спам` сработает на «это спам», но не на «антиспам».

  Примеры стоп-фраз: `казино`, `бесплатные крипто`, `=розыгрыш`, `заработок без вложений`.

- **Игнорируемые слова (Ignored Words)** — слова-исключения, которые не учитываются при анализе Байесовским классификатором. Используйте для частых слов, которые создают ложные срабатывания. Например, если слово «подписка» часто встречается и в спаме и в легитимных сообщениях, добавьте его в игнорируемые — классификатор перестанет учитывать его при подсчёте вероятности.

Словарь общий для всех каналов. После добавления/удаления записей сэмплы автоматически перезагружаются.

#### Пользователи админки

Управление учётными записями веб-панели (только superadmin):
- Создание пользователей с выбором роли (superadmin, admin, moderator)
- Активация/деактивация учётных записей
- Сброс паролей
- Удаление пользователей

#### Системные настройки

Отображение текущей конфигурации системы: версия, тип БД, параметры запуска.

#### Бэкап и экспорт

Доступно через API (superadmin):
- Скачивание бэкапа БД (gzip-сжатый SQL)
- Экспорт SQLite в формат PostgreSQL
- Скачивание спам/хам сэмплов
- Скачивание обнаруженного спама (JSONL)

### Глобальный выбор канала

В шапке панели расположен селектор канала. Выбранный канал влияет на данные во всех разделах: обнаруженный спам, одобренные пользователи, статистика и т.д.

## Настройка

### Режимы работы

**Корпоративный режим** — `--server.enabled` + `--auth.jwt-secret` без Telegram-токена. Боты и каналы управляются через веб-панель. Все per-channel настройки хранятся в БД.

**Legacy-режим** — `--telegram.token` + `--telegram.group` + опционально `--server.enabled`. Один бот, один канал, настройки через CLI-параметры. Веб-панель для мониторинга.

**Гибридный режим** — legacy-параметры + `--auth.jwt-secret`. Основной канал работает через CLI, дополнительные каналы добавляются через веб-панель.

### Параметры веб-сервера

| Параметр | Переменная окружения | Описание | По умолчанию |
|----------|---------------------|----------|-------------|
| `--server.enabled` | `SERVER_ENABLED` | Включить веб-сервер | `false` |
| `--server.listen` | `SERVER_LISTEN` | Адрес прослушивания | `:8080` |

### Модули детекции спама

**Анализ сообщений (классификатор Байеса)** — основной модуль. Использует набор спам/хам-сэмплов для классификации. Активен при наличии обоих типов сэмплов.

**Проверка сходства** — сравнивает сообщение с образцами спама. Порог: `--similarity-threshold` (по умолчанию 0.5).

**Стоп-слова** — проверка на наличие запрещённых фраз. Поддерживается точное совпадение (префикс `=`) и подстрочный поиск.

**CAS (Combot Anti-Spam System)** — проверка по внешней антиспам-базе. Включена по умолчанию.

**OpenAI** — интеграция с GPT для анализа сообщений. В корпоративном режиме настраивается per-channel через веб-панель (собственный токен и модель для каждого канала). В legacy-режиме включается через `--openai.token`.

**Мета-проверки** — проверки на количество ссылок, эмодзи, наличие изображений/видео/аудио/контактов, пересылок, клавиатуры, символы в имени пользователя, розыгрыши.

**Lua-плагины** — расширение детекции спама через пользовательские Lua-скрипты. Включается через `--lua-plugins.enabled`.

**Детекция дубликатов** — отслеживание повторяющихся сообщений. Настраивается через `--duplicates.threshold` и `--duplicates.window`.

**Аномальные пробелы** — проверка на нестандартное форматирование текста. Включается через `--space.enabled`.

### Общие и per-channel ресурсы

| Ресурс | Область |
|--------|---------|
| Сэмплы (спам/хам) | Общие для всех каналов |
| Словарь (стоп-фразы, игнорируемые слова) | Общий для всех каналов |
| Одобренные пользователи | Per-channel |
| Детектор (пороги, OpenAI, мета-проверки) | Per-channel |
| Telegram-бот (API-клиент) | Per-channel (через назначение бота) |
| Admin-группа | Per-channel |

### Административный чат

В корпоративном режиме admin-группа настраивается per-channel через веб-панель (поле Admin Group в настройках канала). В legacy-режиме — через `--admin.group`.

Бот отправляет уведомления о спаме в admin-группу, позволяя подтверждать или отменять бан через inline-кнопки.

Команды администратора в Telegram:
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
- `POST /api/v2/spam/unban` — разбанить пользователя в Telegram (gid + user_id)
- `POST /api/v2/spam/check` — проверить сообщение на спам

**Пользователи:**
- `GET /api/v2/users/approved?gid=xxx` — одобренные пользователи канала
- `POST /api/v2/users/approved` — добавить одобренного пользователя (gid в теле запроса)
- `DELETE /api/v2/users/approved/{user_id}?gid=xxx` — удалить из одобренных

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

**Боты (superadmin):**
- `GET /api/v2/bots/` — список ботов (токены маскируются)
- `POST /api/v2/bots/` — зарегистрировать бота
- `PUT /api/v2/bots/{id}` — обновить бота
- `DELETE /api/v2/bots/{id}` — удалить бота
- `POST /api/v2/bots/{id}/validate` — проверить токен через Telegram API

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
- `GET /api/v2/download/backup` — бэкап базы данных (gzip)
- `GET /api/v2/download/export-to-postgres` — экспорт SQLite в PostgreSQL

## Разработка

### Структура проекта

```
app/                    # Go backend
├── main.go             # точка входа, CLI-параметры, корпоративный режим
├── auth/               # JWT-аутентификация, middleware, пароли, адаптеры
├── bot/                # спам-фильтр бот
├── events/             # Telegram listener, channel manager, channel builder
├── storage/            # хранилище (admin_users, bots, channels, settings, refresh_tokens)
│   └── engine/         # database engine (SQLite)
└── webapi/             # HTTP-хэндлеры (v2 API, SPA serving)
    └── frontend/dist/  # собранный React SPA (go:embed)

frontend/               # React SPA исходники
├── src/
│   ├── api/            # API-клиент (axios): auth, bots, channels, spam, users
│   ├── components/     # React-компоненты (Layout, Sidebar, Header, ProtectedRoute)
│   ├── pages/          # страницы (Login, Dashboard, Bots, Channels, Settings и т.д.)
│   └── store/          # состояние (zustand): auth, channels
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

auth:
      --auth.jwt-secret=                JWT signing secret for admin panel auth [$AUTH_JWT_SECRET]
      --auth.admin-user=                initial admin username (default: admin) [$AUTH_ADMIN_USER]
      --auth.admin-password=            initial admin password, auto-generated if empty [$AUTH_ADMIN_PASSWORD]

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
