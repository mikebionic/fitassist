# FitAssist — AI Health Assistant

## 1. Обзор проекта

FitAssist — self-hosted приложение для агрегации данных здоровья из Mi Fitness (Xiaomi/Zepp),
их визуализации, анализа с помощью AI (Claude API) и взаимодействия через Telegram-бот.

---

## 2. Архитектура

```
┌─────────────┐     ┌──────────────┐     ┌───────────────┐
│  Vue.js SPA │────▶│  Go Backend  │────▶│  PostgreSQL   │
│  (Frontend) │◀────│  (API + WS)  │◀────│  (Data Store) │
└─────────────┘     └──────┬───────┘     └───────────────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
      ┌──────────┐ ┌────────────┐ ┌──────────┐
      │ Telegram  │ │ Mi Fitness │ │ Claude   │
      │ Bot API   │ │ API (Zepp) │ │ API      │
      └──────────┘ └────────────┘ └──────────┘
```

### Компоненты:
- **Go Backend** — единый сервер: REST API, WebSocket, Telegram bot, cron-задачи синхронизации
- **Vue.js SPA** — дашборд с чартами, админ-панель, AI-чат
- **PostgreSQL** — основное хранилище (пользователи, данные здоровья, сессии AI)
- **Redis** (опционально) — кеш, очереди задач, rate limiting

---

## 3. Технологический стек

### Backend (Go)
| Компонент | Библиотека | Причина |
|-----------|-----------|---------|
| HTTP Router | `chi` или `echo` | Легковесный, middleware-friendly |
| ORM / DB | `sqlx` + raw SQL | Контроль, без магии ORM |
| Migrations | `golang-migrate/migrate` | Стандарт для Go |
| Telegram Bot | `go-telegram/bot` (v2) | Официальный-стиль, поддержка Bot API 7+ |
| Config | `viper` | .env + config.json + env vars |
| Logging | `slog` (stdlib) | Встроен в Go 1.21+ |
| Scheduler | `robfig/cron` | Планирование синхронизации |
| WebSocket | `gorilla/websocket` | AI-чат в реальном времени |
| Claude API | `anthropics/anthropic-sdk-go` | Официальный SDK |
| Auth | `golang-jwt/jwt` | JWT токены для веб-сессий |
| Encryption | `crypto/aes` (stdlib) | Шифрование credentials пользователей |

### Frontend (Vue.js 3)
| Компонент | Библиотека | Причина |
|-----------|-----------|---------|
| Framework | Vue 3 + Composition API | Реактивность, простота |
| Build | Vite | Быстрая сборка |
| UI Kit | PrimeVue или Naive UI | Компоненты + тёмная тема |
| Charts | Apache ECharts (vue-echarts) | Мощные интерактивные графики |
| State | Pinia | Стандарт для Vue 3 |
| Router | Vue Router 4 | SPA навигация |
| HTTP | Axios | HTTP-клиент |
| i18n | vue-i18n | RU/EN локализация |

### Инфраструктура
| Компонент | Технология |
|-----------|-----------|
| Database | PostgreSQL 16 |
| Cache | Redis 7 (опционально) |
| Контейнеризация | Docker + Docker Compose |
| Reverse Proxy | Caddy / Traefik (опционально, для продакшена) |

---

## 4. Структура проекта

