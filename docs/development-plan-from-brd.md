# План адаптации chat-bot под BRD

Дата: 2026-05-10  
Репозиторий: `/Users/danila/work/my/chat-bot`  
Главный источник требований: BRD из текущей задачи.  
Текущие источники фактов: `AUDIT.md`, актуальный код, конфиги, миграции, контракты и найденные тесты.

Принятое уточнение по реализации: совместимость со старой БД и старым runtime не требуется. Разрешено очищать БД, переписывать сервисы и удалять legacy-модули, если это быстрее и логичнее приводит к единому рабочему продукту по BRD.

## 0. Как читать документ

В документе разделены четыре типа утверждений:

- **Требование BRD** - целевое поведение, даже если в коде его еще нет.
- **Доказанный факт** - подтвержден текущим кодом, конфигом, миграцией, тестом или `AUDIT.md`.
- **Вывод** - инженерная интерпретация разрыва между BRD и кодом.
- **Открытый вопрос** - требует решения владельца продукта/проекта, но не блокирует весь план.

Статусы требований:

- **Есть** - capability реализована в целевом смысле BRD и подтверждена кодом.
- **Частично** - есть часть механики, но не весь BRD-сценарий.
- **Отсутствует** - нет рабочей реализации.
- **Конфликтует** - текущая реализация противоречит BRD или создает ложный source of truth.

## 1. Executive Summary

**Что проект делает сейчас.**  
Текущий runtime описан в аудите как `console/web UI -> decision-engine /decide -> Python LLM /llm/decide -> локальные actions -> responses.json -> клиент` (`AUDIT.md:10`). Это подтверждается кодом: `MessageWorker.HandleMessage` грузит/создает сессию, сохраняет пользовательское сообщение, берет историю, вызывает LLM, выполняет actions, выбирает response key и обновляет состояние (`services/decision-engine/internal/app/worker/message_worker.go:46`, `services/decision-engine/internal/app/worker/message_worker.go:89`, `services/decision-engine/internal/app/worker/message_worker.go:125`, `services/decision-engine/internal/app/worker/message_worker.go:136`). Веб-адаптер принимает WebSocket-сообщение и проксирует его в `/decide` (`services/transport-adapters/website/backend/internal/websocket/handler.go:85`, `services/transport-adapters/website/backend/internal/websocket/handler.go:119`).

**Что должен делать по BRD.**  
BRD требует демонстрационную интеллектуальную систему поддержки для платформы бьюти-коворкинга: веб-чат клиента, интерфейс оператора, Go backend, отдельный Python NLP/embedding service, PostgreSQL + pgvector, intent-centric база знаний, semantic intent matching по embeddings, заранее заданные ответы, контекст диалога, fallback, операторская эскалация, сохранение истории и состояния после рестарта. BRD явно ограничивает проект: система не использует генеративные LLM, не выполняет платежи, не изменяет пользовательские данные, не управляет бронированием и не заменяет CRM.

**Главный разрыв.**  
Архитектурное ядро сейчас LLM-first, а целевое ядро по BRD должно быть deterministic semantic intent matching без генеративных LLM. В коде уже есть куски, похожие на будущую основу: canned responses (`services/decision-engine/configs/responses.json:1`), доменные intents/states/actions (`services/decision-engine/internal/domain/intent/intent.go:70`, `services/decision-engine/internal/domain/state/state.go:40`, `services/decision-engine/internal/domain/action/action.go:38`), rule-based classifier (`services/decision-engine/internal/infrastructure/nlp/rule_based/classifier.go:113`), transition engine (`services/decision-engine/internal/app/transition/engine.go:34`), Postgres sessions/messages (`services/decision-engine/migrations/20260216002310_init_tables.sql:4`). Но основной runtime их не использует как decision pipeline: `Run()` инициализирует LLM client, presenter, processor и зарегистрированные mock actions, но не подключает rule-based/semantic classifier, transition engine или lemmatizer pipeline (`services/decision-engine/internal/app/run.go:127`, `services/decision-engine/internal/app/run.go:136`, `services/decision-engine/internal/app/run.go:139`).

**Практический путь.**  
Нужно не “улучшить LLM”, а заменить decision core: `normalize/preprocess -> embedding -> semantic top-k intent match -> threshold/ambiguity/fallback policy -> limited mode FSM -> canned response renderer -> action/operator routing -> persistence/audit`. Python-сервис должен стать NLP/embedding-сервисом, а текущий `services/llm` либо удаляется из runtime, либо остается только как dev/offline tool без участия в обработке пользовательского сообщения.

**Правило по внешним данным и мокам.**  
Все сервисы, которые принадлежат нашему продукту, должны быть реализованы как законченные production-like компоненты: decision-engine, NLP service, web chat, operator UI, persistence, contracts, tests, observability. Все интеграции с внешними бизнес-системами должны быть описаны полноценными контрактами и вызываться через adapter/provider слой, но в рамках учебного проекта эти внешние системы реализуются как mock services / mock providers с детерминированными demo-данными. То есть бизнес-операции и use-cases проектируются полностью, но фактические ответы приходят из моков.

### 1.1 Definition of Done полного сервиса

Проект считается законченным не после “рефакторинга архитектуры”, а когда есть полностью рабочий демонстрационный support-сервис:

- сервисы поднимаются одной командой через `docker compose up`;
- миграции применяются на пустую БД без ручных действий;
- seed data загружается автоматически или одной documented-командой;
- клиентский веб-чат открывается в браузере и создает отдельную сессию пользователя;
- два разных пользователя не видят историю и состояние друг друга;
- свободный текст классифицируется через semantic intent matching без генеративного LLM;
- pgvector реально используется для поиска intent examples / knowledge chunks;
- все основные категории BRD имеют рабочие сценарии: запись/бронирование, сайт, оплата, рабочие места, услуги/правила, жалобы;
- canned responses возвращаются без сырых `{placeholder}`;
- quick replies работают как typed actions, а не как произвольный display text;
- low confidence первый раз ведет к уточнению, повторный low confidence ведет к предложению/очереди оператора;
- пользователь может вручную запросить оператора;
- оператор видит очередь, принимает диалог, читает историю и отвечает в той же сессии;
- пока оператор подключен, бот не отвечает автоматически;
- после закрытия handoff сессия фиксирует финальное состояние;
- user/bot/operator messages, decisions, actions and transitions пишутся в БД;
- после рестарта сервисов история и состояние восстанавливаются;
- health/readiness endpoints отражают реальную готовность DB/NLP/vector/seed data;
- security-блокеры закрыты: no `chat_id=1`, Origin allowlist, WSS-ready URL, no inline JS, no raw internal errors;
- есть полная E2E-матрица по бизнес-флоу и она проходит;
- есть UML/component diagram, ERD, deployment diagram, sequence diagrams для message flow и operator handoff;
- root README объясняет запуск, тесты, архитектуру и ограничения demo-системы.

### 1.2 Сквозные сценарии, которые должен пройти продукт

Минимальный завершенный продукт должен демонстрировать такие end-to-end flows:

1. **FAQ / услуги и правила:** пользователь спрашивает “какие есть услуги и цены”, бот находит intent, отвечает canned response, показывает quick replies.
2. **Запись и бронирование:** пользователь спрашивает статус записи, вводит demo-номер, бот делает read-only lookup и подставляет данные записи в ответ.
3. **Рабочие места:** пользователь спрашивает про аренду места или статус брони рабочего места, бот отвечает по базе знаний или demo lookup.
4. **Оплата:** пользователь пишет “деньги списались, услуга не активировалась”, бот определяет payment problem, дает инструкцию и предлагает оператора.
5. **Проблемы сайта/входа:** пользователь пишет про сайт/логин/код, бот дает заранее заданную инструкцию.
6. **Жалоба:** пользователь описывает жалобу, бот собирает контекст и ставит обращение в operator queue.
7. **Непонятный запрос:** первый unknown intent дает уточнение, второй unknown intent создает handoff или предлагает оператора.
8. **Ручной оператор:** пользователь нажимает/пишет “оператор”, сессия уходит в `waiting_operator`.
9. **Операторский ответ:** оператор принимает диалог, отвечает, пользователь видит сообщение в том же чате.
10. **Изоляция пользователей:** два браузера проходят разные сценарии, состояния и истории не пересекаются.
11. **Перезапуск:** после restart decision-engine/website пользователь продолжает с сохраненной историей.
12. **Security regression:** malicious quick reply / HTML text не исполняется, чужой Origin rejected, public errors без internal detail.

### 1.3 Целевая структура репозитория

Целевая структура должна быть понятной по назначению, без конкурирующих архитектурных путей:

```text
.
├── README.md
├── docker-compose.yml
├── .env.example
├── docs/
│   ├── development-plan-from-brd.md
│   ├── architecture.md
│   └── diagrams/
│       ├── component.md
│       ├── erd.md
│       ├── deployment.md
│       ├── sequence-user-message.md
│       └── sequence-operator-handoff.md
├── seeds/
│   ├── intents.json
│   ├── knowledge-base.json
│   ├── demo-bookings.json
│   ├── demo-workspace-bookings.json
│   ├── demo-payments.json
│   ├── demo-users.json
│   └── demo-operators.json
├── services/
│   ├── decision-engine/
│   │   ├── cmd/
│   │   ├── internal/app/decision/
│   │   ├── internal/app/operator/
│   │   ├── internal/app/actions/
│   │   ├── internal/transport/http/
│   │   ├── internal/infrastructure/repository/postgres/
│   │   ├── migrations/
│   │   └── tests/
│   ├── nlp-service/
│   │   ├── app/
│   │   └── tests/
│   └── transport-adapters/
│       ├── website/
│       └── console/
└── tests/
    └── e2e/
```

`services/llm` в целевом продукте отсутствует как runtime-сервис. Если его нужно сохранить для истории, он должен быть вынесен в `legacy/` или `tools/legacy-llm/` и не подключаться compose/runtime-тестами.

## 2. Карта требований BRD

