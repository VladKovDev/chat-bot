# Console Chat Adapter

Простой консольный чат-адаптер для тестирования decision-engine сервиса.

## Установка

```bash
pip install -r requirements.txt
```

## Использование

Запустите адаптер:

```bash
python main.py
```

### Команды

- `quit` или `exit` - выйти из чата
- `chat <id>` - сменить ID чата (например: `chat 42`)

## Пример работы

```
=== Console Chat Adapter ===
Connected to: http://localhost:8080/decide
Type 'quit' or 'exit' to stop the chat
Type 'chat <id>' to change chat ID
----------------------------------------
[Chat #1] You: Привет

🤖 Bot: Здравствуйте! Чем могу помочь?
   State: greeting

[Chat #1] You: chat 5
✓ Chat ID changed to 5

[Chat #5] You: У меня проблема

🤖 Bot: Опишите вашу проблему подробнее
   State: understanding
```

## API

Запрос к `http://localhost:8080/decide`:

**Request:**
```json
{
  "text": "message text",
  "channel": "dev-cli",
  "chat_id": 1
}
```

**Response:**
```json
{
  "text": "response text",
  "state": "current_state",
  "chat_id": 1,
  "success": true
}
```