```
fitassist/
├── cmd/
│   └── fitassist/
│       └── main.go                 # Entry point
├── internal/
│   ├── config/
│   │   └── config.go               # Viper-based config loading
│   ├── server/
│   │   ├── server.go               # HTTP server setup
│   │   ├── middleware.go            # Auth, CORS, logging
│   │   └── routes.go               # Route registration
│   ├── handler/                     # HTTP handlers
│   │   ├── auth.go                  # Login, register, JWT
│   │   ├── dashboard.go            # Dashboard data endpoints
│   │   ├── health.go               # Health data endpoints
│   │   ├── admin.go                # Admin panel endpoints
│   │   ├── ai.go                   # AI chat endpoints (WebSocket)
│   │   └── export.go               # Export/Import endpoints
│   ├── service/                     # Business logic
│   │   ├── user.go
│   │   ├── health.go               # Health data aggregation
│   │   ├── sync.go                 # Mi Fitness sync orchestrator
│   │   ├── ai.go                   # Claude API interaction + context
│   │   └── telegram.go             # Telegram bot logic
│   ├── repository/                  # Database layer
│   │   ├── user.go
│   │   ├── health.go
│   │   ├── ai_session.go
│   │   └── telegram.go
│   ├── mifit/                       # Mi Fitness API client
│   │   ├── client.go               # HTTP client, auth
│   │   ├── auth.go                 # Login flow (Xiaomi/Huami)
│   │   ├── sleep.go                # Sleep data
│   │   ├── heartrate.go            # Heart rate data
│   │   ├── steps.go                # Steps & activity
│   │   ├── workout.go              # Workouts
│   │   ├── spo2.go                 # Blood oxygen
│   │   └── models.go               # API response models
│   ├── telegram/                    # Telegram bot
│   │   ├── bot.go                  # Bot initialization
│   │   ├── handlers.go             # Command handlers
│   │   └── keyboards.go            # Inline keyboards
│   ├── ai/                          # AI module
│   │   ├── claude.go               # Claude API client
│   │   ├── context.go              # Context/memory management
│   │   ├── prompts.go              # System prompts
│   │   └── analyzer.go             # Health data analysis
│   ├── cron/                        # Scheduled tasks
│   │   └── scheduler.go            # Data sync jobs
│   ├── crypto/                      # Encryption utilities
│   │   └── encrypt.go              # AES encrypt/decrypt for credentials
│   └── model/                       # Domain models
│       ├── user.go
│       ├── health.go
│       └── ai.go
├── migrations/                      # SQL migrations
│   ├── 001_init.up.sql
│   ├── 001_init.down.sql
│   └── ...
├── web/                             # Vue.js frontend
│   ├── src/
│   │   ├── views/
│   │   │   ├── DashboardView.vue   # Main dashboard
│   │   │   ├── SleepView.vue       # Sleep analytics
│   │   │   ├── HeartRateView.vue   # Heart rate charts
│   │   │   ├── WorkoutsView.vue    # Workout history
│   │   │   ├── AIAssistantView.vue # AI chat interface
│   │   │   ├── AdminView.vue       # Admin panel
│   │   │   ├── SettingsView.vue    # User settings
│   │   │   └── LoginView.vue       # Auth page
│   │   ├── components/
│   │   │   ├── charts/             # Chart components
│   │   │   ├── layout/             # Header, Sidebar, etc.
│   │   │   └── common/             # Shared components
│   │   ├── stores/                 # Pinia stores
│   │   ├── api/                    # API client
│   │   ├── router/                 # Vue Router config
│   │   └── App.vue
│   ├── package.json
│   └── vite.config.ts
├── config/
│   ├── config.example.json         # Example config
│   └── config.json                 # Local config (gitignored)
├── deployments/
│   ├── docker-compose.yml          # Full stack
│   ├── docker-compose.dev.yml      # Dev overrides
│   ├── Dockerfile                  # Multi-stage Go + Vue build
│   └── .env.example                # Env vars template
├── scripts/
│   ├── migrate.sh                  # Run migrations
│   └── seed.sh                     # Seed initial admin
├── go.mod
├── go.sum
├── .gitignore
└── README.md
```

---

## 5. Схема базы данных (PostgreSQL)

### Таблицы

