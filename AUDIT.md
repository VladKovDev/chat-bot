# Аудит проекта chat-bot

Дата аудита: 2026-05-10  
Репозиторий: `https://github.com/VladKovDev/chat-bot`  
Локальный путь: `/Users/danila/work/my/chat-bot`  
Коммит: `3cffac9 feat: Update decision engine rules and transitions for improved intent handling`

## 1. Executive Summary

Проект представляет собой микросервисный чат-бот для клиентской поддержки. В текущем runtime главный поток такой:

`console/web UI -> decision-engine /decide -> Python LLM /llm/decide -> локальные actions -> responses.json -> клиент`.

Система уже имеет разумное разделение на transport/application/domain/infrastructure, typed contracts внутри Go/Python, Postgres-хранилище сессий и сообщений, LLM service с валидацией ответа модели, circuit breaker на Go-клиенте LLM и базовые health endpoints.

Но проект находится ближе к prototype/MVP, чем к production-ready состоянию. Ключевые проблемы:

- есть две конкурирующие архитектуры принятия решений: текущий LLM-first runtime и неиспользуемые local rules/transitions;
- `/config_llm` разрешает LLM возвращать actions, которые в runtime не зарегистрированы или не реализованы;
- web adapter смешивает всех пользователей в `chat_id=1`;
- actions поиска брони/платежей/аккаунтов являются mock-генераторами, а не интеграциями с реальной БД;
- ответы с плейсхолдерами не интерполируются, пользователь может получить сырой `{service}`, `{date}`, `{status}`;
- lemmatizer-service реализован, но не подключен в основном runtime, плюс есть портовый mismatch;
- отсутствует общий `docker-compose`/CI/root README, поэтому воспроизводимый запуск всей системы не оформлен;
- security gaps: открытый WebSocket Origin, DOM XSS риск в option buttons, `verify=False` в GigaChat client, логирование пользовательского текста/prompt/response.

## 2. Состав Системы

### 2.1 Сервисы

| Сервис | Путь | Стек | Назначение |
|---|---|---|---|
| `decision-engine` | `services/decision-engine` | Go 1.24.1, chi, pgx/sqlc, zap, viper, gobreaker | Центральный HTTP-сервис диалога: сессии, история, вызов LLM, actions, выбор response |
| `llm` | `services/llm` | Python 3.12, FastAPI, Pydantic, structlog, Ollama/GigaChat clients | LLM-классификатор intent/state/actions |
| `lemmatizer-service` | `services/lemmatizer-service` | Python/FastAPI, pymorphy3 | Лемматизация русских токенов |
| `transport-adapters/website` | `services/transport-adapters/website` | Go WebSocket backend + static HTML/CSS/JS | Browser chat UI и WebSocket bridge к decision-engine |
| `transport-adapters/console` | `services/transport-adapters/console` | Python requests CLI | CLI-адаптер для ручного тестирования `/decide` |

### 2.2 Отсутствующие верхнеуровневые элементы

- Корневой `README.md` отсутствует.
- Общий `docker-compose.yml` отсутствует.
- CI workflow не найден.
- Единого контракта всей системы нет; например `services/transport-adapters/website/contracts/websocket.json` пустой.

## 3. Архитектура Runtime

### 3.1 Главный поток сообщения

1. Клиент отправляет текст:
   - console adapter делает `POST http://localhost:8080/decide` (`services/transport-adapters/console/main.py:26`);
   - website frontend открывает `ws://<host>/ws` (`services/transport-adapters/website/frontend/assets/js/websocket.js:97`).
2. Website backend принимает WebSocket message `{type:"message", text}` и отправляет в decision-engine (`services/transport-adapters/website/backend/internal/websocket/handler.go:85`, `:119`).
3. Decision-engine endpoint `/decide` принимает `{text, chat_id?}` (`services/decision-engine/internal/transport/http/handler/decide.go:17`).
4. Если `chat_id` не передан, decision-engine ставит `1` (`services/decision-engine/internal/transport/http/handler/decide.go:64`).
5. `MessageWorker`:
   - грузит/создает session по `chat_id`;
   - сохраняет входящее user message;
   - берет последние 10 сообщений;
   - вызывает LLM `/llm/decide`;
   - выполняет actions;
   - выбирает response key;
   - обновляет session state;
   - возвращает шаблон из `responses.json`.

