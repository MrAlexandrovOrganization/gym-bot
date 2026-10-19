# 🏋️ Gym Bot - Telegram бот для организации походов в спортзал

## Версии сборки

Go берётся из `go.mod`; Alpine и PostgreSQL — из `makefiles/versions.mk`.
`make versions` показывает значения. Для сборки и управления контейнерами
используйте Make (`make build`, `make up`, `make down`): он экспортирует версии
в Compose и Dockerfile. CI также берёт Go из `go.mod`.

Telegram бот для создания опросов о посещении спортзала. Позволяет участникам группового чата голосовать за дни недели, когда они планируют пойти в зал.

## Production: Hostkey Russia

Бот и PostgreSQL размещаются на `hostkey-russia` в
`/home/maxim/projects/telegram-bots/gym-bot`, с обычным
`compose/docker-compose.yml`. PostgreSQL хранит данные в томе
`gym-bot_postgres_data` и публикует только `127.0.0.1:5440`;
Compose project остаётся `gym-bot`. Образ PostgreSQL закреплён на версии
исходного сервера (15.19), без обновления при переносе.

Бот обращается к общему локальному `http://telegram-bot-api:8081` в
`telegram-net` на Russia. API развёрнут проектом `infra/network/telegram-bot-api`:
его MTProto-трафик идёт через SOCKS `entry:10091` и существующий Reality
reverse tunnel в hostkey-finland. Прямого выхода API в интернет нет.
Прежний API на server продолжает обслуживать оставшихся там ботов.
HTTP нужен внешнему контракту Telegram; перехода на cloud API нет.

GitHub Secrets `VM_HOST`, `VM_USER`, `VM_SSH_KEY`, `VM_PROJECT_PATH` этого
репозитория указывают на Russia. Общие secrets организации не меняются.
CI собирает и публикует образ, затем доставляет проверенный SHA и digest
штатной целью `ci-deploy`; на VM приложение не собирается.
`DEPLOY_ENABLED=false` временно отключает доставку, сохраняя проверки,
сборку и публикацию (например, при переносе данных).