```sql
-- Пользователи системы
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(100) NOT NULL UNIQUE,
    email           VARCHAR(255),
    password_hash   VARCHAR(255) NOT NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'user', -- admin, user
    is_active       BOOLEAN DEFAULT true,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

-- Xiaomi/Mi Fitness аккаунты (привязанные к пользователям)
CREATE TABLE mifit_accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mi_email        VARCHAR(255) NOT NULL,       -- зашифровано
    mi_password     BYTEA NOT NULL,               -- AES-зашифровано
    auth_token      TEXT,                          -- текущий токен
    user_id_mi      VARCHAR(100),                 -- Mi user ID
    token_expires   TIMESTAMPTZ,
    last_sync       TIMESTAMPTZ,
    sync_enabled    BOOLEAN DEFAULT true,
    created_at      TIMESTAMPTZ DEFAULT now()
);

-- Telegram чаты
CREATE TABLE telegram_chats (
    id              BIGSERIAL PRIMARY KEY,
    chat_id         BIGINT NOT NULL UNIQUE,       -- Telegram chat ID
    user_id         UUID REFERENCES users(id),     -- привязка к user (после approve)
    username        VARCHAR(255),                  -- Telegram username
    first_name      VARCHAR(255),
    is_approved     BOOLEAN DEFAULT false,         -- ждёт approve в админке
    is_blocked      BOOLEAN DEFAULT false,
    created_at      TIMESTAMPTZ DEFAULT now()
);

-- Данные о шагах и активности
CREATE TABLE health_steps (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    date            DATE NOT NULL,
    total_steps     INTEGER,
    distance_m      INTEGER,                       -- метры
    calories        INTEGER,
    active_minutes  INTEGER,
    stages          JSONB,                         -- детали по периодам
    UNIQUE(user_id, date)
);

-- Данные о сне
CREATE TABLE health_sleep (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    date            DATE NOT NULL,
    sleep_start     TIMESTAMPTZ,
    sleep_end       TIMESTAMPTZ,
    duration_min    INTEGER,                       -- общее время
    deep_min        INTEGER,                       -- глубокий сон
    light_min       INTEGER,                       -- лёгкий сон
    rem_min         INTEGER,                       -- REM фаза
    awake_min       INTEGER,                       -- бодрствование
    stages          JSONB,                         -- подробные стадии
    UNIQUE(user_id, date)
);

-- Данные о пульсе
CREATE TABLE health_heartrate (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    measured_at     TIMESTAMPTZ NOT NULL,
    bpm             SMALLINT NOT NULL,
    type            VARCHAR(20) DEFAULT 'auto',    -- auto, manual, workout
    UNIQUE(user_id, measured_at)
);

-- Индекс для быстрых запросов по дате
CREATE INDEX idx_heartrate_user_date ON health_heartrate(user_id, measured_at);

-- Данные SpO2
CREATE TABLE health_spo2 (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    measured_at     TIMESTAMPTZ NOT NULL,
    value           SMALLINT NOT NULL,             -- процент
    UNIQUE(user_id, measured_at)
);

-- Тренировки
CREATE TABLE health_workouts (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    workout_type    VARCHAR(50) NOT NULL,          -- running, walking, cycling...
    started_at      TIMESTAMPTZ NOT NULL,
    ended_at        TIMESTAMPTZ,
    duration_sec    INTEGER,
    distance_m      INTEGER,
    calories        INTEGER,
    avg_heartrate   SMALLINT,
    max_heartrate   SMALLINT,
    avg_pace        FLOAT,                         -- мин/км
    route_data      JSONB,                         -- GPS трек
    details         JSONB,                         -- дополнительные метрики
    UNIQUE(user_id, started_at)
);

-- Стресс
CREATE TABLE health_stress (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    measured_at     TIMESTAMPTZ NOT NULL,
    value           SMALLINT NOT NULL,             -- 0-100
    UNIQUE(user_id, measured_at)
);

-- AI сессии (хранение контекста)
CREATE TABLE ai_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    title           VARCHAR(500),
    system_prompt   TEXT,                          -- системный промпт с контекстом
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

-- AI сообщения
CREATE TABLE ai_messages (
    id              BIGSERIAL PRIMARY KEY,
    session_id      UUID NOT NULL REFERENCES ai_sessions(id) ON DELETE CASCADE,
    role            VARCHAR(20) NOT NULL,          -- user, assistant, system
    content         TEXT NOT NULL,
    tokens_used     INTEGER,
    created_at      TIMESTAMPTZ DEFAULT now()
);

-- Логи синхронизации
CREATE TABLE sync_logs (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    sync_type       VARCHAR(50),                   -- full, incremental, manual
    status          VARCHAR(20),                   -- success, error, partial
    records_synced  INTEGER DEFAULT 0,
    error_message   TEXT,
    started_at      TIMESTAMPTZ DEFAULT now(),
    finished_at     TIMESTAMPTZ
);

-- Настройки приложения (key-value)
CREATE TABLE app_settings (
    key             VARCHAR(100) PRIMARY KEY,
    value           TEXT,
    updated_at      TIMESTAMPTZ DEFAULT now()
);
```

---

## 6. Mi Fitness API интеграция

### Стратегия
Используем **reverse-engineered API** (на базе huami/zepp endpoints) — нет официального Go SDK,
но endpoints хорошо задокументированы сообществом.

### Аутентификация (2 шага):

**Шаг 1**: Login через Xiaomi Account
```
POST https://api-user.huami.com/registrations/{email}/tokens
Form: state=REDIRECTION&client_id=HuaMi&password={pass}&redirect_uri=...&token=access
```

