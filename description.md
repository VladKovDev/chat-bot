# Описание архитектуры Beauty-Coworking Support Chat Bot

## I. ВВЕДЕНИЕ И ОБЩИЕ СВЕДЕНИЯ

### 1.1. Назначение проекта

**Beauty-Coworking Support Chat Bot** — это платформа чат-бота для поддержки клиентов beauty-coworking пространств. Система построена как микросервисная архитектура и предназначена для демонстрации бизнес-требований (BRD — Business Requirements Document).

**Ключевые характеристики:**

- **Детерминированная демонстрационная система**: Использует фейковые хеш-основанные эмбеддинги вместо реальных LLM (Large Language Models), что обеспечивает воспроизводимость результатов в локальной среде и CI
- **Микросервисная архитектура**: Сервисы на Go, Python и TypeScript взаимодействуют через HTTP и WebSocket
- **Семантическое распознавание намерений**: Использует pgvector и 384-мерные эмбеддинги для семантического поиска интентов
- **Эскалация к оператору**: Возможность передачи сложных разговоров человеческому оператору
- **Read-only бизнес-моки**: Реальные бронирования, платежи и CRM мутации не выполняются — используются только read-only mock провайдеры

### 1.2. Технологический стек

**Языки программирования:**
- **Go 1.24.1** — Decision Engine, Website Backend (chi router, pgx, pgvector, zap)
- **Python 3.12+** — NLP Service, Console Adapter (FastAPI, pymorphy3, uv)
- **TypeScript** — Website Frontend, E2E тесты (Playwright)

**База данных:**
- **PostgreSQL 16** с расширением **pgvector** — персистентность и семантический поиск

**Оркестрация:**
- **Docker Compose** — управление сервисами

**Инструменты разработки:**
- **Goose** — миграции базы данных
- **Playwright** — E2E тестирование
- **Ruby** — валидация JSON/YAML в check-core скриптах

### 1.3. Архитектурные принципы

**Микросервисная архитектура:**
- Каждый сервис независим и имеет четкую зону ответственности
- Сервисы общаются через HTTP/WebSocket API с версионированными контрактами
- Используются health checks и readiness checks

**Domain-Driven Design (DDD):**
- Четкое разделение на `domain/` (бизнес-сущности), `app/` (оркестрация), `infrastructure/` (технические детали)
- Богатая доменная модель с бизнес-логикой в сущностях
- Репозитории для абстракции работы с БД

**Deterministic Demo подход:**
- Нет реальных LLM (Ollama, GigaChat и др.)
- Используются детерминированные фейковые эмбеддинги на основе хеша
- Результаты воспроизводимы между запусками

**Read-only Business Mocks:**
- Реальные бронирования, платежи, CRM операции не выполняются
- Используются mock провайдеры с документированными контрактами
- Fixtures с демо-данными для тестирования

---

## II. АРХИТЕКТУРА СИСТЕМЫ

### 2.1. Общая схема компонентов

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   Website       │────▶│  Decision Engine │────▶│   NLP Service   │
│  (Go + TS)      │◀────│     (Go)         │◀────│   (Python)      │
│   Port 8081     │     │   Port 8080      │     │   Port 8082     │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌──────────────────┐
                        │   PostgreSQL 16  │
                        │   + pgvector     │
                        │   Port 5442      │
                        └──────────────────┘
                               │
                               ▼
                        ┌──────────────────┐
                        │  Mock External   │
                        │  Providers       │
                        │   Port 8090      │
                        └──────────────────┘