| ID | Capability / use-case | Бизнес-смысл | Пользовательский сценарий | Текущий статус | Доказательства | Что сделать |
|---|---|---|---|---|---|---|
| BRD-C01 | Веб-чат клиента | Пользователь получает поддержку без внешних каналов | Клиент открывает сайт, пишет вопрос, получает ответ/кнопки | Частично | HTML содержит экран чата и input (`services/transport-adapters/website/frontend/index.html:19`, `services/transport-adapters/website/frontend/index.html:33`); WS payload `{type,text}` (`services/transport-adapters/website/frontend/assets/js/chat.js:89`) | Ввести session identity, WSS-aware URL, безопасные quick replies, русскоязычный UX, контрактные тесты |
| BRD-C02 | Свободный текстовый ввод | Пользователь не обязан выбирать только меню | Пользователь пишет “не прошла оплата” | Частично | `sendMessage` отправляет произвольный `text` (`services/transport-adapters/website/frontend/assets/js/chat.js:80`) | Подключить semantic matcher и fallback policy вместо LLM-first |
| BRD-C03 | Быстрые ответы | Снизить когнитивную нагрузку и направить сценарий | Бот показывает кнопки “Оплата”, “Оператор” | Частично | `options` рендерятся как кнопки (`services/transport-adapters/website/frontend/assets/js/chat.js:116`) | Заменить строки на typed `{id,label,intent,payload}`; убрать inline `onclick` |
| BRD-C04 | Несколько пользователей одновременно | Нельзя смешивать историю и состояние клиентов | Два браузера ведут разные диалоги | Конфликтует | WebSocket hardcode `chatID := 1` (`services/transport-adapters/website/backend/internal/websocket/handler.go:56`); `/decide` default `chat_id=1` (`services/decision-engine/internal/transport/http/handler/decide.go:64`) | Cookie/client id/session id, unique active session per channel user, regression tests на изоляцию |
| BRD-C05 | Сохранение истории сообщений | Оператор и бот видят контекст, состояние переживает рестарт | Пользователь возвращается и видит продолжение | Частично | Таблица `messages` есть (`services/decision-engine/migrations/20260216002310_init_tables.sql:23`); user message сохраняется (`services/decision-engine/internal/app/worker/message_worker.go:57`) | Сохранять bot/operator messages, message ids, detected intent/confidence, transaction boundary |
| BRD-C06 | Сохранение состояния диалога | Система знает режим и активную тему | После рестарта бот продолжает flow | Частично | `sessions.state`, `summary`, `status` есть (`services/decision-engine/migrations/20260216002310_init_tables.sql:4`) | Добавить active_topic, last_intent, operator_status, fallback_count, metadata persistence |
| BRD-C07 | Ограниченная FSM-модель режимов | Не строить полную FSM, но управлять режимами | standard -> waiting_operator -> operator_connected | Частично/конфликтует | Есть много domain states (`services/decision-engine/internal/domain/state/state.go:5`), transition engine есть (`services/decision-engine/internal/app/transition/engine.go:34`), но не wiring в runtime (`services/decision-engine/internal/app/run.go:73`) | Сузить FSM до режимов BRD и хранить topic отдельно; старые transitions валидировать/переписать |
| BRD-C08 | Semantic intent matching | Понимать свободные формулировки без LLM | “деньги списались, брони нет” -> payment_not_activated | Отсутствует | embeddings path только TODO (`services/decision-engine/internal/infrastructure/nlp/classifier.go:43`); pgvector отсутствует в миграции (`services/decision-engine/migrations/20260216002310_init_tables.sql:1`) | Добавить intent catalog, embeddings, pgvector tables/indexes, top-k/threshold/ambiguity |
| BRD-C09 | Preprocessing / lemmatization | Нормализовать русскоязычный ввод | “записался/запись” сравниваются устойчиво | Частично | Normalization pipeline есть (`services/decision-engine/internal/infrastructure/nlp/normalization/normalization.go:24`); Python lemmatizer endpoint есть (`services/lemmatizer-service/app/api/router.py:8`) | Подключить в decision pipeline или перенести в новый NLP service; исправить порт/config |
| BRD-C10 | Intent-centric база знаний | Ответы и примеры управляются как доменные интенты | Администратор добавляет примеры к intent | Отсутствует/частично | Canned responses есть (`services/decision-engine/configs/responses.json:1`), но нет таблиц KB/intent embeddings | Добавить `intents`, `intent_examples`, `knowledge_articles/chunks`, `quick_replies`, embeddings |
| BRD-C11 | Заранее заданные ответы | Демонстрация без генеративных ответов | Бот возвращает approved response | Частично | Presenter возвращает шаблон из JSON (`services/decision-engine/internal/app/presenter/presenter.go:39`) | Сделать renderer с placeholder interpolation и validation coverage |
| BRD-C12 | Fallback при низкой уверенности | Не давать неправильные ответы | Первый low confidence -> уточнение, второй -> оператор | Частично | `clarify_request` и generic errors есть (`services/decision-engine/configs/responses.json:409`, `services/decision-engine/configs/responses.json:428`), но confidence модели нет | Хранить fallback_count, last_candidates, threshold policy, escalation trigger |
| BRD-C13 | Резкая смена темы сбрасывает flow | Не тащить старый контекст в новую тему | Пользователь из оплаты переходит в “рабочие места” | Отсутствует | Сейчас LLM выбирает state; explicit topic switch detector нет (`services/decision-engine/internal/app/worker/message_worker.go:89`) | Добавить active_topic, topic similarity delta, reset rules |
| BRD-C14 | Ручной запрос оператора | Пользователь может попросить человека | “Позовите оператора” | Отсутствует | Intent/action объявлены (`services/decision-engine/internal/domain/intent/intent.go:16`, `services/decision-engine/internal/domain/action/action.go:34`), action TODO и не registered (`services/decision-engine/internal/app/actions/escalate_to_operator.go:14`, `services/decision-engine/internal/app/run.go:139`) | Реализовать operator queue/action/API/UI |
| BRD-C15 | Автоматическая эскалация | После repeated fallback бот передает оператору | Бот не понял дважды и предлагает оператора | Отсутствует | Нет fallback_count и operator queue; escalation только TODO (`services/decision-engine/internal/app/actions/escalate_to_operator.go:14`) | Добавить policy, queue table, handoff events |
| BRD-C16 | Оператор продолжает в той же сессии | Нет потери контекста при handoff | Оператор читает историю и отвечает в chat | Отсутствует | Operator UI/routes отсутствуют; website server только `/ws`, `/health`, static (`services/transport-adapters/website/backend/internal/websocket/server.go:24`) | Operator API, WS events, message sender_type=operator, queue lifecycle |
| BRD-C17 | Mock business APIs / no real mutations | Демонстрация домена без платежей/CRM | Бот показывает статус из тестовых данных | Частично/конфликтует | Actions генерируют MOCK (`services/decision-engine/internal/app/actions/find_booking.go:12`, `services/decision-engine/internal/app/actions/find_payment.go:12`) | Оформить read-only mock providers как явные contracts/fixtures, убрать иллюзию реальной интеграции |
| BRD-C18 | Docker deployment | Запуск демонстрации одной командой | `docker compose up` поднимает сервисы | Частично | Dockerfile есть, общего compose нет (`AUDIT.md:39`); LLM Dockerfile копирует `poetry.lock` (`services/llm/Dockerfile:12`) | Root compose, env examples, healthchecks, migrations, seed KB |
| BRD-C19 | Защищенный транспорт | Не ломать HTTPS/WSS и Origin | В прод-like стенде чат через WSS | Конфликтует | Frontend hardcodes `ws://` (`services/transport-adapters/website/frontend/assets/js/websocket.js:97`); Origin открыт (`services/transport-adapters/website/backend/internal/websocket/handler.go:16`) | WSS-aware URL, Origin allowlist, reverse proxy/deployment diagram |
| BRD-C20 | Базовая защита от инъекций | Не исполнять пользовательский ввод и не раскрывать внутренности | Пользователь отправляет HTML/кавычки/служебный текст | Частично/конфликтует | user text escape есть (`services/transport-adapters/website/frontend/assets/js/chat.js:190`), но inline onclick риск (`services/transport-adapters/website/frontend/assets/js/chat.js:124`) | DOM APIs, no inline handlers, SQL params уже через sqlc сохранить, raw error masking |
| BRD-C21 | Время ответа < 3 сек | Демонстрация должна быть интерактивной | Пользователь получает ответ быстро | Не доказано/конфликтует | Website timeout 15s (`services/transport-adapters/website/backend/internal/websocket/handler.go:119`); LLM timeout 30s (`services/decision-engine/configs/config.local.yaml:33`) | Latency budget, no runtime LLM, cached embeddings, perf tests |
| BRD-C22 | UML/ERD/deployment/sequence diagrams | Итоговый учебный артефакт должен быть проверяемым | Команда видит архитектуру и flows | Отсутствует | Только `dialogous.md` найден как отдельный doc, root docs отсутствуют (`AUDIT.md:39`) | Добавить `docs/diagrams/*.md` или PlantUML/Mermaid: component, ERD, deployment, request/hand-off sequences |

## 3. Целевая архитектура

### 3.1 Решение по LLM-first

**Требование BRD:** проект демонстрирует интеллектуальную поддержку без генеративных LLM.  
**Доказанный факт:** `services/llm` сейчас именно генеративный LLM service: FastAPI title “LLM Service” (`services/llm/app/main.py:21`), provider Ollama/GigaChat (`services/llm/app/core/config.py:6`), prompt builder вставляет историю в prompt (`services/llm/app/services/prompt_builder.py:43`), `DecideService` вызывает `llm_client.generate` (`services/llm/app/services/decide_service.py:64`).  
**Вывод:** LLM-first подход нужно удалить из production/demo runtime. Иначе проект не соответствует BRD даже при хороших тестах.

Целевой decision flow:

1. `website/console/operator adapter` принимает сообщение и идентичность.
2. `decision-engine` открывает транзакцию, грузит активную session, сохраняет inbound message.
3. `TextPreprocessor` нормализует текст: lowercase/tokenize/stopwords/lemmatize.
4. `EmbeddingClient` вызывает Python NLP service `/embed` или получает cached vector.
5. `SemanticIntentMatcher` ищет nearest examples/intents в Postgres pgvector.
6. `DecisionPolicy` применяет threshold, ambiguity delta, topic-switch detection, fallback_count.
7. `ModeStateMachine` управляет только режимами: `standard`, `waiting_operator`, `operator_connected`, `closed`.
8. `ActionRouter` выполняет только read-only lookup/operator queue actions.
9. `ResponseRenderer` выбирает заранее заданный response и подставляет action data.
10. `PersistenceUnit` пишет bot response, action log, transition log, session context/version.
11. Adapter отправляет клиенту typed WS/HTTP response.

### 3.2 Компоненты оставить

- `services/decision-engine` как основной Go backend и decision owner. Текущий bootstrap в `Run()` уже удобно собирает config, DB, repos, presenter, processor и HTTP (`services/decision-engine/internal/app/run.go:73`).
- `transport-adapters/website` как клиентский web chat, но с новой session identity и безопасным WS contract.
- `transport-adapters/console` как manual/dev adapter для `/decide`, но без статуса production UI.
- `responses.json` как стартовый набор canned responses, после нормализации ключей и interpolation.
- `normalization` и `lemmatizer-service` как полезный слой preprocessing, если он будет включен в runtime.
- `sqlc`/Postgres repository approach.

### 3.3 Компоненты переделать

- `services/llm` переименовать/заменить на `services/nlp-service`: убрать `OllamaClient`, `GigaChatClient`, prompt files и `/llm/decide`; добавить `/preprocess`, `/embed`, `/health`, возможно `/semantic-match` только если поиск остается в Python. Целевой контракт не должен возвращать `state/actions`.
- `decision-engine/internal/app/worker/message_worker.go`: заменить LLM call (`services/decision-engine/internal/app/worker/message_worker.go:89`) на `DecisionService.Decide`.
- `domain/state`: отделить режимы BRD от topic/intent. Сейчас states смешивают режимы (`waiting_clarification`, `escalated_to_operator`) и темы (`booking`, `workspace`, `payment`) (`services/decision-engine/internal/domain/state/state.go:5`).
- `configs/transitions.json`: переписать или удалить как stale source. Сейчас он содержит action name `escalate_operator`, а домен объявляет `escalate_to_operator` (`services/decision-engine/configs/transitions.json:90`, `services/decision-engine/internal/domain/action/action.go:34`).
- `ResponseSelector`: заменить state-only маппинг на `intent/action-result/fallback/mode` selector. Сейчас он выбирает response по action result status или state (`services/decision-engine/internal/app/processor/response_selector.go:29`).

### 3.4 Компоненты удалить или вывести из runtime