Основная orchestration-точка: `services/decision-engine/internal/app/worker/message_worker.go:46`.

### 3.2 Decision-engine wiring

Bootstrap вручную собран в `services/decision-engine/internal/app/run.go:73`:

- config init;
- logger config;
- Postgres config/pool;
- LLM config/client;
- HTTP config/router/server;
- repositories;
- session service;
- presenter;
- processor;
- registration of business actions.

Это понятная и простая схема, но сейчас wiring не включает lemmatizer, rule-based classifier, transition engine и часть объявленных actions.

### 3.3 LLM-first против rules/transitions

В репозитории есть:

- `configs/rules.json`;
- `configs/transitions.json`;
- `internal/app/transition/*`;
- `internal/infrastructure/nlp/rule_based/*`;
- normalization/lemmatizer pipeline.

Но текущий `Run()` не подключает rule-based classifier или transition engine. Реальный runtime зависит от Python LLM decision (`services/decision-engine/internal/app/worker/message_worker.go:89`).

Это создает архитектурную неоднозначность: часть файлов выглядит как источник бизнес-логики, но фактически не участвует в принятии решений.

## 4. API И Контракты

### 4.1 Decision-engine

Роуты объявлены в `services/decision-engine/internal/transport/http/router.go:37`:

- `GET /health`;
- `POST /decide`;
- `GET /config_llm`.

`POST /decide`:

```json
{
  "text": "string",
  "chat_id": 1
}
```

Response:

```json
{
  "text": "string",
  "options": ["string"],
  "state": "string",
  "chat_id": 1,
  "success": true,
  "error": ""
}
```

Риск: internal error раскрывается наружу через `failed to process message: %v` (`services/decision-engine/internal/transport/http/handler/decide.go:81`).

### 4.2 LLM service

`POST /llm/decide` принимает прямой payload и wrapper `{"data": ...}` (`services/llm/app/api/routes/decide.py:20`).

Input schema:

```json
{
  "state": "new",
  "summary": "",
  "messages": [
    {"role": "user", "text": "Привет"}
  ]
}
```

Output:

```json
{
  "data": {
    "intent": "greeting",
    "state": "waiting_for_category",
    "actions": []
  }
}
```

Плюс: Python service валидирует JSON object, типы, membership intent/state/action в domain schema (`services/llm/app/services/decide_service.py:90`).

Минус: Go-side `parseDecideResponse` почти не валидирует форму ответа и может молча принять пустые поля (`services/decision-engine/internal/app/worker/message_worker.go:175`).

### 4.3 WebSocket adapter

Backend endpoints:

- `/ws`;
- `/health`;
- `/` static frontend.

DTO:

- input: `{ "type": "message", "text": "..." }`;
- output: `{ "type": "response", "text": "...", "options": [...], "state": "..." }`;
- error: `{ "type": "error", "text": "...", "code": "processing_error" }`.

Фактический контракт живет в Go DTO и JS (`services/transport-adapters/website/backend/internal/dto/websocket.go:3`), а контрактный JSON-файл пустой.

## 5. Бизнес-Логика И Use Cases

### 5.1 Назначение продукта

По `responses.json`, intents/states/actions и mock data бот ориентирован на поддержку сервиса, совмещающего:

- записи на услуги;
- рабочие места/коворкинг;
- платежи;
- проблемы сайта/логина;
- аккаунт;
- услуги, цены, правила, FAQ;
- жалобы;
- перевод к оператору.

Это не просто FAQ-бот: модель должна выбирать state/action, а decision-engine должен уметь искать данные по брони, платежу, аккаунту и передавать контекст оператору. Сейчас эта часть в основном замокана.

### 5.2 Верхнеуровневые сценарии

1. Приветствие и показ меню.
2. Выбор категории.
3. Уточнение вопроса.
4. Предложение решения.
5. Проверка статуса записи/брони/платежа/аккаунта.
6. Обработка проблем оплаты, входа, сайта, кодов подтверждения.
7. Показ правил/цен/FAQ/контактов.
8. Жалоба и эскалация оператору.
9. Закрытие или сброс диалога.