**Шаг 2**: Получение app_token
```
POST https://account.huami.com/v2/client/login
Form: app_name=com.xiaomi.hm.health&grant_type=access_token&...
→ Возвращает: login_token, app_token, user_id
```

### Endpoints для данных:
| Данные | Endpoint |
|--------|----------|
| Шаги + Сон (сводка) | `GET /v1/data/band_data.json?query_type=summary&date_list={dates}` |
| Пульс | `GET /v1/data/band_data.json?query_type=detail` (бинарные данные) |
| Тренировки (список) | `GET /v1/sport/run/history.json` |
| Тренировка (детали) | `GET /v1/sport/run/detail.json?trackid={id}` |
| SpO2 | Через band_data с decode |

### Синхронизация:
- **Cron job** каждые 30 минут (настраивается)
- **Incremental sync**: запрашиваем только новые данные (по last_sync дате)
- **Full sync**: по запросу из админки или при первом подключении
- Данные декодируются (Base64 + бинарный формат для HR) и сохраняются в PostgreSQL

---

## 7. AI модуль (Claude API)

### Подход:
1. **Контекст пользователя** — при старте AI-сессии формируем system prompt с:
   - Профиль пользователя (возраст, цели, если указаны)
   - Сводка последних 7/30 дней (сон, шаги, пульс, тренировки)
   - Тренды (улучшение/ухудшение)
   - Предыдущие рекомендации (из прошлых сессий)

2. **Сохранение контекста** — вся переписка сохраняется в `ai_sessions` + `ai_messages`.
   При продолжении сессии — восстанавливаем полный контекст.

3. **Автоматический summarize** — кнопка "AI Summary" генерирует:
   - Анализ последних данных
   - Сравнение с предыдущим периодом
   - Конкретные рекомендации (сон, активность, тренировки)
   - Предупреждения (аномальный пульс, мало сна и т.д.)

4. **WebSocket** для real-time streaming ответов Claude в веб-интерфейсе

### Промпт-инженеринг:
```
System: Ты — персональный AI-ассистент по здоровью и фитнесу.
Ты анализируешь данные пользователя с носимых устройств (Mi Band/Amazfit).

Данные пользователя за последние 7 дней:
{health_data_json}

Тренды за 30 дней:
{trends_json}

Твоя задача:
1. Давать конкретные, персонализированные рекомендации
2. Замечать паттерны (плохой сон → низкая активность)
3. Хвалить за прогресс
4. Предупреждать о рисках (высокий пульс покоя, мало сна)
5. Предлагать конкретные действия на следующий день/неделю
```

---

## 8. Telegram Bot

### Команды:
| Команда | Описание |
|---------|----------|
| `/start` | Регистрация чата, приветствие |
| `/link` | Привязать Mi Fitness аккаунт (запрос email + password) |
| `/today` | Сводка за сегодня (шаги, сон, пульс) |
| `/week` | Сводка за неделю |
| `/sleep` | Детали сна за последнюю ночь |
| `/hr` | Текущий пульс / средний за день |
| `/workout` | Последняя тренировка |
| `/ai` | Запрос к AI-ассистенту (свободный текст) |
| `/summary` | AI-summary за неделю |
| `/settings` | Настройки уведомлений |

### Уведомления (push):
- Утренний отчёт (сон прошлой ночи + план на день)
- Вечерний отчёт (итоги дня)
- AI-алерты (аномальный пульс, мало движения)
- Поздравления (достижение цели по шагам, рекорд)

### Flow привязки аккаунта:
1. Пользователь пишет `/start` → бот регистрирует chat_id в БД (is_approved=false)
2. Админ видит в админ-панели новый чат → approve
3. Пользователь пишет `/link` → бот запрашивает email, затем password
4. Credentials шифруются AES и сохраняются в `mifit_accounts`
5. Бот делает тестовый login → подтверждает привязку
6. Запускается первичная full sync

---

## 9. Web Frontend

### Страницы:

#### Dashboard (главная)
- Карточки: шаги сегодня, сон, пульс, калории
- Мини-графики за неделю
- Кнопка "AI Summary" → открывает AI-чат с автоматическим summarize
- Последняя тренировка

#### Sleep (Сон)
- Calendar heatmap (качество сна по дням)
- График стадий сна (deep/light/REM/awake)
- Тренд: среднее время сна за месяц
- Время засыпания / пробуждения

