const messagesContainer = document.getElementById('messages');
const messageForm = document.getElementById('messageForm');
const messageInput = document.getElementById('messageInput');
const sendButton = document.getElementById('sendButton');
const statusDot = document.getElementById('statusDot');
const statusText = document.getElementById('statusText');
const typingIndicator = document.getElementById('typingIndicator');

function init() {
    wsClient.onOpen(() => {
        updateConnectionStatus('connected');
        messageInput.disabled = false;
        sendButton.disabled = false;
        messageInput.focus();
        wsClient.send(wsClient.createEvent('session.start', {
            client_id: wsClient.clientId,
        }));
    });

    wsClient.onClose(() => {
        updateConnectionStatus('disconnected');
        messageInput.disabled = true;
        sendButton.disabled = true;
    });

    wsClient.onError(() => {
        updateConnectionStatus('error');
    });

    wsClient.onMessage((data) => {
        handleWebSocketMessage(data);
    });

    messageForm.addEventListener('submit', (e) => {
        e.preventDefault();
        sendMessage();
    });

    wsClient.connect();
}

function updateConnectionStatus(status) {
    statusDot.className = `status-dot ${status}`;

    switch (status) {
        case 'connected':
            statusText.textContent = 'Connected';
            break;
        case 'disconnected':
            statusText.textContent = 'Disconnected';
            break;
        case 'error':
            statusText.textContent = 'Connection Error';
            break;
        default:
            statusText.textContent = 'Connecting...';
    }
}

function handleWebSocketMessage(data) {
    hideTypingIndicator();

    switch (data.type) {
        case 'session.started':
            window.currentSessionId = data.session_id;
            window.operatorConnected = data.mode === 'operator_connected';
            break;
        case 'message.bot':
            displayBotMessage(data.text, data.quick_replies || []);
            break;
        case 'message.operator':
            displayOperatorMessage(data.text);
            break;
        case 'handoff.queued':
            displaySystemMessage('Оператор подключается. Оставайтесь в этом чате.');
            break;
        case 'handoff.accepted':
            window.operatorConnected = true;
            displaySystemMessage('Оператор подключился к диалогу.');
            break;
        case 'handoff.closed':
            window.operatorConnected = false;
            displaySystemMessage('Диалог с оператором завершен.');
            break;
        case 'error':
            displayErrorMessage((data.error && data.error.message) || 'Не удалось обработать сообщение. Попробуйте позже.');
            break;
        default:
            debugLog('Unknown message type:', data.type);
    }
}

function sendMessage() {
    const text = messageInput.value.trim();
    if (!text || !window.currentSessionId) {
        return;
    }

    displayUserMessage(text);
    messageInput.value = '';

    const sent = wsClient.send(wsClient.createEvent('message.user', {
        session_id: window.currentSessionId,
        text,
    }));

    if (!sent) {
        displayErrorMessage('Failed to send message');
        return;
    }

    if (!window.operatorConnected) {
        showTypingIndicator();
    }
}

function displayUserMessage(text) {
    appendMessage('user-message', text);
}

function displayBotMessage(text, quickReplies = []) {
    const messageDiv = document.createElement('div');
    messageDiv.className = 'message bot-message';

    const content = createMessageContent(text);
    const buttons = renderQuickReplies(quickReplies);
    if (buttons) {
        content.insertBefore(buttons, content.querySelector('.message-time'));
    }
    messageDiv.appendChild(content);

    messagesContainer.appendChild(messageDiv);
    scrollToBottom();
}

function displayOperatorMessage(text) {
    appendMessage('bot-message', `Оператор: ${text}`);
}

function displaySystemMessage(text) {
    appendMessage('bot-message', text);
}

function displayErrorMessage(text) {
    appendMessage('error-message', `❌ ${text}`);
}

function appendMessage(className, text) {
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${className}`;
    messageDiv.appendChild(createMessageContent(text));
    messagesContainer.appendChild(messageDiv);
    scrollToBottom();
}

function createMessageContent(text) {
    const content = document.createElement('div');
    content.className = 'message-content';

    const messageText = document.createElement('div');
    messageText.className = 'message-text';
    messageText.textContent = text;
    content.appendChild(messageText);

    const messageTime = document.createElement('div');
    messageTime.className = 'message-time';
    messageTime.textContent = getCurrentTime();
    content.appendChild(messageTime);

    return content;
}

function renderQuickReplies(quickReplies) {
    if (!Array.isArray(quickReplies) || quickReplies.length === 0) {
        return null;
    }

    const buttonContainer = document.createElement('div');
    buttonContainer.className = 'message-buttons';

    quickReplies.forEach((quickReply) => {
        if (!isTypedQuickReply(quickReply)) {
            return;
        }
        const button = document.createElement('button');
        button.className = 'option-button';
        button.type = 'button';
        button.textContent = quickReply.label;
        button.quickReply = cloneQuickReply(quickReply);
        buttonContainer.appendChild(button);
    });

    return buttonContainer.childElementCount > 0 ? buttonContainer : null;
}

messagesContainer.addEventListener('click', (event) => {
    const button = event.target.closest('.option-button');
    if (!button) {
        return;
    }

    const quickReply = button.quickReply;
    if (!quickReply || !window.currentSessionId) {
        return;
    }

    if (!isTypedQuickReply(quickReply)) {
        displayErrorMessage('Некорректный быстрый ответ.');
        return;
    }

    button.disabled = true;
    displayUserMessage(quickReply.label);

    const sent = wsClient.send(wsClient.createEvent('quick_reply.selected', {
        session_id: window.currentSessionId,
        quick_reply: quickReply,
    }));

    if (!sent) {
        displayErrorMessage('Failed to send message');
        return;
    }

    if (!window.operatorConnected) {
        showTypingIndicator();
    }
});

function showTypingIndicator() {
    typingIndicator.style.display = 'flex';
    scrollToBottom();
}

function hideTypingIndicator() {
    typingIndicator.style.display = 'none';
}

function scrollToBottom() {
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

function getCurrentTime() {
    const now = new Date();
    const hours = String(now.getHours()).padStart(2, '0');
    const minutes = String(now.getMinutes()).padStart(2, '0');
    return `${hours}:${minutes}`;
}

function isTypedQuickReply(quickReply) {
    return Boolean(
        quickReply &&
        typeof quickReply.id === 'string' &&
        quickReply.id.trim() &&
        typeof quickReply.label === 'string' &&
        quickReply.label.trim() &&
        typeof quickReply.action === 'string' &&
        quickReply.action.trim() &&
        (quickReply.payload === undefined || quickReply.payload === null || typeof quickReply.payload === 'object')
    );
}

function cloneQuickReply(quickReply) {
    return {
        id: quickReply.id,
        label: quickReply.label,
        action: quickReply.action,
        payload: clonePayload(quickReply.payload),
        order: quickReply.order,
    };
}

function clonePayload(payload) {
    if (!payload || typeof payload !== 'object') {
        return {};
    }
    return JSON.parse(JSON.stringify(payload));
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