### 5.3 Реальность реализации use cases

| Use case | Реализация сейчас | Статус |
|---|---|---|
| Показ меню/категорий | `responses.json` + ResponseSelector | Частично работает |
| Intent/state/action selection | Python LLM | Работает при доступном LLM и валидном config |
| История диалога | только последние 10 сообщений из DB | Частично |
| Сохранение bot messages | не найдено | Пробел |
| Поиск записи | mock generator `FindBooking` | Не production |
| Поиск брони рабочего места | mock generator | Не production |
| Поиск платежа | mock generator | Не production |
| Поиск аккаунта | mock generator | Не production |
| Валидация идентификатора | regex-based action | Частично |
| Эскалация оператору | TODO, не зарегистрировано | Не работает |
| Reset conversation | advertised action, runtime action не найден/не зарегистрирован | Не работает |
| Лемматизация/rule-based fallback | код есть, runtime не подключен | Не работает в основном пути |

## 6. Данные И Хранилище

### 6.1 Схема БД

Migration `services/decision-engine/migrations/20260216002310_init_tables.sql` создает:

- `sessions`;
- `messages`;
- `users`;
- `transitions_log`;
- `actions_log`.

Сильные стороны:

- есть отдельные таблицы для сообщений, действий и transition log;
- есть индексы по `session_id`, `chat_id`, `state`, `created_at`;
- foreign keys на `session_id`.

Проблемы:

- `chat_id` не уникален, а `GetSessionByChatID` берет последнюю session (`services/decision-engine/internal/infrastructure/repository/postgres/queries/sessions.sql:6`);
- нет DB-level enum/check для `state` и `intent`;
- `actions_log` и `transitions_log` в текущем runtime фактически не пишутся;
- `session.version` есть, но обычный update state не инкрементит version (`services/decision-engine/internal/domain/session/service.go:72`);
- `summary` в модели есть, но generation/update summary в runtime не найден;
- `Metadata` в domain session не персистится (`services/decision-engine/internal/infrastructure/repository/postgres/helpers.go:16`).

### 6.2 Риск смешивания пользователей

Website backend жестко ставит `chatID := 1` на каждое WebSocket-соединение (`services/transport-adapters/website/backend/internal/websocket/handler.go:56`). Decision-engine тоже default-ит `chat_id=1`, если поле не передано.

Итог: разные браузеры будут использовать одну session/history/state. Это одновременно privacy, correctness и UX defect.

## 7. LLM И Prompt Design

### 7.1 Domain schema

LLM получает допустимые:

- intents из `services/decision-engine/internal/domain/intent/intent.go:70`;
- states из `services/decision-engine/internal/domain/state/state.go:40`;
- actions из `services/decision-engine/internal/domain/action/action.go:38`.

Это хорошая идея: модель ограничивается словарем системы.

### 7.2 Prompt

Prompt (`services/llm/app/prompts/decide_prompt.txt`) просит:

1. определить intent;
2. определить next state;
3. выбрать actions;
4. вернуть только JSON.

Риски:

- пользовательский текст вставляется в prompt напрямую как `role: text` (`services/llm/app/services/prompt_builder.py:61`);
- prompt-injection не изолирован;
- нет строгого JSON mode для Ollama;
- GigaChat JSON mode опционален, но не общий обязательный контракт;
- response retry есть, но нет интеграционного теста на весь retry/validation loop.

## 8. Интеграции

### 8.1 Ollama

Default provider: `ollama`, model `qwen2.5:7b` (`services/llm/app/core/config.py:21`). Клиент использует async `ollama.AsyncClient` и timeout.

### 8.2 GigaChat

GigaChat client реализует OAuth token refresh и chat completions.

Критичные риски:

- `verify=False` для API client и OAuth (`services/llm/app/services/llm/gigachat_client.py:69`, `:116`);
- debug logs могут содержать prompt и полный response (`services/llm/app/services/llm/gigachat_client.py:205`, `:234`);
- `temperature=0.7` и `max_tokens=2000` захардкожены (`services/llm/app/services/llm/gigachat_client.py:185`).