#### Heart Rate (Пульс)
- Line chart (пульс за день, за неделю)
- Зоны пульса (покой, лёгкая активность, кардио, пик)
- Средний пульс покоя — тренд
- Min/Max за период

#### Workouts (Тренировки)
- Список тренировок с фильтрами (тип, дата)
- Детали тренировки (карта маршрута если GPS, пульс во время)
- Статистика: общее за месяц (км, минуты, калории)

#### AI Assistant
- Чат-интерфейс (как ChatGPT)
- Streaming ответов через WebSocket
- Список сессий (история разговоров)
- Предустановленные вопросы ("Как улучшить сон?", "План тренировок")

#### Admin Panel
- Список пользователей
- Управление Telegram-чатами (approve/block)
- Логи синхронизации
- Export/Import базы данных
- App settings (частота синхронизации, API ключи)
- Мониторинг (статус API, ошибки)

#### Settings
- Профиль пользователя (имя, возраст, цели)
- Привязка Mi Fitness аккаунта
- Настройки уведомлений Telegram
- Тема (light/dark)
- Язык (RU/EN)

---

## 10. Конфигурация

### config.json
```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "mode": "production"
  },
  "database": {
    "host": "postgres",
    "port": 5432,
    "name": "fitassist",
    "user": "fitassist",
    "password": "",
    "ssl_mode": "disable"
  },
  "redis": {
    "enabled": false,
    "host": "redis",
    "port": 6379
  },
  "telegram": {
    "enabled": true,
    "bot_token": "",
    "admin_chat_ids": []
  },
  "claude": {
    "api_key": "",
    "model": "claude-sonnet-4-5-20250929",
    "max_tokens": 4096
  },
  "mifit": {
    "sync_interval_minutes": 30,
    "api_base_url": "https://api-mifit-de2.huami.com"
  },
  "security": {
    "jwt_secret": "",
    "encryption_key": "",
    "cors_origins": ["http://localhost:5173"]
  },
  "admin": {
    "initial_username": "admin",
    "initial_password": ""
  }
}
```

### .env (для Docker / переопределения)
```env
DB_HOST=postgres
DB_PORT=5432
DB_NAME=fitassist
DB_USER=fitassist
DB_PASSWORD=strongpassword123

TELEGRAM_BOT_TOKEN=123456:ABC...
CLAUDE_API_KEY=sk-ant-...
JWT_SECRET=random-secret
ENCRYPTION_KEY=32-byte-hex-key

ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin123
```

**Приоритет**: env vars > config.json > defaults

---

## 11. Docker Compose

```yaml
version: "3.9"

services:
  app:
    build:
      context: .
      dockerfile: deployments/Dockerfile
    ports:
      - "${APP_PORT:-8080}:8080"
    env_file:
      - .env
    volumes:
      - ./config:/app/config:ro
      - app-data:/app/data          # экспорты, бэкапы
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: ${DB_NAME:-fitassist}
      POSTGRES_USER: ${DB_USER:-fitassist}
      POSTGRES_PASSWORD: ${DB_PASSWORD:-changeme}
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "${DB_PORT_EXPOSE:-5432}:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-fitassist}"]
      interval: 5s
      timeout: 3s
      retries: 5
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    profiles: ["full"]               # только если нужен
    volumes:
      - redisdata:/data
    restart: unless-stopped

volumes:
  pgdata:
  redisdata:
  app-data:
```

### Dockerfile (multi-stage)
```dockerfile
# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.22-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o /fitassist ./cmd/fitassist

# Stage 3: Runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend /fitassist .
COPY --from=backend /app/migrations ./migrations
EXPOSE 8080
CMD ["./fitassist"]
```

---

## 12. Export / Import / Миграции

### Database Migrations
- `golang-migrate` — up/down миграции, версионированные SQL файлы
- Автоматический запуск при старте приложения (опционально, настраивается)
- CLI команда: `./fitassist migrate up|down|version`

### Export/Import (Админ-панель)
- **Export**: `pg_dump` в SQL или JSON формат через API endpoint
  - `GET /api/admin/export?format=sql` → скачивает SQL дамп
  - `GET /api/admin/export?format=json` → скачивает JSON (все таблицы)
- **Import**: загрузка SQL/JSON файла
  - `POST /api/admin/import` (multipart upload)
  - Валидация схемы перед импортом
  - Backup текущей БД перед импортом (автоматически)