- Generative LLM runtime path: Go `llm.Client.Decide` (`services/decision-engine/internal/infrastructure/llm/client.go:78`) и Python prompt-based `/llm/decide` (`services/llm/app/api/routes/decide.py:15`) как production/demo decision path.
- `configs/rules.json`, если он останется старым keyword-only catalog. Его можно сохранить только как deterministic fallback layer после приведения под BRD.
- Старые `/llm/intent`, `/llm/transition`, `/llm/summary`, `/llm/generate_response` методы Go client: они есть в клиенте (`services/decision-engine/internal/infrastructure/llm/client.go:68`), но не являются BRD-контрактом.

### 3.5 Нужен ли rule-based fallback

Да, но не как основной интеллект и не как stale `rules.json`. Целевая роль:

- exact command intents: `request_operator`, `return_to_menu`, `reset_conversation`, `goodbye`;
- safety fallback при недоступном embedding service;
- deterministic smoke tests.

Основной путь для FAQ/use-cases должен быть semantic embeddings + thresholds. Rule-based classifier сейчас keyword/phrase scoring (`services/decision-engine/internal/infrastructure/nlp/rule_based/classifier.go:113`) и имеет TODO на embeddings при unknown (`services/decision-engine/internal/infrastructure/nlp/classifier.go:43`), поэтому его нужно либо вплести как fallback, либо удалить из runtime expectations.

### 3.6 Нужен ли lemmatizer-service

BRD прямо упоминает preprocessing и lemmatization для Python NLP service, поэтому lemmatizer path лучше сохранить, но встроить в новый `nlp-service`. Сейчас сервис предоставляет `/api/lemmatize` (`services/lemmatizer-service/app/api/router.py:8`), а Go config указывает lemmatizer base URL (`services/decision-engine/configs/config.local.yaml:25`), но `Run()` его не загружает. Целевое решение: либо Go вызывает Python `/preprocess`/`/embed`, либо Go нормализует сам, а Python только строит embeddings. Два параллельных preprocessing source of truth оставлять нельзя.

## 4. Что нужно выпилить

| Кандидат | Почему конфликтует/не нужен | Риски удаления | Что заменяет | Проверки |
|---|---|---|---|---|
| `services/llm/app/services/llm/*`, prompt files, `/llm/decide` runtime | BRD запрещает генеративные LLM; текущий flow зависит от `llm_client.generate` (`services/llm/app/services/decide_service.py:64`) | Потеря текущего единственного decision path | `DecisionService` в Go + Python NLP embeddings | Contract tests `/decide`; no calls to `/llm/decide`; mocked NLP tests |
| Go `internal/infrastructure/llm/client.go` runtime use | Делает HTTP call в `/llm/decide` (`services/decision-engine/internal/infrastructure/llm/client.go:78`) | Нужно переписать worker and tests | `nlp.Client` для `/embed` или `EmbeddingProvider` | `rg "llmClient.Decide"` пуст в production path |
| Stale `configs/transitions.json` | Не подключен в `Run()` и содержит missing/stale keys/actions (`services/decision-engine/configs/transitions.json:90`) | Если тихо удалить, потеряется задумка FSM | Новый mode FSM config или Go constants с validation | Startup validation: all state/action/response keys exist |
| Stale `configs/rules.json` в текущем виде | Keyword-only, не semantic, не full BRD catalog (`services/decision-engine/configs/rules.json:1`) | Потеря нескольких smoke intents | Новый `intent_examples` seed + command rules | Classifier tests на exact commands and semantic fallback |
| Неиспользуемый rule-based NLP path как основной path | Не wiring в `Run()` (`services/decision-engine/internal/app/run.go:73`) | Нужна замена unknown handling | Semantic matcher | Integration test low-confidence flow |
| Неиспользуемый lemmatizer path как отдельный “висящий” сервис | Сервис есть, config есть, runtime не подключен (`services/decision-engine/configs/config.local.yaml:25`, `services/decision-engine/internal/app/run.go:73`) | Если удалить полностью, потеряем BRD lemmatization | Новый unified `nlp-service` | `/health`, `/preprocess`, `/embed` contract tests |
| Mock lookup actions как “реальные интеграции” | Actions генерируют data через hash, а не читают источник (`services/decision-engine/internal/app/actions/find_booking.go:31`) | Demo сценарии перестанут показывать статусы | Explicit mock read-only providers/fixtures или DB seed tables | Tests на deterministic fixture lookups |
| `EscalateToOperator` TODO implementation | Action только `fmt.Println` (`services/decision-engine/internal/app/actions/escalate_to_operator.go:14`) | Без замены сломается операторский сценарий | Operator queue + handoff service | Queue lifecycle integration tests |
| Tracked `__pycache__` | Служебные бинарные файлы не должны быть source | Низкий | `.gitignore`, удалить из git | `find . -path "*__pycache__*"` clean |
| Пустые/неактуальные contract files | `contracts/websocket.json` пуст, реальный контракт в DTO (`AUDIT.md:44`) | Может сломать внешние ссылки, если кто-то смотрел файл | Versioned OpenAPI/JSON Schema docs | Contract tests validate schema vs DTO |
| Internal raw error exposure | `/decide` возвращает `%v` (`services/decision-engine/internal/transport/http/handler/decide.go:79`) | Debuggability снизится | Stable error codes + request id | Tests на отсутствие internal body |

## 5. Что нужно переделать

### 5.1 Session/user identity

**Факт:** website и `/decide` смешивают пользователей в `chat_id=1` (`services/transport-adapters/website/backend/internal/websocket/handler.go:56`, `services/decision-engine/internal/transport/http/handler/decide.go:64`). `LoadSession` создает случайный UUID пользователя и прямо содержит TODO про auth context (`services/decision-engine/internal/domain/session/service.go:20`).  
**План:** ввести `channel`, `external_user_id`, `session_id` и cookie/browser id. `/decide` должен требовать identity от adapter или выдавать session explicitly через `/sessions`. Default `chat_id=1` оставить только под dev CLI флагом.

### 5.2 Business actions

**Факт:** actions заявлены как read-only (`services/decision-engine/internal/domain/action/action.go:21`), но фактически mock generators (`services/decision-engine/internal/app/actions/find_booking.go:22`, `services/decision-engine/internal/app/actions/find_payment.go:22`).  
**План:** оформить `BusinessLookupProvider` с read-only contracts. Внутри нашего продукта provider layer должен быть полноценным: typed contracts, timeouts, errors, audit logs, tests. За provider layer в demo будут mock external services с мок-данными:

- `GetBookingStatus(identifier, user/context)`;
- `GetWorkspaceBookingStatus(identifier)`;
- `GetPaymentStatus(identifier)`;
- `GetAccountInfo(identifier)`;
- `GetServiceRules(category)`.

В учебной версии provider вызывает mock service или mock repository, но с тем же contract shape, который имел бы реальный внешний сервис. Responses/logs должны явно показывать `source=mock_external`, чтобы не создавать иллюзию реальной CRM/payment/booking integration.

### 5.3 Operator escalation

**Факт:** domain action есть, implementation TODO и не registered (`services/decision-engine/internal/domain/action/action.go:34`, `services/decision-engine/internal/app/actions/escalate_to_operator.go:14`, `services/decision-engine/internal/app/run.go:139`).  
**План:** добавить `OperatorHandoffService`:

- создать queue item;
- записать reason: `manual_request`, `low_confidence_repeated`, `complaint`, `business_error`;
- прикрепить context snapshot: last messages, active topic, last intent, action data;
- перевести mode `standard -> waiting_operator`;
- при назначении оператора `waiting_operator -> operator_connected`;
- пока operator connected, bot не отвечает, а user messages идут оператору.

### 5.4 Response templating

**Факт:** Presenter возвращает JSON message как есть (`services/decision-engine/internal/app/presenter/presenter.go:39`), а templates содержат `{service}`, `{date}`, `{status}` (`services/decision-engine/configs/responses.json:46`).  
**План:** добавить typed renderer:

- вход: `ResponseTemplate`, `ActionResult`, `SessionContext`;
- strict placeholder validation on startup;
- missing placeholder -> internal log + safe fallback;
- локализованные labels/status mappings;
- tests для `booking_found`, `payment_found`, `account_found`, escalation context.

### 5.5 LLM contract validation

Целевой runtime не должен зависеть от LLM. Если `services/llm` временно остается для сравнения/legacy, нужно:

- скрыть `/config_llm` как public contract или переименовать в `/domain/schema`;
- удалить advertised actions, которых нет в registry;
- Go-side validation сейчас слабая: `parseDecideResponse` молча принимает пустые строки (`services/decision-engine/internal/app/worker/message_worker.go:175`);
- запретить LLM path в production config через feature flag `DECISION_MODE=semantic`.

### 5.6 Prompt design

Так как BRD запрещает генеративные LLM, prompt design не должен быть частью runtime. Prompt files можно удалить или вынести в `tools/legacy-llm/` с пометкой “не используется в BRD runtime”. Если временно оставлять, user content надо изолировать, потому сейчас `_format_messages` напрямую вставляет `role: text` (`services/llm/app/services/prompt_builder.py:61`).

### 5.7 Adapters

- Website adapter: typed session handshake, `quick_reply` event, WSS URL, Origin allowlist, operator events.
- Console adapter: оставить как dev tool; команда `chat <id>` есть (`services/transport-adapters/console/main.py:77`), но нужно добавить explicit session id and structured error output.
- Decision-engine API: сделать stable versioned routes `/api/v1/...`.

### 5.8 Frontend UX

**Факт:** UI английский (`services/transport-adapters/website/frontend/index.html:2`, `services/transport-adapters/website/frontend/index.html:12`), домен и responses русские.  
**План:** русскоязычный чат клиента, quick replies как кнопки с data-id, статус “Ожидаем оператора”, history restore, видимое отделение сообщений бота/оператора. Не добавлять маркетинговую страницу; первый экран - рабочий чат.

### 5.9 Config/env

- Убрать несоответствие ports/env: decision-engine Docker `EXPOSE 3000`, config `:8080` (`services/decision-engine/Dockerfile:44`, `services/decision-engine/configs/config.local.yaml:41`).
- LLM Dockerfile копирует отсутствующий `poetry.lock` (`services/llm/Dockerfile:12`) - после удаления LLM path заменить на NLP service lock/requirements.
- Root `.env.example`, compose profiles `dev`, `demo`.
- Для website backend Origin allowlist должен задаваться через `server.allowed_origins` / `WS_ALLOWED_ORIGINS` как список доверенных browser origins, например `https://chat.example.test,http://localhost:8081`. Пустой allowlist недопустим.

### 5.9.1 Reverse proxy / TLS for website WS

- TLS должен завершаться на reverse proxy или ingress перед website backend.
- Браузер, открытый по `https://...`, должен строить WebSocket URL как `wss://<host>/ws`; страница на `http://...` использует `ws://<host>/ws` только для локальной/dev среды.
- Reverse proxy должен проксировать `GET /ws` на website backend без изменения `Host`/`Origin` и с upgrade-заголовками WebSocket.
- Unknown `Origin` должен отклоняться на backend уровне с HTTP 403 и без раскрытия внутренних деталей в теле ответа; в логах допустимы только origin/host/remote_addr для диагностики.

### 5.10 Logging/observability

- Ввести request/session/message ids в logs.
- Redaction policy: не логировать raw user text, prompt, full upstream bodies в prod. Сейчас website backend логирует `text` (`services/transport-adapters/website/backend/internal/websocket/handler.go:114`), browser console логирует payloads (`services/transport-adapters/website/frontend/assets/js/websocket.js:25`), GigaChat logs prompt/response (`services/llm/app/services/llm/gigachat_client.py:203`).
- Readiness endpoints должны проверять DB, migrations, NLP service, vector extension.

