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
            displaySystemMessage('Оператор подключился к диалогу.');
            break;
        case 'handoff.closed':
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

    showTypingIndicator();
}

function displayUserMessage(text) {
    appendMessage('user-message', text);
}

function displayBotMessage(text, quickReplies = []) {
    const messageDiv = document.createElement('div');
    messageDiv.className = 'message bot-message';

    const buttons = renderQuickReplies(quickReplies);
    messageDiv.innerHTML = `
        <div class="message-content">
            <div class="message-text">${escapeHtml(text)}</div>
            ${buttons}
            <div class="message-time">${getCurrentTime()}</div>
        </div>
    `;

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
    messageDiv.innerHTML = `
        <div class="message-content">
            <div class="message-text">${escapeHtml(text)}</div>
            <div class="message-time">${getCurrentTime()}</div>
        </div>
    `;
    messagesContainer.appendChild(messageDiv);
    scrollToBottom();
}

function renderQuickReplies(quickReplies) {
    if (!Array.isArray(quickReplies) || quickReplies.length === 0) {
        return '';
    }

    const buttons = quickReplies.map((quickReply) => {
        const encoded = encodeURIComponent(JSON.stringify(quickReply));
        return `<button class="option-button" data-quick-reply="${encoded}">${escapeHtml(quickReply.label)}</button>`;
    }).join('');

    return `<div class="message-buttons">${buttons}</div>`;
}

messagesContainer.addEventListener('click', (event) => {
    const button = event.target.closest('.option-button');
    if (!button) {
        return;
    }

    const rawQuickReply = button.getAttribute('data-quick-reply');
    if (!rawQuickReply || !window.currentSessionId) {
        return;
    }

    let quickReply;
    try {
        quickReply = JSON.parse(decodeURIComponent(rawQuickReply));
    } catch (_error) {
        displayErrorMessage('Не удалось обработать быстрый ответ.');
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

    showTypingIndicator();
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

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