При переносе остановите исходный polling, запретите restart старых контейнеров,
снимите финальный `pg_dump -Fc` и сохраните `.env`, manifests и приватную
сессию Bot API вне VM. Восстановите dump в пустую БД через
`pg_restore --exit-on-error` и сравните количество строк, полный хэш
упорядоченного `polls`, sequence, схему и ограничения. Для переноса сессии
без потери очереди вызовите deleteWebhook без drop_pending_updates, затем
close на старом локальном API и перенесите только каталог этого бота в новый
том с сохранением владельца. Не выводите путь сессии: он может содержать токен.
Порядок описан в
[официальном руководстве Telegram](https://github.com/tdlib/telegram-bot-api#moving-a-bot-from-one-local-server-to-another).
Не запускайте два polling-процесса одного токена.

После доставки проверьте SHA, digest, PostgreSQL readiness, авторизацию и
успешные циклы getUpdates, отсутствие конфликтов polling и ошибок/OOM.
Рабочие сообщения в чат при инфраструктурной проверке не отправляются.
Старые контейнеры и том остаются остановленными для отката. После новых записей
и обновлений Telegram откат требует переноса актуальной БД и последней закрытой
сессии обратно; нельзя подменять их старым dump/архивом.

## 📋 Возможности

- ✅ Создание не анонимных опросов с выбором дней недели
- 📊 Один опрос на неделю (защита от дубликатов)
- 🕗 Автоматическое создание опроса каждый понедельник в 08:00 (UTC+3)
- 💾 PostgreSQL база данных для надёжного хранения данных
- 🔍 Быстрый поиск опроса текущей недели
- 📅 Автоматическое определение недели года
- 🔒 Работа только с одним разрешённым чатом (указывается в конфигурации)
- 🐳 Полная поддержка Docker и docker-compose

## 🚀 Быстрый старт с Docker

### Требования

- Docker и Docker Compose
- Telegram аккаунт
- Токен бота от [@BotFather](https://t.me/BotFather)
- ID группового чата, где будет работать бот

### Шаги установки

1. **Клонируйте репозиторий:**
```bash
git clone <repository-url>
cd gym-bot
```

2. **Создайте файл `.env`:**
```bash
cp .env.example .env
```

3. **Получите токен бота:**
   - Откройте Telegram и найдите [@BotFather](https://t.me/BotFather)
   - Отправьте команду `/newbot`
   - Следуйте инструкциям для создания бота
   - Скопируйте полученный токен
   
   **Важно! Настройте бота для работы в группах:**
   - Отправьте [@BotFather](https://t.me/BotFather) команду `/mybots`
   - Выберите вашего бота
   - Нажмите **Bot Settings** → **Group Privacy**
   - Выберите **Disable** (отключить приватность)

4. **Получите ID чата:**
   - Добавьте бота [@userinfobot](https://t.me/userinfobot) в ваш групповой чат
   - Бот отправит информацию о чате, включая его ID
   - ID чата будет отрицательным числом, например: `-1001234567890`

5. **Настройте `.env` файл:**
```env
TELEGRAM_BOT_TOKEN=ваш_токен_от_botfather
ALLOWED_CHAT_ID=-1001234567890
POSTGRES_USER=gymbot
POSTGRES_PASSWORD=измените_на_надёжный_пароль
POSTGRES_DB=gymbot
```

6. **Запустите с помощью Make (рекомендуется):**
```bash
make up
```

Дополнительные операции Compose также запускаются через Make, чтобы получить версии:
```bash
make up
```

7. **Проверьте логи:**
```bash
make logs-bot
```

Или:
```bash
docker compose --project-directory . logs -f bot
```

Вы должны увидеть:
```
bot_1       | Авторизован как @ваш_бот
bot_1       | Бот работает только с чатом ID: -1001234567890
bot_1       | Успешное подключение к PostgreSQL
bot_1       | ✅ Бот запущен и готов к работе!
```

### Управление контейнерами

Проект включает [`Makefile`](Makefile:1) для упрощения работы. Все команды:

```bash
# Показать все доступные команды
make help

# Создать .env файл из шаблона
make env

# Запустить контейнеры
make up

# Остановить контейнеры
make down

# Перезапустить контейнеры
make restart

# Посмотреть логи всех контейнеров
make logs

# Посмотреть только логи бота
make logs-bot

# Посмотреть только логи базы данных
make logs-db

# Показать статус контейнеров
make ps

# Пересобрать образы
make build

# Пересобрать и перезапустить
make rebuild

# Подключиться к PostgreSQL
make db

# Пересоздать базу данных (удалит все данные!)
make db-reset

# Остановить и удалить всё включая данные
make clean

# Запустить тесты и статические проверки
make test

# Проверить весь локальный gate, включая Compose без .env
make check
```

`make test` не требует `.env`: он запускает изолированные Go-тесты. Для
проверки Compose используется `make config-check` с синтетическими значениями;
реальные секреты и подключения к Telegram/PostgreSQL не используются.

Или используя docker-compose напрямую:

```bash
# Запустить
docker compose --project-directory . -f compose/docker-compose.yml up -d

# Остановить
make down

# Перезапустить
make restart

# Посмотреть логи
make logs-bot

# Остановить и удалить данные (включая БД)
make compose ARGS='down -v'
```

## 📱 Использование

### Добавление бота в группу

1. **Добавьте бота в групповой чат:**
   - Откройте групповой чат в Telegram
   - Нажмите на название группы → Добавить участников
   - Найдите вашего бота по username и добавьте его

2. **Убедитесь в правильных настройках:**
   - Бот должен иметь права на отправку сообщений (обычно установлены по умолчанию)
   - Проверьте, что Group Privacy отключена в [@BotFather](https://t.me/BotFather)
   - Убедитесь, что ID чата совпадает с указанным в `.env` (ALLOWED_CHAT_ID)
   
3. **Протестируйте бота:**
   - Отправьте команду `/start` в группе
   - Бот должен ответить приветственным сообщением
   - Если бот не отвечает, проверьте настройки и логи

### Команды

#### `/start`
Показывает приветственное сообщение и список доступных команд.

```
/start
```

#### `/newpoll`
Создаёт новый опрос на текущую неделю. Опрос содержит:
- Все дни недели (Понедельник - Воскресенье)
- Вариант "Не могу" для тех, кто не сможет пойти

**Ограничения:**
- Можно создать только один опрос на неделю
- Опрос не анонимный (видно, кто голосовал)
- Поддерживается множественный выбор

```
/newpoll
```

#### `/findpoll`
Отправляет ответ на сообщение с опросом текущей недели. Это помогает быстро найти опрос в истории чата.

```
/findpoll
```

#### `/help`
Показывает справку по доступным командам.

```
/help
```

## 🗄️ База данных

Используется PostgreSQL 15 для надёжного хранения данных.

### Структура таблицы `polls`

| Поле | Тип | Описание |
|------|-----|----------|
| id | SERIAL | Уникальный идентификатор |
| chat_id | BIGINT | ID чата в Telegram |
| message_id | INTEGER | ID сообщения с опросом |
| poll_id | TEXT | ID опроса Telegram |
| week_number | INTEGER | Номер недели в году |
| year | INTEGER | Год |
| created_at | TIMESTAMP | Время создания |

### Уникальные ограничения

- Только один опрос на комбинацию (chat_id, week_number, year)
- Уникальный poll_id для каждого опроса

### Подключение к PostgreSQL

База данных доступна на `localhost:5440` (при использовании docker-compose).

Для прямого подключения:
```bash
docker compose --project-directory . -f compose/docker-compose.yml exec postgres psql -U gymbot -d gymbot
```

Или через любой PostgreSQL клиент:
```
Host: localhost
Port: 5440
Database: gymbot
User: gymbot
Password: (из .env)
```

## 📦 Зависимости

- [`github.com/mymmrac/telego`](https://github.com/mymmrac/telego) - Telegram Bot API для Go
- [`github.com/lib/pq`](https://github.com/lib/pq) - PostgreSQL драйвер для Go
- [`github.com/joho/godotenv`](https://github.com/joho/godotenv) - Загрузка переменных окружения из .env файла
- [`golang.org/x/sync/errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup) - Параллельный запуск горутин с обработкой ошибок
- [`github.com/robfig/cron/v3`](https://github.com/robfig/cron) - Cron-расписание

## 🛠️ Разработка

### Структура проекта

```
gym-bot/
├── main.go              # Точка входа: инициализация (errgroup), cron, polling, graceful shutdown
├── database.go          # Интерфейс DB, тип Database, подключение к PostgreSQL, CRUD
├── handlers.go          # Маршрутизация команд, обработчики, бизнес-логика
├── cron.go              # Еженедельный автоопрос каждый понедельник в 08:00 UTC+3
├── constants.go         # Текстовые константы, список дней недели
├── go.mod               # Go модули
├── go.sum               # Контрольные суммы модулей
├── Dockerfile           # Docker образ для бота
├── compose/docker-compose.yml   # Оркестрация контейнеров
├── .env.example         # Пример конфигурации
├── .gitignore           # Игнорируемые файлы
└── README.md            # Документация
```

### Локальная разработка без Docker

Если вы хотите разрабатывать без Docker:

1. Установите PostgreSQL локально
2. Создайте базу данных:
```sql
CREATE DATABASE gymbot;
CREATE USER gymbot WITH PASSWORD 'password';
GRANT ALL PRIVILEGES ON DATABASE gymbot TO gymbot;
```

3. Настройте `.env`:
```env
TELEGRAM_BOT_TOKEN=ваш_токен
ALLOWED_CHAT_ID=-1001234567890
DATABASE_URL=postgresql://gymbot:password@localhost:5440/gymbot?sslmode=disable
```

4. Запустите бота:
```bash
go run .
```

### Сборка Docker образа вручную

```bash
# Собрать образ
make docker-build

# Запустить контейнер
docker run --env-file .env gym-bot:latest
```

## 🔒 Безопасность

- **Никогда не коммитьте файл `.env`** с реальными данными
- Используйте надёжные пароли для PostgreSQL
- Бот работает только с одним разрешённым чатом (ALLOWED_CHAT_ID)
- Docker образ использует непривилегированного пользователя
- База данных изолирована в Docker сети

## 🐛 Troubleshooting

### Бот не отвечает на команды в группе
- **Проверьте настройки приватности:** [@BotFather](https://t.me/BotFather) → `/mybots` → ваш бот → **Bot Settings** → **Group Privacy** → **Disable**
- Убедитесь, что токен в `.env` файле правильный
- Проверьте, что ID чата в `.env` совпадает с реальным ID группы
- Проверьте логи: `docker compose --project-directory . logs -f bot`

### Бот работает в личных сообщениях, но не в группе
Это означает, что у бота включена Group Privacy или ID чата не совпадает:
- Отключите Group Privacy в [@BotFather](https://t.me/BotFather)
- Проверьте ALLOWED_CHAT_ID в `.env` файле

### Ошибка подключения к базе данных
```bash
# Проверьте статус контейнера PostgreSQL
docker compose --project-directory . -f compose/docker-compose.yml ps

# Проверьте логи PostgreSQL
docker compose --project-directory . -f compose/docker-compose.yml logs postgres

# Убедитесь, что база здорова
docker compose --project-directory . -f compose/docker-compose.yml exec postgres pg_isready -U gymbot
```

### Контейнер постоянно перезапускается
```bash
# Посмотрите логи для диагностики
docker compose --project-directory . logs -f bot

# Проверьте все переменные окружения
docker compose --project-directory . -f compose/docker-compose.yml config
```

### База данных потеряна после перезапуска
Убедитесь, что вы не используете `docker compose --project-directory . -f compose/docker-compose.yml down -v`, который удаляет volumes с данными.

### Как сбросить базу данных
```bash
# Остановить контейнеры и удалить volumes
docker compose --project-directory . -f compose/docker-compose.yml down -v

# Запустить заново (создаст чистую БД)
docker compose --project-directory . -f compose/docker-compose.yml up -d
```

## 📝 Примеры использования

### Типичный workflow

1. **В начале недели:** Участник создаёт опрос
   ```
   /newpoll
   ```

2. **В течение недели:** Участники голосуют за дни, когда планируют пойти в зал

3. **Позже:** Если нужно найти опрос в истории
   ```
   /findpoll
   ```

4. **Следующая неделя:** Процесс повторяется

### Как узнать ID чата

Несколько способов:

1. **Через @userinfobot:**
   - Добавьте бота в группу
   - ID будет в сообщении бота

2. **Через @raw_data_bot:**
   - Добавьте бота в группу
   - Отправьте любое сообщение
   - Бот покажет полную информацию

3. **Через логи вашего бота:**
   - Временно уберите проверку ALLOWED_CHAT_ID
   - Отправьте команду в группе
   - ID будет в логах

## Доставка образов из CI

Рекомендуемый путь релиза: собирать образ после проверок в CI и разворачивать
опубликованный образ по digest. На VM не пересобирать приложение из checkout:
так запускается тот же артефакт, который прошёл проверки.

CI выполняет существующие проверки, затем `make image-build` собирает все собственные
образы из build-секций Compose без runtime-секретов и соседних checkout.
Push main публикует **эти же** образы в GHCR через `make image-publish`.
Тег содержит SHA, run ID и attempt; artifact `image-release` содержит digest manifest.
Платформа — `linux/amd64` (текущая VM). На PR публикации и доставки нет.
Версии build args по-прежнему приходят из Make, определения сборки — из Compose.

SSH CD передаёт SHA, manifest и временный token в `make ci-deploy`.
Эта цель из `makefiles/ci-deploy.mk` под одной блокировкой проверяет чистый main,
отклоняет устаревший workflow, fast-forward обновляет checkout до проверенного SHA,
временно авторизует Docker в GHCR и вызывает `make image-deploy` из нового Makefile.
Сначала скачиваются все образы, затем `image-activate` сохраняет подготовку/миграции
этого проекта и запускает Compose с `--no-build --pull never --wait`.
Доступ к registry удаляется и при ошибке. `ci-deploy` требует переменные
`DEPLOY_SHA`, `IMAGE_MANIFEST_B64`, `GHCR_USER`, `GHCR_TOKEN` из окружения;
не передавайте token аргументом Make или команды.

Короткий SSH-шаг делает fetch и загружает только `makefiles/ci-deploy.mk` через
`git show` из SHA этого workflow. Поэтому первый деплой работает и со старым
checkout без новой цели. Загруженный файл сам проверяет актуальность SHA и повторно
делает fetch после захвата блокировки; параллельные доставки не обновляют checkout
во время активации. Обычные `make up` и существующие `make deploy` сохраняют
свои прежние контракты; для CI служит отдельная цель `ci-deploy`.

Для CI разрешите GitHub Actions запись Packages. Новые GHCR packages по умолчанию
private; настройте наследование доступа от этого репозитория. Автоматический CD
использует временный `GITHUB_TOKEN` с `packages: read` и временный Docker config,
который удаляется после SSH-шагa. Для ручной доставки нужен отдельный read-only
доступ (`read:packages`) или публичный package; приложение секреты получает только на VM.
VM нужны Python 3.9+, Make, Docker Compose с `--wait`, Git и flock для SSH CD.

`.images/current.json` хранит последний успешный manifest, `previous.json` — предыдущий.
Для отката сначала согласуйте совместимость БД и конфигурации, верните checkout
к `revision` выбранного manifest и выполните `make image-deploy IMAGE_MANIFEST=...`.
Автоматического отката БД/образов нет; не удаляйте нужные digests в GHCR.
`make up` остаётся локальной сборкой из исходников: на VM для релиза используйте
`image-deploy`; `logs`/`ps` читают те же контейнеры. Сторонние образы не пересобираются,
их версии остаются в конфигурации проекта. Внешний registry-кеш между CI runs пока
не настроен. Успешный build не подтверждает runtime: проверяйте readiness и сценарий
сервиса после доставки.

Для диагностики и явных CLI-операций из активного образа используйте
`make image-compose IMAGE_ARGS='ps'`, `IMAGE_ARGS='logs -f'` или
`IMAGE_ARGS='run --rm SERVICE COMMAND'`. Последняя команда автоматически
запрещает сборку и pull. Она подходит для явных миграций/CLI, которые раньше
запускались через локальный Compose-образ. После неуспешной активации
`.images/requested.json` сохраняет manifest попытки; `current.json` остаётся
последним успешным и не гарантирует, что частичный rollout откатился.

Ручной `image-deploy` использует тот же host lock, что SSH CD; не запускайте
обходные Compose-операции одновременно с доставкой. При первом переходе на GHCR
`previous.json` ещё отсутствует: сохраните прежние локальные образы и конфигурацию
для восстановления по runbook до первого релиза.

## Структура Compose

Файлы Docker Compose находятся в `compose/`; основной файл — `compose/docker-compose.yml`.
Команды запускайте из корня проекта через Makefile: он задаёт `--project-directory`
на корень checkout. Build contexts, bind mounts, `env_file` и секреты остаются
относительными к этому корню; имена Compose project, сервисов, сетей и томов сохранены.
При явном вызове Docker используйте `docker compose --project-directory . -f compose/docker-compose.yml ...`.
Если в приватной `.env` или окружении задан `COMPOSE_FILE`, добавьте `compose/`
к прежним именам файлов, сохранив прежний набор overlays и порядок. Для плавного
перехода Make также распознаёт прежние короткие имена, если файлы уже перенесены.
Не меняйте project name и не удаляйте прежние тома для переноса файлов.


## Перенос 19 октября 2026

На Russia восстановлены все 29 записей `polls`; совпали полный хэш таблицы,
sequence, столбцы и ограничения с финальным снимком после остановки старого бота.
Закрытая сессия Bot API перенесена с сохранением владельца и файлов, бот
авторизовался как прежний аккаунт. Данные, `.env` и manifests сохранены в
приватных резервных копиях на обеих VM и вне VM; публично они не публикуются.
Старые контейнеры gym-bot и PostgreSQL остановлены с restart=no.

Новый общий API на Russia направляет Telegram TCP через внутренний SOCKS
существующего reverse-туннеля в Финляндию; маршруты хостов не менялись.
API на server не остановлен и продолжает обслуживать остальные боты.
GitHub Secrets gym-bot направлены на Russia, `DEPLOY_ENABLED=true`;
последующие push main доставляются обычным CI/CD по digest.