### 5.11 Security

См. раздел 8. Главное P0: session isolation, Origin, WSS/HTTPS story, no raw errors, no inline JS handlers, TLS verification, rate limits.

## 6. База данных и миграции

### 6.0 Greenfield-подход к БД

Поддерживать старую схему и старые данные не нужно. Правильный путь для этого проекта:

- удалить/заменить текущую init migration, если она мешает целевой модели;
- проектировать схему сразу под BRD, operator handoff, semantic search and audit trail;
- считать пустую БД нормальным стартовым состоянием;
- загружать demo data через seed files;
- не писать backfill-миграции для старых `sessions/chat_id=1`;
- не сохранять legacy `summary varchar(255)`, если нужна нормальная `session_context`;
- не добавлять transitional fields “на всякий случай”;
- держать одну финальную модель данных и одну финальную runtime-архитектуру.

Миграции нужны не для совместимости со старым кодом, а для воспроизводимого развертывания целевой demo-системы с нуля.

### 6.1 Что уже есть

Текущая миграция создает:

- `sessions`: `id`, `chat_id`, `user_id`, `state`, `summary`, `version`, `status`, timestamps (`services/decision-engine/migrations/20260216002310_init_tables.sql:4`).
- `messages`: `session_id`, `sender_type`, `text`, `intent`, `created_at`; sender type допускает `user`, `bot`, `operator` (`services/decision-engine/migrations/20260216002310_init_tables.sql:23`).
- `users`: `id`, `external_id`, timestamps (`services/decision-engine/migrations/20260216002310_init_tables.sql:40`).
- `transitions_log`: `session_id`, `from_state`, `to_state`, `created_at` (`services/decision-engine/migrations/20260216002310_init_tables.sql:48`).
- `actions_log`: `session_id`, `action_type`, request/response payloads, error (`services/decision-engine/migrations/20260216002310_init_tables.sql:59`).

Repos/queries есть для session/message/user/action/transition. Но action log repo создается, а processor его не вызывает (`services/decision-engine/internal/app/run.go:122`, `services/decision-engine/internal/app/processor/processor.go:64`). Transition repo существует, но не wiring в app (`services/decision-engine/internal/infrastructure/repository/postgres/transitionlog_repo.go:18`).

### 6.2 Чего не хватает под BRD

- `pgvector` extension and vector columns.
- `intents`: key, category, response_key, escalation flags, fallback policy, active.
- `intent_examples`: intent_id, text, normalized_text, embedding, locale, weight.
- `knowledge_articles` / `knowledge_chunks`: category, title, body, embedding, source, active.
- `quick_replies`: response_key/intent_id, label, action, payload JSONB, order.
- `session_context`: or new fields on `sessions`: active_topic, last_intent, fallback_count, operator_status, metadata JSONB.
- `operator_queue`: session_id, user_id, status, reason, priority, assigned_operator_id, created_at, accepted_at, closed_at.
- `operators`: demo operator accounts or user role table.
- `message_events` or `inbox_events`: idempotency by external event id.
- `decision_logs`: selected intent, confidence, top candidates, threshold result, fallback reason.

### 6.3 Поля и constraints

- `sessions.user_id` должен иметь FK на `users.id`.
- `sessions` нужна уникальность активной сессии по `(channel, external_user_id)` или `(channel, client_id, status='active')`.
- `sessions.version` должен инкрементиться при state/context update; сейчас обычный update не инкрементит (`services/decision-engine/internal/infrastructure/repository/postgres/queries/sessions.sql:16`).
- `messages` нужно расширить: `idempotency_key`, `detected_intent`, `confidence`, `metadata`, `created_by`.
- `actions_log` добавить `message_id`, `status`, `duration_ms`, `provider`, `redacted_payload`.
- `transitions_log` добавить `message_id`, `event`, `reason`, `actor_type`.
- Добавить `CREATE EXTENSION IF NOT EXISTS pgcrypto` для `gen_random_uuid()` и `CREATE EXTENSION IF NOT EXISTS vector`; текущая миграция использует `gen_random_uuid()` без extension (`services/decision-engine/migrations/20260216002310_init_tables.sql:5`).

### 6.4 Индексы

- `messages(session_id, created_at)`.
- `sessions(channel, external_user_id, status)`.
- `operator_queue(status, created_at)`, `operator_queue(assigned_operator_id, status)`.
- `intent_examples USING hnsw (embedding vector_cosine_ops)` или `ivfflat` после выбора размерности.
- `knowledge_chunks USING hnsw (embedding vector_cosine_ops)`.
- `decision_logs(session_id, created_at)`.

### 6.5 Какие audit tables реально писать

Писать обязательно:

- `messages` для user/bot/operator сообщений;
- `decision_logs` для intent/confidence/fallback;
- `actions_log` для business lookup/operator queue actions;
- `transitions_log` для mode/state changes;
- `operator_queue` lifecycle.

Не писать raw prompt/user text в отдельные debug logs. User message already stored in `messages`; logs должны ссылаться на `message_id`.

### 6.6 Новая схема с нуля

Целевая схема должна собираться на пустой БД в таком порядке:

1. Extensions: `pgcrypto`, `vector`.
2. Core identity: `users`, `operators`, `sessions`, `session_context`.
3. Messaging: `messages`, `message_events` for idempotency.
4. Knowledge base: `intents`, `intent_examples`, `knowledge_articles`, `knowledge_chunks`, `quick_replies`.
5. Embeddings: vector columns and vector indexes for `intent_examples` and `knowledge_chunks`.
6. Decision audit: `decision_logs`, `decision_candidates`.
7. Business demo data: `demo_bookings`, `demo_workspace_bookings`, `demo_payments`, `demo_accounts` or explicit JSON-backed provider tables.
8. Operator handoff: `operator_queue`, `operator_assignments`, `operator_events`.
9. Runtime audit: `actions_log`, `transitions_log`.
10. Seed loader: insert BRD catalog, KB, demo business data, demo operators.
11. sqlc generation and repository tests.

Write flow сразу проектируется транзакционно:

`inbound message -> preprocess/embedding decision -> decision log -> action logs -> transition log -> session context update -> outbound bot/operator message`.

Если любой обязательный write в этом flow падает, публичный ответ должен быть controlled error, а не “продолжаем без истории”. Для demo-системы сохранность истории и состояния является частью BRD.

### 6.7 Что хранить и что не хранить

Хранить:

- session/user/channel identifiers;
- messages user/bot/operator;
- intent key, confidence, top candidates;
- active topic, mode, fallback count, operator status;
- action metadata and safe response payloads;
- KB/intent examples and embeddings.

Не хранить:

- секреты внешних сервисов;
- raw LLM prompts/responses в production;
- карточные данные, платежные реквизиты, реальные персональные данные сверх demo identifiers;
- неизолированные browser console logs.

### 6.8 Seed/demo data plan

Seed data должен быть достаточным, чтобы все BRD use-cases проходили без ручного редактирования БД:

- `intents.json`: все intent keys, category, response_key, escalation policy, fallback policy.
- `knowledge-base.json`: FAQ/правила/цены/адрес/часы/инструкции по оплате, сайту, входу.
- `intent_examples.json`: минимум 8-15 русскоязычных пользовательских формулировок на каждый основной intent.
- `demo-bookings.json`: найденная запись, pending, cancelled, completed, not_found case.
- `demo-workspace-bookings.json`: hot desk, fixed desk, small office, large office, unavailable/not_found.
- `demo-payments.json`: completed, pending, failed, debited_not_activated, refunded, not_found.
- `demo-users.json`: active account, blocked/demo issue, not_found.
- `demo-operators.json`: минимум 2 оператора: available and offline/busy.
- `mock-external-services.json` или отдельные service fixtures: expected response/error cases for booking/workspace/payment/account/pricing providers.

Каждый seed object должен иметь stable ID, чтобы E2E-тесты могли ссылаться на конкретный номер записи/платежа/брони и получать предсказуемый ответ.

## 7. API и контракты

### 7.1 Decision-engine HTTP API

Текущий API: `GET /health`, `POST /decide`, `GET /config_llm` (`services/decision-engine/internal/transport/http/router.go:37`). Целевой API:

- `POST /api/v1/sessions` - создать/возобновить session.
- `POST /api/v1/messages` или `POST /api/v1/decide` - обработать user message.
- `GET /api/v1/sessions/{id}/messages` - история для клиента/operator UI.
- `GET /api/v1/domain/schema` - intents/states/actions/quick replies без LLM naming.
- `GET /api/v1/health` - liveness.
- `GET /api/v1/ready` - DB, migrations, NLP, pgvector, seed data.

Response `/decide` должен вернуть:

```json
{
  "session_id": "uuid",
  "message_id": "uuid",
  "mode": "standard",
  "active_topic": "payment",
  "intent": "payment_not_activated",
  "confidence": 0.86,
  "text": "заранее заданный ответ",
  "quick_replies": [
    {"id": "operator", "label": "Связаться с оператором", "action": "request_operator"}
  ],
  "handoff": null
}
```

### 7.2 NLP service API

Текущий `services/llm` API `POST /llm/decide` должен уйти из runtime. Целевой Python NLP:

- `POST /api/v1/preprocess` -> `{tokens, lemmas, normalized_text}`.
- `POST /api/v1/embed` -> `{embedding, model, dimension}`.
- `POST /api/v1/embed/batch` for seed/build scripts.
- `GET /api/v1/health`.
- `GET /api/v1/ready`.

Нельзя возвращать `intent/state/actions` из Python NLP, если source of truth остается в Go/Postgres. Python отвечает за NLP operations, decision-engine отвечает за бизнес-решение.

### 7.3 WebSocket contract

Текущий DTO: input `{type,text}`, output `{type,text,options,state}` (`services/transport-adapters/website/backend/internal/dto/websocket.go:3`). Целевой contract:

- client -> server: `session.start`, `message.user`, `quick_reply.selected`, `operator.close`.
- server -> client: `session.started`, `message.bot`, `message.operator`, `handoff.queued`, `handoff.accepted`, `handoff.closed`, `error`.
- все events содержат `session_id`, `message_id` или `event_id`, `timestamp`, `correlation_id`.
- quick replies object-based, not display string based.

`contracts/websocket.json` должен стать реальным JSON Schema или удалиться как ложный контракт.

### 7.4 Operator escalation contract

Минимум:

- `POST /api/v1/operator/queue/{session_id}/request`.
- `GET /api/v1/operator/queue?status=waiting`.
- `POST /api/v1/operator/queue/{handoff_id}/accept`.
- `POST /api/v1/operator/sessions/{session_id}/messages`.
- `POST /api/v1/operator/queue/{handoff_id}/close`.

События в WS должны сохраняться в той же session и `messages.sender_type='operator'`, что уже разрешено миграцией (`services/decision-engine/migrations/20260216002310_init_tables.sql:26`).

### 7.5 Business data lookup contracts

Так как BRD говорит о симулированных API и тестовых данных, contracts должны быть полноценными, но provider implementations в demo ходят в mock external services:

- `BookingLookupRequest {session_id, identifier, channel_user_id}`.
- `BookingLookupResponse {found, booking_number, service, date, status, source:"mock_external"}`.
- Аналогично workspace/payment/account.

Business actions не должны менять бронь, оплату или аккаунт.

### 7.7 Knowledge, external data and QA resolution model