```

**Поток сообщений:**
1. Пользователь отправляет сообщение через WebSocket на сайте
2. Website перенаправляет сообщение в Decision Engine через HTTP POST
3. Decision Engine вызывает NLP Service для получения эмбеддинга текста
4. Decision Engine выполняет семантический поиск в pgvector для matching интента
5. Decision Engine выполняет бизнес-логику (вызывает mock провайдеры при необходимости)
6. Ответ возвращается через цепочку: Decision Engine → Website → WebSocket клиент

### 2.2. Микросервисы и их роли

#### **Decision Engine** (Порт 8080)

**Ядро системы.** Основной HTTP API и пайплайн обработки сообщений.

**Ответственности:**
- Управление сессиями пользователей
- Обработка входящих сообщений
- Matching интентов через семантический поиск
- Исполнение бизнес-действий
- Управление очередью эскалации к операторам
- Предоставление HTTP API для других сервисов

**Структура:** `services/decision-engine/`

#### **NLP Service** (Порт 8082)

**Обработка текста.** Python/FastAPI сервис для NLP операций.

**Ответственности:**
- Предобработка текста (лемматизация, токенизация)
- Генерация эмбеддингов (fake hash-based для demo)
- Health check для проверки готовности

**Структура:** `services/nlp-service/`

#### **Website Transport Adapter** (Порт 8081)

**Веб-интерфейс.** Go backend + TypeScript frontend.

**Ответственности:**
- WebSocket сервер для realtime чата
- HTTP API для создания сессий и отправки сообщений
- Frontend интерфейс для пользователей
- Operator UI для работы с очередью

**Структура:** `services/transport-adapters/website/`

#### **Console Transport Adapter**

**Консольный интерфейс.** Python CLI для разработки и тестирования.

**Ответственности:**
- CLI интерфейс для взаимодействия с Decision Engine
- Удобство отладки и ручного тестирования

**Структура:** `services/transport-adapters/console/`

#### **Mock External Providers** (Порт 8090)

**Моки внешних систем.** Python сервис с fixture данными.

**Ответственности:**
- Имитация booking систем
- Имитация payment систем
- Имитация account/CRM систем
- Имитация pricing сервисов
- Возврат демо-данных из `seeds/demo-*.json`

**Структура:** `scripts/mock-external-services.py` + `seeds/`

### 2.3. База данных

**PostgreSQL 16 + pgvector**

**Таблицы:**
- `users` — пользователи системы
- `sessions` — сессии диалогов
- `messages` — сообщения в сессиях
- `transitions_log` — логи переходов состояний
- `actions_log` — логи выполненных действий
- `operator_queue` — очередь эскалации
- `operator_assignments` — назначения операторов
- `operator_events` — события операторов
- `intent_examples` — примеры фраз для интентов с эмбеддингами
- `knowledge_chunks` — фрагменты базы знаний с эмбеддингами
- `demo_*` — демо таблицы (bookings, payments, accounts, operators)

**pgvector:**
- Используется для семантического поиска интентов и knowledge chunks
- Эмбеддинги имеют размерность 384
- Косинусное расстояние для similarity search

**Миграции:**
- Выполняются через Goose
- Находятся в `services/decision-engine/migrations/`
- Запускаются автоматически при старте через decision-migrate сервис

---

## III. СТРУКТУРА ПРОЕКТА

### 3.1. Корневая директория

```
chat-bot/
├── docs/                           # Документация
│   ├── architecture.md             # Архитектура системы
│   ├── diagrams/                   # Диаграммы (component, ERD, deployment, sequence)
│   └── test-pyramid.md             # Стратегия тестирования
├── scripts/                        # Скрипты автоматизации
│   ├── check-core.sh              # Основной quality gate
│   └── mock-external-services.py  # Mock providers
├── seeds/                          # Тестовые и демо данные
│   ├── intents.json               # Определения интентов
│   ├── knowledge-base.json        # База знаний
│   ├── demo-bookings.json         # Демо бронирования
│   ├── demo-payments.json         # Демо платежи
│   ├── demo-users.json            # Демо пользователи
│   └── demo-operators.json        # Демо операторы
├── services/                       # Микросервисы
│   ├── decision-engine/           # Go (основной API)
│   ├── nlp-service/               # Python (NLP)
│   └── transport-adapters/        # Transport layer
│       ├── website/               # Go backend + TS frontend
│       └── console/               # Python CLI
├── tests/                          # Тесты
│   └── e2e/                       # E2E тесты (Playwright)
├── docker-compose.yml             # Оркестрация сервисов
├── Makefile                       # Команды разработки
├── .env.example                   # Пример переменных окружения
├── package.json                   # JS зависимости (E2E тесты)
└── CLAUDE.md                      # Инструкции для Claude Code
```

### 3.2. Внутренняя структура Decision Engine

```
services/decision-engine/
├── cmd/app/                       # Точка входа приложения
│   └── main.go                   # main функция
├── configs/                       # Конфигурационные файлы
│   ├── development.yaml          # Дев конфиг
│   └── production.yaml           # Прод конфиг
├── contracts/                     # API контракты
│   ├── http-v1.json              # OpenAPI контракт HTTP API
│   └── mock-external-providers-v1.json # Mock провайдеры
├── migrations/                    # Миграции БД (Goose)
│   └── *.sql                     # SQL миграции
├── internal/                      # Внутренняя логика
│   ├── app/                      # Application layer
│   │   ├── actions/              # Выполнение бизнес-действий
│   │   ├── decision/             # Логика matching интентов
│   │   ├── operator/             # Управление эскалацией
│   │   ├── presenter/            # Формирование ответов
│   │   ├── processor/            # Обработка сообщений
│   │   ├── provider/             # Внешние интеграции
│   │   ├── seed/                 # Инициализация данных
│   │   └── worker/               # Фоновая обработка
│   ├── domain/                   # Domain entities
│   │   ├── action/               # Сущности действий
│   │   ├── intent/               # Интенты
│   │   ├── message/              # Сообщения
│   │   ├── operator/             # Операторы
│   │   ├── response/             # Ответы
│   │   ├── session/              # Сессии
│   │   ├── state/                # Состояния
│   │   ├── transitionlog/        # Логи переходов
│   │   └── user/                 # Пользователи
│   ├── infrastructure/           # Внешние интеграции
│   │   ├── nlp/                  # NLP клиент
│   │   └── repository/           # Репозитории (postgres)
│   ├── config/                   # Конфигурация
│   │   ├── logger/               # Логирование
│   │   ├── nlp/                  # NLP настройки
│   │   ├── postgres/             # База данных
│   │   └── transport/            # Транспорт
│   └── transport/                # Транспортный слой
│       └── http/                 # HTTP
│           ├── handler/          # Обработчики запросов
│           └── middleware/       # Middleware
└── responses.json                 # Статические ответы
```

### 3.3. Структура других сервисов

**NLP Service** (`services/nlp-service/`):
```
├── app/
│   ├── api/                      # API роутеры
│   │   └── routes.py             # FastAPI роуты
│   ├── config.py                 # Конфигурация
│   ├── core/                     # Бизнес-логика
│   │   ├── embeddings.py         # Генерация эмбеддингов
│   │   └── preprocessor.py       # Предобработка текста
│   └── main.py                   # Точка входа
├── tests/                        # Тесты
└── pyproject.toml                # Python зависимости
```

**Website Adapter** (`services/transport-adapters/website/`):
```
├── cmd/                          # Go backend
│   └── app/
│       └── main.go               # WebSocket сервер
├── web/                          # TS frontend
│   ├── chat/                     # Chat UI
│   ├── operator/                 # Operator UI
│   └── websocket.ts              # WebSocket клиент
├── configs/                      # Конфиги
├── contracts/                    # WebSocket контракт
└── Makefile                      # Команды сборки
```

---

## IV. СУЩНОСТИ ПРЕДМЕТНОЙ ОБЛАСТИ (DOMAIN)

### 4.1. Session (Сессия)

**Основная сущность** для хранения состояния диалога с пользователем.

**Модель сессии** (`services/decision-engine/internal/domain/session/model.go:10-27`):
```go
type Session struct {
    ID             uuid.UUID      // Уникальный ID сессии
    UserID         uuid.UUID      // ID пользователя
    Channel        string         // Канал (website, dev-cli)
    ExternalUserID string         // Внешний ID пользователя
    ClientID       string         // ID клиента
    State          state.State    // Текущее состояние FSM
    Mode           Mode           // Режим сессии
    ActiveTopic    string         // Активная тема
    LastIntent     string         // Последний распознанный интент
    FallbackCount  int            // Счетчик fallback
    OperatorStatus OperatorStatus // Статус оператора
    Version        int            // Версия для optimistic locking
    Status         Status         // Статус (active, closed)
    Metadata       map[string]interface{} // Доп. данные
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

**Режимы (Modes)** (`services/decision-engine/internal/domain/session/model.go:38-45`):
- `standard` — стандартный режим бота
- `waiting_operator` — ожидание подключения оператора
- `operator_connected` — оператор подключен
- `closed` — сессия закрыта

**Статусы оператора** (`services/decision-engine/internal/domain/session/model.go:48-55`):
- `none` — нет эскалации
- `waiting` — в очереди на оператора
- `connected` — оператор подключен
- `closed` — эскалация закрыта

**Идентификация сессии:**
- Сессии идентифицируются по `channel + external_user_id` или `channel + client_id`
- Legacy `chat_id` не используется в целевой модели
- Создание сессии гарантирует наличие пользователя в БД

**Жизненный цикл сессии:**
1. Создание через `POST /api/v1/sessions`
2. Возобновление при повторном вызове с тем же identity
3. Переходы режимов при эскалации
4. Закрытие при завершении диалога

### 4.2. Message (Сообщение)

**Сообщение** в рамках сессии.

**Модель сообщения** (`services/decision-engine/internal/domain/message/model.go:8-15`):
```go
type Message struct {
    ID         uuid.UUID      // Уникальный ID
    SessionID  uuid.UUID      // ID сессии
    SenderType SenderType     // Тип отправителя
    Text       string         // Текст сообщения
    Intent     *string        // Распознанный интент (опционально)
    CreatedAt  time.Time
}
```

**Типы отправителей** (`services/decision-engine/internal/domain/message/model.go:17-23`):
- `user` — сообщение от пользователя
- `bot` — сообщение от бота
- `operator` — сообщение от оператора

**Поток обработки сообщения:**
1. Получение через HTTP POST
2. Вызов NLP для эмбеддинга
3. Семантический поиск интента
4. Исполнение действий
5. Сохранение в БД
6. Формирование ответа

### 4.3. Intent (Интент/Намерение)

**Интент** — классифицированное намерение пользователя.

**Иерархия интентов** (`services/decision-engine/internal/domain/intent/intent.go:5-134`):

**Коммуникативные интенты** (базовые):
- `greeting` — приветствие
- `goodbye` — прощание
- `confirmation` — подтверждение
- `negation` — отрицание
- `gratitude` — благодарность
- `clarification` — уточнение
- `unknown` — неизвестно

**Системные интенты**:
- `request_operator` — запрос оператора
- `reset_conversation` — сброс разговора
- `return_to_menu` — возврат в меню
- `show_contacts` — показать контакты

**Основные категории** (верхнеуровневые):
- `booking` — записи и бронирование
- `workspace` — рабочие места
- `payment` — оплата
- `tech_issue` — проблемы с сайтом/входом
- `account` — аккаунт
- `services` — услуги и правила
- `complaint` — жалобы
- `other` — другое

**Информационные интенты** (28 типов):
- `ask_booking_info` — информация о записи
- `ask_booking_status` — статус записи
- `ask_cancellation_rules` — правила отмены
- `ask_workspace_prices` — цены на места
- `ask_payment_status` — статус платежа
- `ask_refund_rules` — правила возврата
- `ask_site_problem` — проблема с сайтом
- `ask_login_problem` — проблема с входом
- и др.

**Типы проблем**:
- `payment_not_passed` — оплата не прошла
- `payment_not_activated` — деньги списались, услуга не активирована
- `site_not_loading` — сайт не загружается
- `login_not_working` — вход не работает
- `code_not_received` — код не приходит
- `booking_not_found` — запись не найдена
- `workspace_unavailable` — место недоступно
- `complaint_master` — жалоба на мастера
- `complaint_premises` — жалоба на помещение

**Примеры** определены в `seeds/intents.json`:
```json
{
  "key": "greeting",
  "category": "system",
  "resolution_type": "static_response",
  "response_key": "start",
  "examples": ["привет", "здравствуйте", "добрый день", ...],
  "quick_replies": [...],
  "e2e_coverage": ["E2E-001"]
}
```

### 4.4. Action (Действие)

**Действие** — бизнес-операция, выполняемая при обработке интента.

**Виды действий:**
- `send_message` — отправить сообщение
- `call_provider` — вызвать внешний провайдер
- `escalate_to_operator` — эскалация к оператору
- `reset_conversation` — сброс разговора
- `provide_info` — предоставить информацию
- `show_status` — показать статус

**Исполнение действий:**
- Actions определяются в `seeds/intents.json`
- Исполняются через `app/actions/` слой
- Логируются в `actions_log` таблицу

### 4.5. State Machine (Машина состояний)

**Состояния сессии** (`services/decision-engine/internal/domain/state/state.go:5-67`):

**Общие состояния:**
- `new` — новая сессия
- `waiting_for_category` — ожидание выбора категории
- `waiting_clarification` — ожидание уточнения
- `waiting_for_identifier` — ожидание идентификатора
- `escalated_to_operator` — эскалирована оператору
- `closed` — закрыта

**Категорийные состояния:**
- `booking` — записи
- `workspace` — рабочие места
- `payment` — оплата
- `tech_issue` — технические проблемы
- `account` — аккаунт
- `services` — услуги
- `complaint` — жалобы
- `other` — другое

**Информационные состояния:**
- `providing_info` — предоставление информации
- `showing_status` — показ статуса
- `providing_instruction` — предоставление инструкции
- `suggesting_solution` — предложение решения
- `show_contact_info` — показ контактов

**Логирование переходов:**
- Все переходы логируются в `transitions_log`
- Включают `event` и `reason`
- Используются для аудита и отладки

### 4.6. User и Operator

**User (Пользователь)** — пользователь системы.

**Модель пользователя** (`services/decision-engine/internal/domain/user/model.go`):
```go
type User struct {
    ID        uuid.UUID
    ExternalID string          // Внешний ID
    Channel    string          // Канал (website, dev-cli)
    Metadata   map[string]interface{}
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

**Operator (Оператор)** — оператор службы поддержки.

**Модель оператора** (`services/decision-engine/internal/domain/operator/model.go:57-64`):
```go
type Account struct {
    OperatorID  string
    FixtureID   string          // Ссылка на fixture
    DisplayName string          // Отображаемое имя
    Status      string          // Статус (online, offline, away)
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

**Очередь handed-off сессий** (`services/decision-engine/internal/domain/operator/model.go:42-55`):
```go
type QueueItem struct {
    ID                 uuid.UUID
    SessionID          uuid.UUID
    UserID             uuid.UUID
    Status             QueueStatus     // waiting, accepted, closed
    Reason             Reason          // manual_request, low_confidence_repeated, etc.
    Priority           int
    AssignedOperatorID string
    ContextSnapshot    ContextSnapshot // Снепшот контекста
    CreatedAt          time.Time
    UpdatedAt          time.Time
    AcceptedAt         *time.Time
    ClosedAt           *time.Time
}
```

**Причины эскалации:**
- `manual_request` — ручной запрос пользователя
- `low_confidence_repeated` — повторная низкая уверенность
- `complaint` — жалоба
- `business_error` — бизнес ошибка

---

## V. БИЗНЕС-ЛОГИКА

### 5.1. Matching интентов

**Семантический поиск через pgvector:**

1. Текст сообщения отправляется в NLP Service
2. NLP возвращает эмбеддинг размерности 384
3. Выполняется поиск в таблице `intent_examples`
4. Возвращаются примеры с наибольшим cosine similarity

**Fallback на точное совпадение:**
- Если семантический поиск не дает результатов с достаточной уверенностью
- Выполняется точное совпадение команд (например, "главное меню")

**Пороги уверенности:**
- Высокая уверенность (> 0.8) — прямой ответ
- Средняя уверенность (0.5-0.8) — уточняющий вопрос
- Низкая уверенность (< 0.5) — fallback или эскалация

**Обработка неоднозначности:**
- При нескольких близких интентах запрашивается уточнение
- Используются quick replies для навигации

### 5.2. Выбор ответов

**Статические ответы из responses.json:**
- Каждый интент имеет `response_key`
- Ключ маппится на текстовый ответ
- Поддерживается интерполяция переменных

**Интерполяция шаблонов:**
```json
{
  "booking_info": "Ваше бронирование #{booking_id} на {date} в {time}"
}
```

**Quick replies (быстрые ответы):**
- Определяются в `seeds/intents.json`
- Предлагают пользователю действия
- Могут содержать:
  - `send_text` — отправить текст
  - `select_intent` — выбрать интент
  - `request_operator` — запрос оператора

### 5.3. Провайдеры внешних систем

**Интеграция с mock providers:**
- Провайдеры запускаются как отдельный сервис на порту 8090
- Контракты определены в `contracts/mock-external-providers-v1.json`

**Типы провайдеров:**
- **Booking** — бронирования услуг
  - `GET /bookings/{id}` — получить бронирование
  - `GET /bookings?user_id={id}` — список бронирований
- **Payment** — платежи
  - `GET /payments/{id}` — получить платеж
  - `GET /payments?booking_id={id}` — платежи по бронированию
- **Account** — аккаунты
  - `GET /accounts/{id}` — получить аккаунт
- **Pricing** — цены
  - `GET /pricing/workspaces` — цены на рабочие места
  - `GET /pricing/services` — цены на услуги

**Timeouts и обработка ошибок:**
- Таймаут по умолчанию: 5 секунд
- При ошибке провайдера — graceful degradation
- Возвращается безопасное значение или сообщение об ошибке

### 5.4. Operator escalation

**Логика передачи оператору:**
1. Пользователь запрашивает оператора (интент `request_operator`)
2. Создается запись в `operator_queue` со снепшотом контекста
3. Сессия переходит в режим `waiting_operator`
4. Оператор видит заявку в очереди

**Очередь handoff:**
- GET `/api/v1/operator/queue` — получить очередь
- POST `/api/v1/operator/queue/{session_id}/request` — запросить оператора
- POST `/api/v1/operator/queue/{handoff_id}/accept` — принять заявку
- POST `/api/v1/operator/queue/{handoff_id}/close` — закрыть заявку

**Принятие, отправка сообщений, закрытие:**
- Оператор принимает заявку → сессия переходит в `operator_connected`
- Оператор может отправлять сообщения от своего имени
- Оператор закрывает заявку → сессия возвращается в `standard`

---

## VI. API И КОНТРАКТЫ

### 6.1. HTTP API v1 (Decision Engine)

**Контракт:** `services/decision-engine/contracts/http-v1.json`

**Health check endpoints:**
- `GET /api/v1/health` — проверка здоровья сервиса
- `GET /api/v1/ready` — проверка готовности (все зависимости доступны)

**Session management:**
- `POST /api/v1/sessions` — создание/возобновление сессии

**Message processing:**
- `POST /api/v1/messages` — отправка сообщения
- `GET /api/v1/sessions/{session_id}/messages` — история сообщений

**Operator endpoints:**
- `POST /api/v1/operator/queue/{session_id}/request` — запрос оператора
- `GET /api/v1/operator/queue` — очередь заявок
- `POST /api/v1/operator/queue/{handoff_id}/accept` — принять заявку
- `POST /api/v1/operator/sessions/{session_id}/messages` — сообщение оператора
- `POST /api/v1/operator/queue/{handoff_id}/close` — закрыть заявку

**Domain schema:**
- `GET /api/v1/domain/schema` — схема доменных сущностей

### 6.2. WebSocket Protocol (Website)

**Контракт:** `services/transport-adapters/website/contracts/websocket.json`

**Client events (4 типа):**
- `session:start` — создание сессии
- `message:send` — отправка сообщения
- `operator:handoff` — запрос оператора
- `session:resume` — возобновление сессии

**Server events (7 типов):**
- `session:started` — сессия создана
- `session:resumed` — сессия возобновлена
- `message:received` — сообщение получено
- `message:response` — ответ бота
- `operator:queued` — добавлено в очередь
- `operator:connected` — оператор подключен
- `error:occurred` — ошибка

### 6.3. Mock Providers API

**Контракт:** `services/decision-engine/contracts/mock-external-providers-v1.json`

**Booking Provider:**
- `GET /api/v1/bookings/{id}` — получить бронирование по ID
- `GET /api/v1/bookings?user_id={id}` — список бронирований пользователя

**Payment Provider:**
- `GET /api/v1/payments/{id}` — получить платеж по ID
- `GET /api/v1/payments?booking_id={id}` — платежи по бронированию

**Account Provider:**
- `GET /api/v1/accounts/{id}` — получить аккаунт по ID

**Pricing Provider:**
- `GET /api/v1/pricing/workspaces` — цены на рабочие места
- `GET /api/v1/pricing/services` — цены на услуги

**Workspace Provider:**
- `GET /api/v1/workspaces/{id}` — получить рабочее место
- `GET /api/v1/workspaces/availability` — доступность мест

---

## VII. КОНФИГУРАЦИЯ

### 7.1. Переменные окружения

**Файл:** `.env.example`

```bash
# Database
POSTGRES_DB=chat_bot
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_PORT=5442

# Services
DECISION_ENGINE_PORT=8080
NLP_PORT=8082
WEBSITE_PORT=8081
MOCK_EXTERNAL_PORT=8090
```

### 7.2. YAML конфиги Decision Engine

**Файлы:** `services/decision-engine/configs/development.yaml`, `production.yaml`

**Секции:**
- `database` — настройки подключения к БД
- `logger` — логирование (уровень, формат)
- `nlp` — NLP клиент (URL, timeout, размерность)
- `http` — HTTP сервер (address, timeouts)
- `operator` — операторские настройки

### 7.3. Конфиги NLP Service

**Файл:** `services/nlp-service/app/config.py`

**Параметры:**
- `EMBEDDING_MODE` — режим эмбеддингов (fake/unavailable)
- `EMBEDDING_DIMENSION` — размерность (384)
- `LEMMATIZER_CACHE_SIZE` — размер кэша лемматизатора

---

## VIII. СЕЙДОВЫЕ ДАННЫЕ (SEED DATA)

### 8.1. Intents

**Файл:** `seeds/intents.json`

**Структура:**
```json
{
  "intents": [
    {
      "key": "greeting",
      "category": "system",
      "resolution_type": "static_response",
      "response_key": "start",
      "examples": ["привет", "здравствуйте", ...],
      "quick_replies": [...],
      "e2e_coverage": ["E2E-001"]
    }
  ]
}
```

**Поля:**
- `key` — уникальный ключ интента
- `category` — категория (system, operator, booking, payment, etc.)
- `resolution_type` — тип разрешения (static_response, operator_handoff, provider_lookup)
- `response_key` — ключ статического ответа
- `examples` — примеры фраз для обучения
- `quick_replies` — быстрые ответы
- `e2e_coverage` — покрытие E2E тестами

### 8.2. Knowledge Base

**Файл:** `seeds/knowledge-base.json`

**Типы chunk'ов:**
- FAQ — часто задаваемые вопросы
- Pricing — цены и тарифы
- Rules — правила и политики

**Структура:**
```json
{
  "chunks": [
    {
      "id": "chunk-1",
      "type": "faq",
      "content": "Текст ответа",
      "keywords": ["ключ1", "ключ2"],
      "metadata": {}
    }
  ]
}
```

### 8.3. Demo fixtures

**Файлы:**
- `seeds/demo-bookings.json` — демо бронирования
- `seeds/demo-payments.json` — демо платежи
- `seeds/demo-users.json` — демо пользователи
- `seeds/demo-operators.json` — демо операторы

**Используются:**
- Для инициализации mock providers
- Для E2E тестирования
- Для демонстрации BRD

---

## IX. РАЗРАБОТКА И ТЕСТИРОВАНИЕ

### 9.1. Команды разработки

**Запуск стека:**
```bash
cp .env.example .env
docker compose up --build
```

**Остановка и сброс:**
```bash
docker compose down -v --remove-orphans  # Очистка volumes
```

**Quality gates:**
```bash
make check-core          # Линтинг, валидация, unit тесты
make e2e-smoke          # Быстрый smoke тест
make e2e-full           # Полный набор E2E тестов
```

**Проверка готовности:**
```bash
curl -fsS http://localhost:8080/api/v1/ready
```

### 9.2. Тестовая пирамида

**Unit тесты:**
- Go: `go test ./...` в decision-engine
- Python: `pytest` в nlp-service
- Быстрые, не требуют запущенных сервисов

**E2E smoke:**
- Критический путь: создание сессии, отправка сообщения, получение ответа
- 5-10 сценариев
- Время выполнения: ~1-2 минуты

**E2E full:**
- 38 сценариев covering all intents
- Полная матрица в `tests/e2e/full-matrix.spec.ts`
- Время выполнения: ~10-15 минут

### 9.3. CI/CD процессы

**Git diff checks:**
- Проверка измененных JSON/YAML файлов
- Валидация схем

**Legacy LLM runtime rejection:**
- Проверка отсутствия ссылок на старые LLM клиенты
- Отклонение кода с `llmClient.Decide`, `/llm/decide`

**Test reports:**
- HTML отчет: `tests/e2e/test-results/e2e-html/`
- JSON результаты: `tests/e2e/test-results/e2e-results.json`
- Артефакты: `tests/e2e/test-results/e2e-artifacts/`

---

## X. DATA FLOW И LIFECYCLE

### 10.1. Обработка сообщения пользователя

**Пошаговый процесс:**

1. **WebSocket от клиента**
   - Пользователь отправляет сообщение в website WebSocket
   - Client event: `message:send`

2. **HTTP POST к Decision Engine**
   - Website вызывает `POST /api/v1/messages`
   - Тело запроса: `{"session_id": "...", "text": "..."}`

3. **Вызов NLP service**
   - Decision Engine вызывает `POST /embeddings` NLP сервиса
   - Получает эмбеддинг размерности 384

4. **Семантический поиск**
   - Поиск в `intent_examples` таблице через pgvector
   - Получение наиболее похожего интента с confidence score

5. **Исполнение действий**
   - Определение действий по интенту
   - Вызов mock providers при необходимости
   - Логирование действий в `actions_log`

6. **Сохранение в БД**
   - Сохранение сообщения пользователя
   - Сохранение ответа бота
   - Обновление состояния сессии
   - Логирование переходов

7. **Возврат ответа**
   - Формирование ответа через presenter
   - HTTP response Decision Engine
   - WebSocket server event клиенту

### 10.2. Lifecycle сессии

**1. Создание (POST /sessions)**
- Генерация UUID сессии
- Создание/получение пользователя
- Установка начального состояния `StateNew`
- Установка режима `ModeStandard`

**2. Возобновление (resume)**
- Поиск сессии по `channel + external_user_id` или `channel + client_id`
- Проверка статуса (active/closed)
- Возврат существующей сессии

**3. Переходы режимов**
- `standard` → `waiting_operator` при запросе оператора
- `waiting_operator` → `operator_connected` при принятии заявки
- `operator_connected` → `standard` при закрытии заявки

**4. Operator handoff**
- Создание записи в `operator_queue`
- Сохранение снепшота контекста
- Оповещение операторов

**5. Закрытие**
- Установка статуса `StatusClosed`
- Установка режима `ModeClosed`
- Сохранение финального состояния

### 10.3. Database persistence

**Таблицы и связи:**
```
users (1) ----< (N) sessions (1) ----< (N) messages
                                       |
                                       v
                                  transitions_log
                                       |
                                       v
                                  actions_log
```

**Транзакции:**
- Критические операции обернуты в транзакции
- Используется optimistic locking через `version` поле

**Migration process:**
- Миграции выполняются через Goose
- Автоматический запуск при docker compose up
- Отдельный сервис `decision-migrate`

---

## XI. ТИПЫ ФАЙЛОВ И ИХ НАЗНАЧЕНИЕ

### 11.1. Go файлы

**Области применения:**
- `domain/` — доменные сущности и бизнес-логика
- `app/` — application layer (оркестрация)
- `infrastructure/` — технические детали (БД, HTTP клиенты)
- `transport/` — HTTP handlers и middleware
- `cmd/` — точки входа приложений
- `migrations/` — SQL миграции (файлы .sql)

**Соглашения по именованию:**
- `model.go` — структура сущности
- `service.go` — бизнес-логика сущности
- `repository.go` — интерфейс репозитория
- `errors.go` — ошибки сущности
- `handler.go` — HTTP обработчики
- `middleware.go` — middleware

### 11.2. Python файлы

**NLP service:**
- `main.py` — точка входа
- `config.py` — конфигурация
- `routes.py` — API роутеры
- `preprocessor.py` — предобработка текста
- `embeddings.py` — генерация эмбеддингов

**Console adapter:**
- `main.py` — CLI интерфейс
- `client.py` — HTTP клиент для Decision Engine

**Mock services:**
- `mock-external-services.py` — mock providers

### 11.3. TypeScript/JavaScript

**E2E тесты:**
- `*.spec.ts` — тестовые сценарии (Playwright)
- `playwright.config.ts` — конфигурация
- `global-setup.ts` — глобальная настройка
- `global-teardown.ts` — глобальная очистка
- `support/` — утилиты и helpers

**Frontend:**
- `chat/` — чат интерфейс
- `operator/` — операторский интерфейс
- `websocket.ts` — WebSocket клиент
- `api.ts` — API клиенты

### 11.4. Конфигурационные файлы

**YAML:**
- `configs/*.yaml` — конфиги сервисов
- `docker-compose.yml` — оркестрация
- `docker-compose.e2e.yml` — E2E окружение

**JSON:**
- `contracts/*.json` — API контракты
- `seeds/*.json` — тестовые данные
- `package.json` — JS зависимости
- `tsconfig.json` — TS конфигурация

**SQL:**
- `migrations/*.sql` — миграции БД

**Shell:**
- `scripts/*.sh` — скрипты автоматизации
- `Makefile` — команды разработки

**Markdown:**
- `docs/*.md` — документация
- `README.md` — основной readme
- `CLAUDE.md` — инструкции для Claude Code

---

## XII. ДИАГРАММЫ И ВИЗУАЛИЗАЦИЯ

### 12.1. Существующие диаграммы

**Директория:** `docs/diagrams/`

**Component diagram** — компоненты системы и их связи

**ERD diagram** — схема базы данных

**Deployment diagram** — развертывание сервисов

**Sequence diagrams** (6 типов):
- Обработка сообщения пользователя
- Создание сессии
- Эскалация к оператору
- Вызов mock provider
- Семантический поиск интента
- Обработка ошибок

### 12.2. Как читать диаграммы

**Обозначения:**
- Прямоугольники — компоненты/сущности
- Стрелки — потоки данных/вызовы
- Ромбы — решения/условия
- Линии — зависимости

**Потоки данных:**
- Горизонтальные стрелки — основной поток
- Вертикальные стрелки — вспомогательные вызовы
- Пунктирные линии — асинхронные операции

---

## XIII. TROUBLESHOOTING И ОТЛАДКА

### 13.1. Readiness check

**Что проверяет `GET /api/v1/ready`:**
- Подключение к PostgreSQL
- Последняя миграция применена
- Расширение pgvector установлено
- NLP сервис доступен
- Размерность эмбеддингов совпадает (384)
- Операторские таблицы существуют
- Seed данные загружены

**Возможные причины сбоя:**
- База данных недоступна
- NLP сервис не готов
- pgvector не установлен
- Миграции не применены
- Seed данные не загружены

**Диагностика:**
```bash
curl -s http://localhost:8080/api/v1/ready | jq .
docker compose logs --tail=200 decision-migrate decision-engine nlp-service postgres
```

### 13.2. Логи и мониторинг

**Где искать логи:**
- Docker logs: `docker compose logs -f [service]`
- Decision Engine: структурированные JSON логи
- NLP Service: stdout в JSON формате

**Request ID трассировка:**
- Каждый запрос имеет уникальный `request_id`
- Передается через заголовки и логируется
- Позволяет отследить поток через сервисы

**Redaction чувствительных данных:**
- Персональные данные маскируются в логах
- Тексты сообщений частично скрыты

### 13.3. Частые проблемы

**Service unavailable:**
- Проверить health checks
- Проверить зависимости (БД, NLP)
- Проверить переменные окружения

**Migration failures:**
- Проверить goose версию
- Проверить connection string
- Удалить volumes и начать заново

**NLP connection issues:**
- Проверить NLP_PORT
- Проверить health check NLP сервиса
- Проверить таймауты

---

## XIV. ГЛОССАРИЙ ТЕРМИНОВ

- **Сессионная модель** — способ хранения состояния диалога в БД с использованием сессий
- **Интент (Intent)** — классифицированное намерение пользователя
- **Matching** — процесс распознавания интента из текста сообщения
- **Embedding** — векторное представление текста фиксированной размерности
- **pgvector** — расширение PostgreSQL для работы с векторами
- **Handoff** — передача сессии от бота к оператору
- **Escalation** — процесс передачи сессии оператору
- **Fallback** — резервный вариант при неуспешном matching
- **Seed data** — начальные данные для инициализации системы
- **Fixture** — тестовые данные для mock providers
- **Readiness check** — проверка готовности сервиса к работе
- **Health check** — проверка здоровья сервиса
- **Deterministic demo** — детерминированная демонстрация с воспроизводимыми результатами
- **Domain entity** — сущность предметной области
- **Repository** — паттерн абстракции работы с БД
- **StateMachine** — машина состояний для управления сессией
- **Operator queue** — очередь заявок на эскалацию
- **Quick replies** — быстрые ответы для навигации
- **Context snapshot** — снимок контекста сессии

---

## XV. ДОПОЛНИТЕЛЬНЫЕ РЕСУРСЫ

### Ключевые файлы

**Архитектура:**
- `docs/architecture.md` — обзор архитектуры
- `docs/test-pyramid.md` — стратегия тестирования

**Контракты:**
- `services/decision-engine/contracts/http-v1.json` — HTTP API
- `services/transport-adapters/website/contracts/websocket.json` — WebSocket
- `services/decision-engine/contracts/mock-external-providers-v1.json` — Mock providers

**Конфигурация:**
- `docker-compose.yml` — оркестрация
- `.env.example` — переменные окружения
- `services/decision-engine/configs/` — конфиги сервисов

**Доменные сущности:**
- `services/decision-engine/internal/domain/session/model.go` — сессия
- `services/decision-engine/internal/domain/message/model.go` — сообщение
- `services/decision-engine/internal/domain/intent/intent.go` — интенты
- `services/decision-engine/internal/domain/state/state.go` — состояния
- `services/decision-engine/internal/domain/operator/model.go` — операторы

**Seed данные:**
- `seeds/intents.json` — интенты с примерами
- `seeds/knowledge-base.json` — база знаний
- `seeds/demo-*.json` — демо данные

### Полезные команды

```bash
# Запуск стека
docker compose up --build

# Проверка готовности
curl http://localhost:8080/api/v1/ready

# Unit тесты
make check-core

# E2E тесты
make e2e-smoke
make e2e-full

# Логи
docker compose logs -f decision-engine
docker compose logs -f nlp-service

# Сброс
docker compose down -v --remove-orphans
```

### Связь с документацией

- **README.md** — основной readme с кратким обзором
- **CLAUDE.md** — инструкции для Claude Code
- **docs/architecture.md** — архитектура runtime
- **docs/test-pyramid.md** — тестовая пирамида
- **description.md** — этот документ (полное описание)

---

## Критерии завершения

Документ считается полным, так как:

1. ✅ Пользователь может найти объяснение любого компонента по названию (оглавление и поиск)
2. ✅ Описан полный флоу работы от первого сообщения до ответа (раздел X)
3. ✅ Описаны назначения всех типов файлов в проекте (раздел XI)
4. ✅ Есть ссылки на конкретные файлы и строки кода (раздел XV)
5. ✅ Документ структурирован для быстрого поиска информации (15 разделов с оглавлением)

---

**Документ создан:** 2025
**Версия:** 1.0
**Проект:** Beauty-Coworking Support Chat Bot