### 8.3 Lemmatizer

Lemmatizer service работает как отдельный FastAPI `/api/lemmatize` на pymorphy3. Однако:

- decision-engine local config указывает `http://localhost:8000`;
- lemmatizer-service default/Docker слушает `8080`;
- `decision-engine Run()` не загружает lemmatizer config и не создает normalization/classifier pipeline.

Вывод: lemmatizer-service сейчас не участвует в основном runtime.

## 9. Frontend И UX

### 9.1 Website UI

Static frontend:

- показывает chat window;
- connection status;
- input disabled до WebSocket open;
- отправляет текст;
- показывает typing indicator;
- option buttons отправляют текст опции как новое сообщение.

Плюсы:

- простой и понятный UI;
- user text экранируется через `escapeHtml`;
- есть reconnect attempts.

Минусы/риски:

- весь UI на английском, при том что бот и домен русскоязычные;
- `ws://` hardcoded, HTTPS даст mixed content (`services/transport-adapters/website/frontend/assets/js/websocket.js:97`);
- option buttons строятся через inline `onclick`, HTML escaping не является JS string escaping (`services/transport-adapters/website/frontend/assets/js/chat.js:124`);
- все option buttons на странице disable после выбора одной опции, включая старые и новые (`services/transport-adapters/website/frontend/assets/js/chat.js:159`);
- нет client/session identity;
- ошибки decision-engine с `success:false` backend может отправить как normal response, потому `Success/Error` игнорируются в `sendResponse`.

### 9.2 Console adapter

Console adapter полезен для ручного тестирования:

- можно менять `chat_id` командой `chat <id>`;
- печатает text/state/options;
- использует прямой `/decide`.

Минус: `raise_for_status()` теряет structured error body decision-engine.

## 10. Операционное Качество

### 10.1 Docker/Make

Есть Dockerfile/Makefile для отдельных сервисов, но нет общей orchestration.

Проблемы:

- `services/llm/Dockerfile` копирует `poetry.lock`, но файла нет, а `.gitignore` его игнорирует;
- LLM Dockerfile запускается root-пользователем;
- `poetry`/`poetry-core` ставятся без версии;
- website runtime image использует `alpine:latest`;
- LLM Makefile ссылается на отсутствующий `.env.local`;
- decision-engine Dockerfile ставит `APP_ENV=dev`, но в repo есть `config.local.yaml` и `config.prod.yaml`, не `config.dev.yaml`;
- decision-engine Dockerfile exposes `3000`, а config default/local HTTP address `:8080`.

### 10.2 Observability

Есть:

- health endpoints;
- structured logger;
- LLM circuit breaker;
- middleware request id/logging/recovery/timeout/body limit в decision-engine.

Не найдено:

- metrics;
- tracing;
- readiness checks зависимостей;
- dashboard/log redaction policy;
- audit persistence for action/transition runtime.

### 10.3 Тесты

Фактически найдено:

- Go tests только для rule-based classifier;
- Python tests для lemmatizer API/basic behavior;
- Python unit tests для LLM DomainService/PromptBuilder.

Главный runtime `decision-engine -> LLM -> actions -> response` тестами почти не закрыт.

Проверки, выполненные во время аудита:

- `cd services/decision-engine && go test ./...` - passed; большинство пакетов без tests, `rule_based` ok.
- `cd services/transport-adapters/website/backend && go test ./...` - passed; все пакеты без tests.
- `cd services/lemmatizer-service && python3 -m pytest -q` - не запустилось из-за отсутствующего `fastapi` в текущем Python окружении.
- `cd services/llm && python3 -m pytest -q` - не запустилось из-за отсутствующих `structlog`/`pydantic` в текущем Python окружении.

## 11. Security Review

### Critical/High

1. WebSocket Origin открыт для всех (`services/transport-adapters/website/backend/internal/websocket/handler.go:20`).
2. Все website users смешиваются в `chat_id=1`.
3. DOM XSS риск через inline option button handler (`services/transport-adapters/website/frontend/assets/js/chat.js:124`).
4. GigaChat TLS verification disabled (`services/llm/app/services/llm/gigachat_client.py:69`, `:116`).
5. Internal errors возвращаются клиентам (`decision-engine decide.go:81`, `llm decide.py:74`).