Вопрос-ответ в целевой системе должен быть не “ответом из LLM”, а управляемым разрешением запроса через intent catalog, knowledge base and read-only providers.

#### 7.7.1 Источники данных

1. **Intent catalog.**  
   Карта понимания пользовательских формулировок. Содержит intent key, category, examples, response key, resolution type, fallback policy, escalation flags.

   Пример:

   ```json
   {
     "intent": "ask_workspace_prices",
     "category": "workspace",
     "examples": [
       "сколько стоит рабочее место",
       "цена аренды места",
       "почасовая аренда коворкинга"
     ],
     "resolution_type": "knowledge",
     "knowledge_key": "workspace.prices",
     "response_key": "workspace_types_prices"
   }
   ```

2. **Static knowledge base.**  
   Документация сервиса: цены, правила, FAQ, часы работы, адрес, инструкции по оплате, сайту, входу, отмене и возвратам. Хранится в `knowledge_articles` / `knowledge_chunks`, индексируется embeddings and pgvector.

   Пример:

   ```json
   {
     "knowledge_key": "workspace.prices",
     "category": "workspace",
     "title": "Цены на рабочие места",
     "body": "Горячее место - 200 руб/час. Фиксированное место - 400 руб/час...",
     "linked_intents": ["ask_workspace_prices"]
   }
   ```

3. **Business external providers.**  
   Данные, которые зависят от пользователя, номера записи, платежа или текущего состояния внешней системы: booking status, workspace booking status, payment status, account status, operator availability. В demo эти providers вызывают mock external services, но contracts описываются как полноценные внешние интеграции.

4. **Operator context.**  
   История сообщений, active topic, last intent, confidence, action summaries. Используется только для handoff and operator UI.

#### 7.7.2 Как происходит question-answer

Базовый flow:

1. Пользователь отправляет текст.
2. Decision-engine сохраняет inbound message.
3. NLP service делает preprocessing and embedding.
4. Decision-engine ищет ближайший intent/example in pgvector.
5. DecisionPolicy проверяет confidence, ambiguity, topic switch and fallback count.
6. Intent metadata определяет `resolution_type`.
7. Resolver выполняет один из путей:
   - `knowledge` - берет article/chunk and response template;
   - `business_lookup` - вызывает read-only provider/mock external service;
   - `operator_handoff` - создает handoff;
   - `clarification` - просит уточнение;
   - `static_response` - отдает canned response без data lookup.
8. ResponseRenderer подставляет данные в template.
9. Decision-engine пишет decision/action/transition logs and outbound message.
10. Adapter отправляет ответ/quick replies пользователю.

#### 7.7.3 Resolution types

Каждый intent обязан иметь явный `resolution_type`:

```json
{
  "intent": "ask_booking_status",
  "resolution_type": "business_lookup",
  "required_action": "find_booking",
  "required_entities": ["booking_identifier"],
  "fallback_response_key": "booking_request_identifier"
}
```

Допустимые типы:

- `static_response` - ответ только из predefined response.
- `knowledge` - ответ из базы знаний / документации.
- `business_lookup` - read-only lookup во внешней бизнес-системе через mock provider.
- `operator_handoff` - передача оператору.
- `clarification` - уточнение, если не хватает данных.
- `fallback` - низкая уверенность или ошибка resolution.

#### 7.7.4 Пример: вопрос про цены

Если пользователь пишет “Сколько стоит горячее место?”:

1. semantic matcher выбирает `ask_workspace_prices`;
2. intent metadata указывает `resolution_type=knowledge`, `knowledge_key=workspace.prices`;
3. knowledge resolver берет актуальную статью/чанк с ценами;
4. renderer собирает response `workspace_types_prices`;
5. пользователь получает canned answer with prices and quick replies.

Если цены должны считаться внешними динамическими данными, intent может быть переведен на `business_lookup`:

```text
ask_workspace_prices -> PricingProvider.GetWorkspacePrices() -> mock-pricing-service -> rendered response
```

В обоих вариантах пользователь получает заранее заданный шаблон, но фактические значения цен приходят либо из KB, либо из mock external pricing contract.

#### 7.7.5 Пример: статус записи

Если пользователь пишет “Проверь запись БРГ-482910”:

1. semantic matcher выбирает `ask_booking_status`;
2. entity extractor находит `БРГ-482910`;
3. resolver вызывает `BookingProvider.GetBookingStatus`;
4. provider обращается к `mock-booking-service`;
5. mock service возвращает typed response;
6. renderer подставляет номер, услугу, мастера, дату, статус;
7. action log пишет provider, request id, status, duration, source `mock_external`.

#### 7.7.6 Контракт mock external services

Mock external services должны быть не случайными генераторами внутри action, а явными сервисами/адаптерами с контрактами:

- `mock-booking-service`: записи, отмены, переносы, статусы.
- `mock-workspace-service`: типы мест, брони, доступность.
- `mock-payment-service`: статусы платежей, списание без активации, failed/pending/completed.
- `mock-account-service`: статус аккаунта, email/phone lookup, blocked/not_found.
- `mock-pricing-service` или KB-backed pricing: цены услуг and workspace.

Для каждого mock service:

- OpenAPI/JSON schema или documented DTO;
- deterministic seed data;
- timeout/error behavior;
- negative cases;
- contract tests;
- E2E happy and failure paths.

#### 7.7.7 Поведение при недоступности внешней системы

Если provider/mock external service недоступен:

- пользователь не видит raw error;
- action log пишет `status=failed`, `provider`, `duration_ms`, safe error code;
- response renderer возвращает controlled fallback: “Не удалось проверить данные, попробуйте позже или подключим оператора”;
- repeated provider failure может trigger operator handoff;
- E2E должен покрывать provider timeout/error.

#### 7.7.8 Актуализация документации и данных

Для demo достаточно seed files. Для будущего развития:

- KB articles должны иметь `source`, `version`, `updated_at`;
- embeddings пересчитываются при изменении article/example;
- prices/rules can be stored in KB when static, or in provider when dynamic;
- readiness может проверять, что embeddings dimension соответствует текущей NLP model;
- tests должны ловить orphan intents: intent points to missing response/knowledge/action.

### 7.6 Health/readiness

Liveness можно оставить простым. Readiness должна проверять:

- DB connection;
- migrations version;
- `vector` extension;
- NLP service available;
- intent catalog seeded;
- operator queue tables exist.

Текущий Python health возвращает только `domain_loaded` (`services/llm/app/api/routes/health.py:9`), а это будет неактуально после отказа от LLM domain pull.

## 8. Security и privacy

| Риск | Факт | План исправления | Проверка |
|---|---|---|---|
| User/session isolation | `chat_id=1` в website и decision-engine (`services/transport-adapters/website/backend/internal/websocket/handler.go:56`, `services/decision-engine/internal/transport/http/handler/decide.go:64`) | session cookie/client id, no default in prod, unique active session constraint | E2E two browsers: different history/state |
| WebSocket Origin | `CheckOrigin` always true (`services/transport-adapters/website/backend/internal/websocket/handler.go:16`) | allowlist from env, reject unknown Origin | WS test rejected Origin |
| HTTPS/WSS | frontend hardcodes `ws://` (`services/transport-adapters/website/frontend/assets/js/websocket.js:97`) | protocol switch `wss` under HTTPS, reverse proxy doc | Browser test under HTTPS base |
| DOM XSS | inline `onclick` with escaped text (`services/transport-adapters/website/frontend/assets/js/chat.js:124`) | `createElement`, `textContent`, `addEventListener`, no inline JS | XSS regression fixture with quotes/HTML |
| Raw internal errors | `/decide` returns `%v` (`services/decision-engine/internal/transport/http/handler/decide.go:79`); LLM returns internal error (`services/llm/app/api/routes/decide.py:71`) | stable public codes, request id, server-side redacted detail | Contract test no internal substrings |
| Prompt/user text logging | website logs text (`services/transport-adapters/website/backend/internal/websocket/handler.go:114`); GigaChat logs prompt/response (`services/llm/app/services/llm/gigachat_client.py:203`) | redaction middleware, prod log level, no browser debug logs | Log tests |
| TLS verification | `verify=False` (`services/llm/app/services/llm/gigachat_client.py:66`) | remove with LLM service; if kept dev-only flag with default secure | Config validation |
| Secrets/env handling | no root env standard (`AUDIT.md:39`) | `.env.example`, no committed secrets, compose secrets | secret scan |
| Rate limits/auth | no auth/rate limits found | demo rate limit per session/IP, operator auth at least basic demo token | API tests 429/401 |
| Basic injection protection | sqlc params good, prompt path bad | no runtime prompt; DOM safe rendering; length limits | fuzz/light property tests |

## 9. Тестовая стратегия

### Unit tests

- Semantic matcher: top-k, threshold, ambiguity, topic switch.
- Normalization/preprocess: Russian text, lemmatization unavailable fallback.
- ResponseRenderer: placeholders, missing data fallback.
- DecisionPolicy: first fallback, repeated fallback, manual operator.
- Action contracts: booking/payment/workspace/account demo fixtures.

Где: `services/decision-engine/internal/app/decision/...`, `internal/infrastructure/nlp/...`, `internal/app/presenter/...`.

### Contract tests

- Decision API JSON schemas for `/api/v1/decide`, session start, error codes.
- NLP `/preprocess` and `/embed` schemas.
- WebSocket events and quick reply payloads.
- Operator queue API.

Где: `services/decision-engine/internal/transport/http/...`, `services/transport-adapters/website/contracts/...`, `services/nlp-service/tests/...`.

### Integration tests

- Postgres migrations up/down.
- Session isolation: two users, two active sessions.
- Transactional message handling: inbound, decision log, action log, transition log, outbound.
- Operator handoff lifecycle.
- pgvector search with seeded embeddings.

Где: `services/decision-engine/tests/integration` или Go package-level integration tests with testcontainers/temporary DB.

### Migration tests

- fresh empty DB initializes successfully;
- `gen_random_uuid()` extension present;
- `vector` extension present;
- all target tables exist;
- FK `sessions.user_id`;
- unique active session constraints;
- seed loader inserts full demo catalog;
- repeated seed run is idempotent or explicitly resets demo data;
- no legacy/backfill migrations are required.

### Полная E2E-матрица

E2E-тесты должны быть отдельным обязательным блоком готовности. Они должны запускать полный продукт: Postgres + NLP service + decision-engine + website adapter + browser + operator UI/API. Моки допустимы только на уровне внешнего мира, которого по BRD нет: реальные платежи/CRM/бронирование не подключаются, вместо них используются demo seed data.

