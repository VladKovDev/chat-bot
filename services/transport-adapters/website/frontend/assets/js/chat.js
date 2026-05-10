// DOM elements
const messagesContainer = document.getElementById('messages');
const messageForm = document.getElementById('messageForm');
const messageInput = document.getElementById('messageInput');
const sendButton = document.getElementById('sendButton');
const statusDot = document.getElementById('statusDot');
const statusText = document.getElementById('statusText');
const typingIndicator = document.getElementById('typingIndicator');

// Initialize chat
function init() {
    // Set up WebSocket event handlers
    wsClient.onOpen(() => {
        updateConnectionStatus('connected');
        messageInput.disabled = false;
        sendButton.disabled = false;
        messageInput.focus();
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

    // Set up form submit handler
    messageForm.addEventListener('submit', (e) => {
        e.preventDefault();
        sendMessage();
    });

    // Connect WebSocket
    wsClient.connect();
}

// Update connection status UI
function updateConnectionStatus(status) {
    statusDot.className = 'status-dot ' + status;

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

// Handle WebSocket message
function handleWebSocketMessage(data) {
    hideTypingIndicator();

    switch (data.type) {
        case 'session':
            window.currentSessionId = data.session_id;
            break;
        case 'response':
            displayBotMessage(data.text, data.options);
            break;
        case 'error':
            displayErrorMessage(data.text);
            break;
        default:
            console.warn('Unknown message type:', data.type);
    }
}

// Send user message
function sendMessage() {
    const text = messageInput.value.trim();
    if (!text) return;

    // Display user message
    displayUserMessage(text);
    messageInput.value = '';

    // Send to server
    const message = {
        type: 'message',
        text: text
    };

    if (!wsClient.send(message)) {
        displayErrorMessage('Failed to send message');
    } else {
        showTypingIndicator();
    }
}

// Display user message
function displayUserMessage(text) {
    const messageDiv = document.createElement('div');
    messageDiv.className = 'message user-message';
    messageDiv.innerHTML = `
        <div class="message-content">
            <div class="message-text">${escapeHtml(text)}</div>
            <div class="message-time">${getCurrentTime()}</div>
        </div>
    `;
    messagesContainer.appendChild(messageDiv);
    scrollToBottom();
}

// Display bot message
function displayBotMessage(text, options = []) {
    const messageDiv = document.createElement('div');
    messageDiv.className = 'message bot-message';

    let buttonsHtml = '';
    if (options && options.length > 0) {
        buttonsHtml = '<div class="message-buttons">';
        options.forEach(option => {
            buttonsHtml += `<button class="option-button" onclick="sendOption('${escapeHtml(option)}')">${escapeHtml(option)}</button>`;
        });
        buttonsHtml += '</div>';
    }

    messageDiv.innerHTML = `
        <div class="message-content">
            <div class="message-text">${escapeHtml(text)}</div>
            ${buttonsHtml}
            <div class="message-time">${getCurrentTime()}</div>
        </div>
    `;
    messagesContainer.appendChild(messageDiv);
    scrollToBottom();
}

// Display error message
function displayErrorMessage(text) {
    const messageDiv = document.createElement('div');
    messageDiv.className = 'message error-message';
    messageDiv.innerHTML = `
        <div class="message-content">
            <div class="message-text">❌ ${escapeHtml(text)}</div>
            <div class="message-time">${getCurrentTime()}</div>
        </div>
    `;
    messagesContainer.appendChild(messageDiv);
    scrollToBottom();
}

// Send option button text as message
function sendOption(text) {
    messageInput.value = text;
    sendMessage();

    // Disable all option buttons after selection
    const buttons = document.querySelectorAll('.option-button');
    buttons.forEach(button => {
        button.disabled = true;
    });
}

// Show typing indicator
function showTypingIndicator() {
    typingIndicator.style.display = 'flex';
    scrollToBottom();
}

// Hide typing indicator
function hideTypingIndicator() {
    typingIndicator.style.display = 'none';
}

// Scroll to bottom of messages
function scrollToBottom() {
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

// Get current time in HH:MM format
function getCurrentTime() {
    const now = new Date();
    const hours = String(now.getHours()).padStart(2, '0');
    const minutes = String(now.getMinutes()).padStart(2, '0');
    return `${hours}:${minutes}`;
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Initialize chat when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