### Medium

1. Логи могут содержать user text, prompt, full LLM response, upstream body.
2. Нет authentication/authorization для browser/API flows.
3. Нет rate limiting.
4. Prompt injection никак специально не ограничен.
5. Secrets/env handling не стандартизирован на уровне всего repo.

## 12. Главные Архитектурные Риски

1. **Две модели принятия решений.** LLM-first путь работает сейчас, но рядом лежат stale rules/transitions. Нужно выбрать один source of truth.
2. **Advertised actions != registered actions.** LLM может вернуть `escalate_to_operator`/`reset_conversation`, но runtime их не обработает корректно.
3. **Session identity сломана для web.** `chat_id=1` делает систему некорректной при любом количестве пользователей больше одного.
4. **Mock actions маскируют отсутствие бизнес-интеграций.** Ответы выглядят production-like, но данные фейковые.
5. **Контракты не зафиксированы тестами.** Go/Python/WS contracts легко разъедутся.
6. **Audit tables не используются.** БД обещает трассируемость, runtime ее не создает.
7. **Response templating отсутствует.** Action data не попадает в user-facing шаблоны.
8. **Запуск всей системы не воспроизводим одной командой.**

## 13. Рекомендации

### P0 - перед любым production-like использованием

1. Ввести реальную session identity в website:
   - session cookie или generated browser client id;
   - передавать `chat_id`/session id в decision-engine;
   - убрать default `chat_id=1` как production behavior.
2. Закрыть security gaps:
   - WebSocket Origin allowlist;
   - убрать inline `onclick`;
   - выбрать `ws/wss` по `window.location.protocol`;
   - убрать `verify=False` или документировать и ограничить dev-only;
   - не возвращать raw internal errors наружу.
3. Синхронизировать `/config_llm` и runtime registry:
   - либо не отдавать actions, которых нет в processor;
   - либо зарегистрировать и реализовать `escalate_to_operator`, `reset_conversation`.
4. Сделать strict Go validation LLM response:
   - required `intent/state/actions`;
   - membership в domain schema;
   - reject unknown state/action before DB update.
5. Добавить integration tests:
   - fake LLM server -> decision-engine `/decide`;
   - LLM returns unknown action/state;
   - web adapter handles `success:false`;
   - option escaping/XSS regression.

### P1 - стабилизация архитектуры

1. Принять решение по rules/transitions:
   - удалить/архивировать stale path;
   - или сделать deterministic fallback и покрыть тестами.
2. Реализовать response templating:
   - action result -> response placeholders;
   - tests на `booking_found`, `payment_found`, etc.
3. Персистить bot messages, actions_log, transitions_log.
4. Подключить или удалить lemmatizer-service:
   - если нужен fallback classifier, подключить config/wiring и исправить порт;
   - если LLM-only, убрать неиспользуемый сервис из runtime expectations.
5. Добавить root `docker-compose.yml`:
   - Postgres;
   - decision-engine;
   - LLM service;
   - lemmatizer if needed;
   - website adapter;
   - documented env.

### P2 - качество продукта

1. Привести UI к русскоязычному домену.
2. Добавить operator workflow вместо TODO.
3. Добавить real integrations for booking/payment/account.
4. Добавить readiness checks, metrics и log redaction.
5. Убрать tracked `__pycache__` из репозитория и добавить корректный `.gitignore`.
6. Зафиксировать dependency lock strategy.

## 14. Итоговая Оценка

Текущая система хорошо показывает направление: модульный chatbot backend с LLM-классификацией, stateful sessions и несколькими transport adapters. Как учебный/MVP проект она уже полезна.

Для production readiness нужно сначала закрыть идентичность пользователя, разрыв между LLM domain schema и зарегистрированными actions, security issues и воспроизводимый запуск. После этого главным техническим долгом останется выбор единственной архитектуры принятия решений и замена mock business actions на реальные read-only интеграции.