| E2E ID | Flow | Вход | Ожидание в UI/API | Ожидание в БД |
|---|---|---|---|---|
| E2E-001 | Новый пользователь | открыть web chat | создана новая session, показано приветствие/меню | `sessions`, `session_context`, стартовое bot message |
| E2E-002 | Изоляция пользователей | два браузера задают разные вопросы | разные истории, разные active_topic | разные `session_id`, сообщения не пересекаются |
| E2E-003 | FAQ услуги/цены | “какие услуги и цены” | intent `services_prices`, canned answer, quick replies | `decision_logs` с confidence, bot message |
| E2E-004 | Правила отмены | “как отменить запись” | ответ про правила отмены, без business action | decision/action logs без lookup action |
| E2E-005 | Адрес/часы | “где вы находитесь и когда работаете” | contact/location response | correct intent and response key |
| E2E-006 | Статус записи found | “статус записи БРГ-482910” | данные записи подставлены без `{placeholder}` | action log `find_booking`, bot message with rendered values |
| E2E-007 | Статус записи not found | unknown booking id | controlled not_found response + operator quick reply | action log not_found, no crash |
| E2E-008 | Правила переноса записи | “можно перенести запись?” | canned reschedule/cancellation guidance | no mutation in demo booking tables |
| E2E-009 | Рабочие места prices | “сколько стоит рабочее место” | prices/types response | intent `workspace_prices` |
| E2E-010 | Workspace booking found | demo workspace booking id | rendered workspace booking status | `find_workspace_booking` action log |
| E2E-011 | Workspace unavailable | “место недоступно” | explanation + operator/admin quick reply | decision log active_topic workspace |
| E2E-012 | Payment completed | demo payment id completed | payment status rendered | `find_payment` action log |
| E2E-013 | Payment failed | “оплата не прошла” | failed payment instruction | intent `payment_not_passed` |
| E2E-014 | Debited not activated | “деньги списались, услуга не активировалась” | instruction + offer operator | active_topic payment, optional escalation suggestion |
| E2E-015 | Payment not found | unknown payment id | not_found response + retry/operator replies | action log not_found |
| E2E-016 | Site not loading | “сайт не открывается” | browser/cache/basic instruction | intent `site_not_loading` |
| E2E-017 | Login problem | “не могу войти” | login instruction / reset password quick replies | active_topic tech_issue/account |
| E2E-018 | Code not received | “не приходит код” | code troubleshooting response | intent `code_not_received` |
| E2E-019 | Account lookup found | demo email/phone | account status response | `find_user_account` action log |
| E2E-020 | Account lookup not found | unknown email/phone | safe not_found response | no internal errors |
| E2E-021 | Complaint handoff | user describes complaint | queue handoff created, user sees waiting operator | `operator_queue` waiting, context snapshot |
| E2E-022 | Manual operator | click/write “оператор” | bot queues handoff | `operator_queue.reason=manual_request` |
| E2E-023 | Repeated fallback | two unclear messages | first clarification, second handoff/offer operator | fallback_count increments, queue or explicit offer |
| E2E-024 | Operator accepts | operator opens queue and accepts | user sees operator connected | queue status accepted, assignment row |
| E2E-025 | Operator replies | operator sends message | user chat shows operator message, bot silent | `messages.sender_type=operator` |
| E2E-026 | User writes during operator mode | user sends next message | routed to operator, no bot auto response | user message stored, no bot message generated |
| E2E-027 | Operator closes handoff | operator closes conversation | user sees close/return-to-bot state | queue closed, transition log |
| E2E-028 | Restore after restart | restart decision-engine/website | same browser resumes history/context | persisted messages/session_context used |
| E2E-029 | Quick reply typed payload | click quick reply | payload id/action sent, not label parsing | message event records quick_reply id |
| E2E-030 | XSS regression | malicious text/quick reply fixture | no script execution, text rendered safely | message stored as text only |
| E2E-031 | Origin rejection | connect WS from disallowed Origin | connection rejected | security log without raw user data |
| E2E-032 | Public error masking | force NLP/DB error in test mode | user gets stable error code/message | internal detail only server-side |
| E2E-033 | NLP unavailable fallback | stop NLP or fake timeout | controlled fallback/operator offer under 3s | decision log reason `nlp_unavailable` |
| E2E-034 | Latency budget | run representative FAQ/lookup flows | p95 under 3 seconds in demo env | metrics/test report |
| E2E-035 | No LLM runtime | full E2E suite | no HTTP calls to `/llm/decide`, no Ollama/GigaChat env needed | test spy/assertion passes |
| E2E-036 | Mock external contract | force booking/payment/workspace/account provider calls | responses follow documented contracts | action logs source `mock_external` |
| E2E-037 | Provider unavailable | stop/fail one mock external service | controlled fallback/operator offer, no raw error | action log failed provider, decision log fallback |
| E2E-038 | Knowledge-backed prices | ask current prices | answer comes from KB/pricing provider, not hardcoded JS | decision/action evidence points to knowledge/provider source |

E2E implementation requirements:

- tests live in `tests/e2e`;
- use Playwright for browser/client and operator UI;
- use API helpers for setup/seed reset;
- test environment starts from empty DB;
- each test either resets DB or uses isolated session/user ids;
- every business category has at least one happy path and one negative/fallback path;
- every external provider contract has at least one happy path, not_found path and unavailable/error path;
- E2E artifacts include trace/screenshot/video on failure;
- CI must run a smoke subset on every push and the full matrix before declaring project complete.

### Security regression tests

- Rejected Origin.
- No raw internal error in public body.
- No inline onclick in quick replies.
- Rate limit.
- Log redaction fixtures.

### LLM mocked tests

Since target runtime has no generative LLM, “LLM mocked tests” should become legacy tests only if legacy path remains. For target tests, mock NLP embeddings instead:

- fake embed service returns vectors;
- fake service timeout triggers fallback;
- no call is made to `/llm/decide`.

## 10. Roadmap работ

### 10.0 Плавная линия реализации без legacy-петли

Так как старую БД и старый LLM-first runtime поддерживать не нужно, реализация должна идти не через долгую совместимость, а через последовательное построение новой системы:

1. **Зафиксировать contracts before code:** HTTP/WS/NLP/operator DTO, response format, session model, E2E fixture IDs.
2. **Собрать fresh DB:** новая init migration, pgvector, target tables, seed loader.
3. **Поднять skeleton продукта:** compose запускает DB, decision-engine, nlp-service, website; `/ready` честно показывает, чего не хватает.
4. **Сделать session identity:** web chat создает/возобновляет session, два браузера изолированы.
5. **Сделать decision pipeline без NLP complexity:** fake embedding/matcher возвращает seed intent, чтобы сразу пройти первый E2E.
6. **Подключить real NLP `/embed`:** заменить fake provider, проверить pgvector search.
7. **Закрыть canned response rendering:** placeholders, quick replies, no raw template leaks.
8. **Добавить business demo providers:** записи, рабочие места, платежи, аккаунт, услуги/правила.
9. **Добавить fallback policy:** confidence, ambiguity, fallback_count, topic switch.
10. **Добавить operator handoff:** queue, accept, reply, close, bot silence in operator mode.
11. **Расширить E2E matrix до полного BRD:** каждый бизнес-сценарий, negative paths, security, restart.
12. **Удалить legacy:** LLM service, stale configs, unused clients, tracked caches, empty contracts.
13. **Финализировать продукт:** README, diagrams, compose smoke, full test report.

Каждый шаг должен оставлять продукт запускаемым. Если шаг меняет контракт, сначала обновляются schema/tests, затем runtime, затем UI.

### P0: блокеры соответствия BRD

0. **Перейти на greenfield implementation contract.**
   - Модули: root docs, migrations, compose, seeds.
   - Зависимости: решение владельца, что legacy DB/runtime не поддерживаются.
   - Готовность: старые LLM/rules/transitions больше не считаются source of truth; новая схема стартует на пустой БД.
   - Проверки: clean DB bootstrap test.
   - Риск: средний, но снижает общий объем работ.

1. **Зафиксировать целевой domain catalog.**
   - Модули: `configs/responses.json`, new seeds, domain intent/state/action.
   - Зависимости: BRD categories.
   - Готовность: все BRD categories mapped to intents/responses/quick replies.
   - Проверки: startup validation на response keys/placeholders.
   - Риск: средний.

2. **Убрать LLM-first из основного runtime.**
   - Модули: `message_worker.go`, `internal/infrastructure/llm`, `services/llm`.
   - Зависимости: DecisionService skeleton.
   - Готовность: `/decide` работает с fake semantic matcher без `/llm/decide`.
   - Проверки: integration test with fake NLP; `rg "llmClient.Decide"` не в production path.
   - Риск: высокий.

3. **Session identity and isolation.**
   - Модули: website handler/client DTO, decision handler, sessions schema.
   - Готовность: два браузера не смешиваются.
   - Проверки: E2E multi-client.
   - Риск: высокий privacy.

4. **pgvector KB schema and semantic matching MVP.**
   - Модули: migrations, sqlc, NLP service, matcher.
   - Готовность: seeded examples produce deterministic intents with confidence.
   - Проверки: pgvector search integration.
   - Риск: высокий.

5. **Fallback and operator queue MVP.**
   - Модули: session context, operator queue table/API, escalation action.
   - Готовность: repeated low confidence queues handoff.
   - Проверки: queue lifecycle integration.
   - Риск: высокий.

6. **Полная E2E-база с первого этапа.**
   - Модули: `tests/e2e`, compose test profile, seed reset.
   - Зависимости: session identity, base decision API, seed data.
   - Готовность: первые 8-10 E2E проходят по web chat/API.
   - Проверки: Playwright traces + DB assertions.
   - Риск: высокий, но без этого нельзя доказать “полностью рабочий сервис”.

### P1: архитектурная стабилизация

1. Response renderer with placeholders.
2. Action/provider contracts and demo fixtures.
3. Transactional persistence for messages/actions/transitions.
4. Replace stale transitions/rules with validated source of truth.
5. Versioned HTTP/WS/NLP contracts.
6. Root Docker Compose with DB/NLP/decision/website.
7. Operator UI/API до полного handoff flow.
8. Full E2E matrix for all business operations.

### P2: качество, эксплуатация, UX

1. Русскоязычный frontend.
2. Operator UI: queue, history, reply, close.
3. Security hardening: Origin, WSS, DOM safe buttons, raw error masking.
4. Observability: readiness, structured redacted logs, latency metrics.
5. Tests: e2e/security/perf.
6. Diagrams: UML component, ERD, deployment, sequence diagrams.

### P3: улучшения/масштабирование

1. Admin UI for KB/intent examples.
2. Re-embedding jobs and versioned embedding model.
3. Better topic-switch detection.
4. Basic analytics dashboard for unresolved intents.
5. Optional auth model for operators/admins.

## 11. Backlog задач