### Volumes
- PostgreSQL данные на Docker volume `pgdata` — переживают перезапуск контейнера
- Volume `app-data` для экспортов и бэкапов

---

## 13. Безопасность

1. **Credentials пользователей** (Mi Fitness email/password):
   - Шифрование AES-256-GCM
   - Ключ шифрования из конфига (не хранится в БД)
   - В БД только зашифрованные данные

2. **JWT Auth**:
   - Access token (15 мин) + Refresh token (7 дней)
   - HttpOnly cookies для refresh token

3. **Telegram**:
   - Approve flow — только одобренные чаты получают данные
   - Rate limiting на команды

4. **API**:
   - CORS настраивается через конфиг
   - Rate limiting (по IP или по user)
   - Input validation на всех endpoints

5. **Admin**:
   - Initial setup (первый запуск — создание admin аккаунта)
   - Доступ к админке только для role=admin

---

## 14. Этапы разработки

### Фаза 1: Фундамент (MVP Backend)
1. Инициализация Go проекта, структура каталогов
2. Config loading (viper: config.json + .env)
3. PostgreSQL подключение + миграции
4. Модели данных + repository layer
5. HTTP server (chi/echo) + базовые middleware
6. JWT auth (register/login)
7. Docker Compose (app + postgres)

### Фаза 2: Mi Fitness интеграция
1. Mi Fitness API client (auth flow)
2. Получение данных: шаги, сон, пульс
3. Декодирование бинарных данных (HR)
4. Сохранение в PostgreSQL
5. Cron-синхронизация
6. API endpoints для получения health data

### Фаза 3: Telegram Bot
1. Bot initialization + webhook/polling
2. Команды: /start, /link, /today, /week
3. Flow привязки аккаунта (credentials через Telegram)
4. Уведомления (утренние/вечерние отчёты)
5. Approve flow (новые чаты)

### Фаза 4: Web Frontend (Dashboard)
1. Vue 3 + Vite + PrimeVue setup
2. Auth pages (login/register)
3. Dashboard (карточки + мини-графики)
4. Sleep page (heatmap + stages chart)
5. Heart Rate page (line charts)
6. Workouts page (list + details)
7. Settings page

### Фаза 5: AI Assistant
1. Claude API integration (anthropic-sdk-go)
2. Context builder (формирование промпта из данных)
3. AI sessions + messages (сохранение в БД)
4. WebSocket endpoint для streaming
5. AI chat в веб-интерфейсе
6. "AI Summary" кнопка на Dashboard
7. AI команды в Telegram (/ai, /summary)

### Фаза 6: Admin Panel + Polish
1. Admin panel (users, chats, sync logs)
2. Export/Import (SQL/JSON)
3. App settings через UI
4. Темная/светлая тема
5. i18n (RU/EN)
6. Multi-stage Dockerfile
7. Документация (README)

---

## 15. API Endpoints (краткий обзор)

```
Auth:
  POST   /api/auth/register
  POST   /api/auth/login
  POST   /api/auth/refresh
  POST   /api/auth/logout

Health Data:
  GET    /api/health/dashboard          # сводка для дашборда
  GET    /api/health/steps?from=&to=
  GET    /api/health/sleep?from=&to=
  GET    /api/health/heartrate?from=&to=
  GET    /api/health/spo2?from=&to=
  GET    /api/health/workouts?from=&to=
  GET    /api/health/workouts/:id
  GET    /api/health/stress?from=&to=

Mi Fitness:
  POST   /api/mifit/link               # привязать аккаунт
  POST   /api/mifit/sync               # ручная синхронизация
  GET    /api/mifit/status              # статус подключения

AI:
  GET    /api/ai/sessions               # список сессий
  POST   /api/ai/sessions               # новая сессия
  WS     /api/ai/chat/:sessionId        # WebSocket чат
  POST   /api/ai/summary                # быстрый summarize

Admin:
  GET    /api/admin/users
  PATCH  /api/admin/users/:id
  GET    /api/admin/chats               # Telegram чаты
  PATCH  /api/admin/chats/:id           # approve/block
  GET    /api/admin/sync-logs
  GET    /api/admin/export?format=sql
  POST   /api/admin/import
  GET    /api/admin/settings
  PUT    /api/admin/settings

User Settings:
  GET    /api/settings/profile
  PUT    /api/settings/profile
  PUT    /api/settings/notifications
```