| ID | Название | Цель | Scope | Affected files/modules | Acceptance criteria | Tests | Dependencies | Risk |
|---|---|---|---|---|---|---|---|---|
| CB-BRD-001 | BRD intent catalog | Перевести BRD в конкретные intents/use-cases | categories, examples, responses, quick replies | `configs/responses.json`, new `seeds/intents.*`, domain intent | Все BRD categories покрыты | catalog validation | none | M |
| CB-BRD-002 | Replace LLM runtime with DecisionService | Убрать генеративный LLM из обработки | worker, service, interfaces | `message_worker.go`, new `internal/app/decision` | `/decide` no longer calls `/llm/decide` | integration fake matcher | 001 | H |
| CB-BRD-003 | NLP service contract | Сделать Python NLP/embedding service | preprocess/embed/health | replace `services/llm` or create `services/nlp-service` | `/embed` returns dimensioned vector | Python contract tests | 002 | H |
| CB-BRD-004 | pgvector migrations | Хранить intent/KB embeddings | extensions, tables, indexes | migrations, sqlc | vector search works | migration + integration | 003 | H |
| CB-BRD-005 | Semantic matcher | Intent detection by embeddings | top-k, threshold, ambiguity | `internal/app/decision`, repos | confidence below threshold falls back | unit/integration | 004 | H |
| CB-BRD-006 | Session identity | Изолировать пользователей | cookie/client id/session id | website handler, DTO, sessions schema | no shared chat_id=1 in prod | E2E two clients | none | H |
| CB-BRD-007 | Context model | Хранить active topic/mode/last intent/fallback count | schema + service | sessions/session_context | restart restores context | integration | 006 | H |
| CB-BRD-008 | Response renderer | Подставлять action data | placeholders, status labels | presenter/renderer, responses | no raw `{service}` in user response | unit | 001 | M |
| CB-BRD-009 | Demo business providers | Явные read-only mock lookups | booking/payment/workspace/account | actions/providers, fixtures | deterministic fixture results | unit/integration | 008 | M |
| CB-BRD-010 | Operator queue schema | Основа handoff | queue/assignments | migrations, repos | handoff item can be queued/accepted/closed | integration | 006 | H |
| CB-BRD-011 | Escalation action | Реальная передача оператору | action + policy | `escalate_to_operator.go`, processor | manual and auto escalation work | integration | 010 | H |
| CB-BRD-012 | Operator API/UI | Оператор отвечает в той же сессии | queue page, message send | website/operator frontend, decision API | bot stops auto-reply in operator mode | E2E | 011 | H |
| CB-BRD-013 | WebSocket v1 contract | Стабилизировать клиентский протокол | event types/schemas | `contracts/websocket.json`, DTO, JS | schema matches events | contract tests | 006 | M |
| CB-BRD-014 | Quick replies typed objects | Не связывать display text с командой | id/label/action/payload | responses, renderer, frontend | clicking sends id/payload | UI test | 013 | M |
| CB-BRD-015 | Remove inline JS / XSS fix | Закрыть DOM XSS | frontend rendering | `chat.js` | no inline onclick | security test | 014 | H |
| CB-BRD-016 | Origin/WSS hardening | Protected transport | WS URL, allowlist | `websocket.js`, WS handler, config | WSS under HTTPS, Origin rejected | WS tests | 013 | H |
| CB-BRD-017 | Public error codes | Не раскрывать internals | HTTP/WS errors | handlers, client | stable codes only | contract/security | none | M |
| CB-BRD-018 | Action/transition persistence | Реальный audit trail | transaction logs | processor, worker, repos | logs written per message | integration | 007 | M |
| CB-BRD-019 | Docker compose demo | Воспроизводимый запуск | DB/NLP/decision/website | root compose, env examples | one command boot + health | smoke test | 003/004 | M |
| CB-BRD-020 | Diagrams package | Выполнить учебный результат | UML/ERD/deployment/sequences | `docs/diagrams` | diagrams reflect implemented contracts | doc review | P1 APIs | L |
| CB-BRD-021 | Test pyramid baseline | Не заявлять готовность без проверок | unit/contract/integration/e2e | all services | CI runs green core checks | CI | 002+004+006 | M |
| CB-BRD-022 | Clean generated artifacts | Убрать tracked pycache | repo hygiene | `__pycache__`, `.gitignore` | no tracked pycache | git check | none | L |
| CB-BRD-023 | Greenfield DB reset | Переписать БД под целевой продукт без legacy support | fresh migrations, remove backfill assumptions | migrations, sqlc, compose | empty DB boots and seeds cleanly | migration/bootstrap tests | 001 | H |
| CB-BRD-024 | Seed demo dataset | Дать данные для всех use-cases | intents, KB, bookings, payments, workspaces, users, operators | `seeds/*`, seed loader | all E2E IDs have stable fixture data | seed tests | 023 | M |
| CB-BRD-025 | Full E2E suite | Доказать рабочий продукт от и до | client chat, operator UI, business flows, security, restart, mock external providers | `tests/e2e`, compose test profile | E2E-001..E2E-038 pass | Playwright + DB assertions | 002-024 | H |
| CB-BRD-026 | Root product README | Объяснить запуск и работу сервиса | setup, architecture, tests, limitations | `README.md`, `.env.example` | new developer can run demo from scratch | doc smoke | 019/025 | L |
| CB-BRD-027 | External provider contracts | Описать внешние бизнес-интеграции полностью | booking, workspace, payment, account, pricing provider DTO/errors | contracts, provider interfaces, mock services | all providers have schema, mock implementation, contract tests | contract/e2e | 024 | H |

## 12. Открытые вопросы

1. **Размерность и модель embeddings.** Временное решение: выбрать легкую multilingual/sentence-transformer модель для demo. Требует ответа перед финальным seed/index.
2. **Где должен жить semantic search: Go+pgvector или Python `/semantic-match`?** Рекомендация: Go owns decision, Postgres owns search; Python только embeddings. Можно временно делать `/embed`.
3. **Нужна ли авторизация операторов в учебном scope?** Временное решение: demo token/basic auth. Для полноценного operator UI нужен ответ.
4. **Достаточен ли предложенный seed dataset для учебной защиты/демо?** Временное решение: использовать набор из раздела 6.8 и расширять только если E2E-матрица выявит дырки.
5. **Сколько fallback попыток до оператора?** BRD говорит “при повторной неудаче”. Временное решение: 2 consecutive low-confidence attempts.
6. **Какие confidence thresholds принять?** Временное решение: `match >= 0.78`, ambiguity delta `>= 0.08`, уточнение при ниже; уточнить после тестового датасета.
7. **Нужен ли admin UI для базы знаний в scope результата?** BRD требует структуру БД и демонстрацию, но не admin UI. Можно оставить seed files.
8. **Какие данные можно показывать в operator context?** Временное решение: последние 20 сообщений, active topic, last intent, action summaries без raw secrets.
9. **Должен ли console adapter остаться в итоговом проекте?** Рекомендация: оставить как dev tool, не считать пользовательским каналом BRD.
10. **Нужно ли сохранять legacy LLM для сравнения?** Рекомендация: нет в runtime; если нужен для учебной истории, вынести в `legacy/` или `tools/`, не подключать compose and E2E.

## 13. Конкретный целевой intent catalog

Этот список должен стать основой `seeds/intents.json`. Для каждого intent нужны examples, response_key, resolution_type, quick replies and E2E coverage.

| Intent | Category | Resolution type | Response / action | Примеры пользовательских фраз |
|---|---|---|---|---|
| `greeting` | system | `static_response` | `start` / `main_menu` | “привет”, “здравствуйте”, “добрый день” |
| `goodbye` | system | `static_response` | `goodbye` | “пока”, “до свидания”, “спасибо, все” |
| `return_to_menu` | system | `static_response` | `main_menu` | “в меню”, “назад”, “главное меню” |
| `reset_conversation` | system | `static_response` | `start` + reset context | “начать заново”, “сбросить диалог” |
| `request_operator` | operator | `operator_handoff` | `operator_handoff_requested` | “оператор”, “позовите человека”, “хочу в поддержку” |
| `ask_booking_info` | booking | `knowledge` | `booking_info` | “как записаться”, “как работает запись” |
| `ask_booking_status` | booking | `business_lookup` | `find_booking` -> `booking_found/not_found` | “проверь запись”, “статус записи БРГ-482910” |
| `ask_cancellation_rules` | booking | `knowledge` | `booking_cancellation_rules` | “как отменить запись”, “правила отмены” |
| `ask_reschedule_rules` | booking | `knowledge` | `booking_reschedule_rules` | “можно перенести запись”, “перенос брони” |
| `booking_not_found` | booking | `clarification` | `booking_request_identifier` / operator | “не вижу запись”, “запись пропала” |
| `ask_workspace_info` | workspace | `knowledge` | `workspace_info` | “какие есть рабочие места”, “что за коворкинг” |
| `ask_workspace_prices` | workspace | `knowledge` or `business_lookup` | `workspace_types_prices` or `PricingProvider` | “сколько стоит место”, “цена аренды” |
| `ask_workspace_rules` | workspace | `knowledge` | `workspace_rental_rules` | “правила аренды”, “можно шуметь” |
| `ask_workspace_status` | workspace | `business_lookup` | `find_workspace_booking` | “статус брони места WS-1001” |
| `workspace_unavailable` | workspace | `knowledge` + optional operator | `workspace_unavailable` | “нет свободных мест”, “место недоступно” |
| `ask_payment_status` | payment | `business_lookup` | `find_payment` | “статус платежа PAY-123456” |
| `payment_not_passed` | payment | `knowledge` | `payment_failed` | “оплата не прошла”, “платеж отклонен” |
| `payment_not_activated` | payment | `knowledge` + operator offer | `payment_debited_not_activated` | “деньги списались, услуга не активировалась” |
| `ask_refund_rules` | payment | `knowledge` | `payment_refund_rules` | “как вернуть деньги”, “возврат оплаты” |
| `ask_site_problem` | tech_issue | `knowledge` | `tech_site_not_loading` | “сайт не работает”, “страница не открывается” |
| `login_not_working` | tech_issue/account | `knowledge` | `tech_login_problem` | “не могу войти”, “ошибка входа” |
| `code_not_received` | tech_issue/account | `knowledge` | `tech_code_not_received` | “не приходит код”, “смс нет” |
| `ask_account_help` | account | `knowledge` | `account_category` | “помощь с аккаунтом”, “что с профилем” |
| `ask_account_status` | account | `business_lookup` | `find_user_account` | “проверь аккаунт”, “аккаунт по телефону” |
| `ask_services_info` | services | `knowledge` | `services_prices` | “какие услуги есть”, “услуги салона” |
| `ask_prices` | services | `knowledge` or `business_lookup` | `services_prices` or `PricingProvider` | “цены на услуги”, “сколько стоит маникюр” |
| `ask_rules` | services | `knowledge` | `services_rules` | “правила”, “условия сервиса” |
| `ask_location` | services | `knowledge` | `services_location` | “адрес”, “где находитесь”, “часы работы” |
| `ask_faq` | services | `knowledge` | `services_faq` | “частые вопросы”, “FAQ” |
| `report_complaint` | complaint | `operator_handoff` | `complaint_info_collected` + queue | “хочу пожаловаться”, “плохой сервис” |
| `complaint_master` | complaint | `operator_handoff` | queue with category master | “жалоба на мастера” |
| `complaint_premises` | complaint | `operator_handoff` | queue with category premises | “грязно”, “не работает оборудование” |
| `general_question` | other | `knowledge` or `fallback` | best KB chunk / clarify | “у меня вопрос” |
| `unknown` | fallback | `fallback` | `clarify_request` or operator | unclear/low confidence |

Coverage rule: каждый intent из таблицы должен иметь минимум 8 examples, 1 positive unit test, 1 contract/seed validation, and at least one E2E path for each category.

## 14. Mock external services: concrete contracts

Внешние системы в demo реализуются как mock services, но их контракты должны быть такими, чтобы позже можно было заменить mock на реальный adapter без переписывания decision logic.

### 14.1 `mock-booking-service`

Endpoints:

- `GET /api/v1/bookings/{booking_number}`
- `GET /api/v1/bookings?phone={phone}`

Response:

```json
{
  "found": true,
  "booking_number": "BRG-482910",
  "service": "Стрижка женская",
  "master": "Анна Петрова",
  "date": "2026-05-15",
  "time": "14:30",
  "status": "confirmed",
  "price": 1500,
  "source": "mock_external"
}
```

Errors:

- `404 booking_not_found`;
- `422 invalid_identifier`;
- `503 booking_service_unavailable`;
- timeout.

### 14.2 `mock-workspace-service`

Endpoints:

- `GET /api/v1/workspaces/prices`
- `GET /api/v1/workspace-bookings/{booking_number}`
- `GET /api/v1/workspaces/availability?date=YYYY-MM-DD&type=hot_desk`

Response for booking:

```json
{
  "found": true,
  "booking_number": "WS-1001",
  "workspace_type": "hot_desk",
  "date": "2026-05-15",
  "start_time": "10:00",
  "duration_hours": 3,
  "status": "active",
  "source": "mock_external"
}
```

### 14.3 `mock-payment-service`

Endpoints:

- `GET /api/v1/payments/{payment_id}`
- `GET /api/v1/payments?order_id={order_id}`

Response:

```json
{
  "found": true,
  "payment_id": "PAY-123456",
  "amount": 2000,
  "currency": "RUB",
  "status": "completed",
  "purpose": "Бронирование рабочего места",
  "created_at": "2026-05-14T10:15:00Z",
  "source": "mock_external"
}
```

Statuses required in seed data: `completed`, `pending`, `failed`, `debited_not_activated`, `refunded`, `not_found`.

### 14.4 `mock-account-service`

Endpoints:

- `GET /api/v1/accounts/by-phone/{phone}`
- `GET /api/v1/accounts/by-email/{email}`
- `GET /api/v1/accounts/{account_id}`

Response:

```json
{
  "found": true,
  "account_id": "USR-1001",
  "email": "demo@example.com",
  "phone": "+79990000001",
  "status": "active",
  "source": "mock_external"
}
```

### 14.5 `mock-pricing-service`

Endpoints:

- `GET /api/v1/prices/services`
- `GET /api/v1/prices/workspaces`

Use when prices should be treated as dynamic external data instead of static KB. If prices are static for the demo, this service can still exist and be seeded from the same `knowledge-base.json` to preserve the contract.

### 14.6 Provider adapter rules

All providers must:

- use typed request/response DTOs;
- have per-provider timeout below global 3-second budget;
- return stable error codes;
- never return raw upstream errors to user;
- write `actions_log` with `provider`, `source`, `status`, `duration_ms`;
- have contract tests for success, not_found, invalid input, unavailable;
- be replaceable by real HTTP clients later.

## 15. Detailed API DTO draft

### 15.1 Session API

`POST /api/v1/sessions`

```json
{
  "channel": "web",
  "client_id": "browser-generated-id"
}
```

Response:

```json
{
  "session_id": "uuid",
  "user_id": "uuid",
  "mode": "standard",
  "active_topic": null,
  "resumed": false
}
```

### 15.2 Message / decide API

`POST /api/v1/messages`

```json
{
  "session_id": "uuid",
  "event_id": "uuid",
  "type": "user_message",
  "text": "Сколько стоит горячее место?"
}
```

Response:

```json
{
  "session_id": "uuid",
  "user_message_id": "uuid",
  "bot_message_id": "uuid",
  "mode": "standard",
  "active_topic": "workspace",
  "intent": {
    "key": "ask_workspace_prices",
    "confidence": 0.88,
    "resolution_type": "knowledge"
  },
  "text": "Горячее место - 200 руб/час...",
  "quick_replies": [
    {"id": "workspace_rules", "label": "Правила аренды", "action": "select_intent", "payload": {"intent": "ask_workspace_rules"}},
    {"id": "operator", "label": "Связаться с оператором", "action": "request_operator", "payload": {}}
  ],
  "handoff": null
}
```

### 15.3 Operator queue API

`GET /api/v1/operator/queue?status=waiting`

```json
{
  "items": [
    {
      "handoff_id": "uuid",
      "session_id": "uuid",
      "reason": "manual_request",
      "active_topic": "payment",
      "last_intent": "payment_not_activated",
      "created_at": "2026-05-10T12:00:00Z",
      "preview": "Деньги списались, услуга не активировалась"
    }
  ]
}
```

`POST /api/v1/operator/queue/{handoff_id}/accept`

```json
{
  "operator_id": "operator-1"
}
```

`POST /api/v1/operator/sessions/{session_id}/messages`

```json
{
  "operator_id": "operator-1",
  "text": "Здравствуйте, я подключился и проверю ваш вопрос."
}
```

### 15.4 NLP API

`POST /api/v1/embed`

```json
{
  "text": "сколько стоит горячее место",
  "normalize": true
}
```

Response:

```json
{
  "normalized_text": "сколько стоить горячий место",
  "tokens": ["сколько", "стоить", "горячий", "место"],
  "embedding": [0.01, 0.02],
  "model": "demo-multilingual-embedding",
  "dimension": 384
}
```

### 15.5 Error shape

All public APIs use:

```json
{
  "error": {
    "code": "provider_unavailable",
    "message": "Не удалось проверить данные. Попробуйте позже или подключим оператора.",
    "request_id": "uuid"
  }
}
```

No public response may include stack traces, SQL errors, upstream bodies or raw internal exception text.

## 16. UI scope

### 16.1 Client web chat

Required screens/states:

- initial loading / session resume;
- connected chat;
- disconnected/reconnecting;
- bot messages;
- user messages;
- operator messages;
- typing/processing indicator;
- quick replies as buttons;
- fallback clarification;
- waiting for operator;
- operator connected;
- handoff closed / return to bot;
- controlled error state.

Required visible information:

- support chat title;
- connection status;
- message history;
- input box;
- send button;
- quick replies;
- operator status when applicable.

Not required:

- marketing landing page;
- user registration;
- payment UI;
- real booking management UI.

### 16.2 Operator UI

Required screens:

- login/demo operator selection;
- waiting queue list;
- session detail with full message history;
- context panel: active topic, last intent, confidence, fallback count, action summaries;
- accept handoff;
- reply input;
- close handoff;
- status filters: waiting, accepted, closed.

Operator UI acceptance:

- operator can accept one waiting session;
- operator can send message into same user chat;
- user messages after acceptance appear for operator;
- bot does not auto-answer while mode is `operator_connected`;
- close handoff writes transition and notifies user.

## 17. Diagrams checklist

Required diagram artifacts:

| Diagram | File | Must show |
|---|---|---|
| Component diagram | `docs/diagrams/component.md` | web chat, operator UI, decision-engine, nlp-service, Postgres/pgvector, mock external services |
| ERD | `docs/diagrams/erd.md` | users, sessions, session_context, messages, intents, intent_examples, KB, operator_queue, logs |
| Deployment diagram | `docs/diagrams/deployment.md` | docker compose services, networks, ports, volumes, healthchecks |
| Sequence: normal question | `docs/diagrams/sequence-user-message.md` | user message -> NLP -> pgvector -> response |
| Sequence: business lookup | `docs/diagrams/sequence-business-lookup.md` | intent -> provider -> mock external -> renderer |
| Sequence: fallback | `docs/diagrams/sequence-fallback.md` | low confidence -> clarification -> repeated failure -> handoff |
| Sequence: operator handoff | `docs/diagrams/sequence-operator-handoff.md` | queue -> accept -> operator message -> close |
| Sequence: restart/restore | `docs/diagrams/sequence-restore.md` | browser resumes session from persisted DB |

Diagrams are accepted only if they match actual contracts and E2E flows.

## 18. Implementation packages

Реализацию удобно вести пакетами, каждый пакет оставляет продукт запускаемым:

### Package 1: Contracts and schema

- HTTP/WS/NLP/provider DTOs;
- fresh DB schema;
- seed format;
- validation commands;
- first contract tests.

### Package 2: Seed and mock external services

- intent catalog;
- KB articles/chunks;
- business demo data;
- mock booking/payment/workspace/account/pricing services;
- provider contract tests.

### Package 3: NLP and semantic search

- `/preprocess`;
- `/embed`;
- batch embedding for seeds;
- pgvector indexing;
- semantic matcher;
- thresholds and ambiguity policy.

### Package 4: Decision-engine core

- session identity;
- message transaction;
- DecisionService;
- resolution types;
- response renderer;
- action logs and decision logs.

### Package 5: Client web chat

- session handshake;
- typed WS events;
- quick replies;
- reconnect/resume;
- operator mode states;
- XSS-safe rendering.

### Package 6: Operator workflow

- operator queue API;
- operator UI;
- accept/reply/close;
- bot silence in operator mode;
- handoff audit.

### Package 7: E2E, ops and docs

- docker compose;
- readiness;
- E2E-001..E2E-038;
- security regression;
- diagrams;
- README;
- final cleanup checklist.

## 19. Final acceptance table

| Area | Acceptance command/check | Expected result |
|---|---|---|
| Bootstrap | `docker compose up --build` | all services healthy |
| Fresh DB | run compose on empty volume | migrations + seeds complete |
| Readiness | `GET /api/v1/ready` | DB, vector, NLP, seeds ready |
| No LLM | inspect compose/env/E2E spy | no Ollama/GigaChat, no `/llm/decide` |
| Web chat | open website | session created, welcome shown |
| Semantic FAQ | ask prices/rules/location | correct intent and KB response |
| Business lookup | ask booking/payment/workspace status | provider called, mock data rendered |
| Operator | request operator | queue item created and accepted |
| Persistence | restart services | history/context restored |
| Security | run security E2E subset | Origin/XSS/raw error tests pass |
| Full E2E | run `tests/e2e` full matrix | E2E-001..E2E-038 pass |
| Docs | inspect README/diagrams | match implemented contracts |
| Cleanup | run final checklist | no legacy/conflicting artifacts |

## 20. Cutover / cleanup checklist

Финальная приемка должна включать жесткую проверку, что не осталось второго конкурирующего пути:

- нет production/runtime вызовов `/llm/decide`;
- нет обязательных Ollama/GigaChat env vars для запуска продукта;
- нет `services/llm` в `docker-compose.yml`;
- нет `chat_id=1` в production web/decision path;
- нет сырых `{service}`, `{date}`, `{status}`, `{question}` в пользовательских ответах;
- нет пустого `services/transport-adapters/website/contracts/websocket.json`;
- нет stale `transitions.json` с отсутствующими response keys;
- нет advertised action, который не зарегистрирован и не реализован;
- нет рассинхрона `escalate_operator` vs `escalate_to_operator`;
- нет inline `onclick` в quick replies;
- нет `verify=False` в runtime-коде;
- нет tracked `__pycache__`;
- нет raw internal errors в HTTP/WS responses;
- нет prompt/user text/full response logging in production mode;
- есть fresh DB bootstrap from empty database;
- есть seed loader for complete demo dataset;
- есть root `README.md`;
- есть root `docker-compose.yml`;
- есть `.env.example`;
- есть readiness endpoint, который валится без DB/vector/NLP/seed;
- есть E2E full matrix, включая business operations and operator handoff;
- есть diagrams, соответствующие фактическим contracts;
- `git status` после clean build/test не содержит generated garbage.

## Первый спринт: 5-10 первых задач

1. CB-BRD-023 - принять greenfield DB/runtime: переписать fresh migrations без legacy/backfill.
2. CB-BRD-001 - зафиксировать BRD intent catalog, response keys, quick replies и validation rules.
3. CB-BRD-024 - подготовить полный seed dataset для всех use-cases и будущих E2E.
4. CB-BRD-027 - описать external provider contracts и mock service behavior для business lookups.
5. CB-BRD-006 - убрать `chat_id=1` из web path, ввести browser/session identity.
6. CB-BRD-002 - создать `DecisionService` и заменить LLM call на injectable matcher interface.
7. CB-BRD-003 - определить/создать NLP `/embed` contract с fake implementation для тестов.
8. CB-BRD-004 + CB-BRD-005 - добавить pgvector schema и semantic matcher MVP с thresholds/fallback.
9. CB-BRD-008 + CB-BRD-010 + CB-BRD-011 - renderer, operator queue and working escalation.
10. CB-BRD-025 - завести E2E harness сразу и покрыть первые сквозные сценарии до расширения полной матрицы.
