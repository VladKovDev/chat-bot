// WebSocket connection management
class WebSocketClient {
    constructor(url) {
        this.url = url;
        this.ws = null;
        this.messageHandlers = [];
        this.errorHandlers = [];
        this.closeHandlers = [];
        this.openHandlers = [];
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 2000; // 2 seconds
        this.clientId = getOrCreateClientId();
    }

    connect() {
        try {
            const separator = this.url.includes('?') ? '&' : '?';
            this.ws = new WebSocket(`${this.url}${separator}client_id=${encodeURIComponent(this.clientId)}`);

            this.ws.onopen = () => {
                debugLog('WebSocket connected');
                this.reconnectAttempts = 0;
                this.openHandlers.forEach(handler => handler());
            };

            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    debugLog('WebSocket message received:', data);
                    this.messageHandlers.forEach(handler => handler(data));
                } catch (error) {
                    debugError('Failed to parse WebSocket message');
                }
            };

            this.ws.onerror = (error) => {
                debugError('WebSocket error');
                this.errorHandlers.forEach(handler => handler(error));
            };

            this.ws.onclose = (event) => {
                debugLog('WebSocket closed:', event.code, event.reason);
                this.closeHandlers.forEach(handler => handler(event));

                // Attempt to reconnect
                if (this.reconnectAttempts < this.maxReconnectAttempts) {
                    this.reconnectAttempts++;
                    debugLog(`Reconnecting... Attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts}`);
                    setTimeout(() => this.connect(), this.reconnectDelay);
                }
            };
        } catch (error) {
            debugError('Failed to create WebSocket connection');
        }
    }

    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            const jsonMessage = JSON.stringify(message);
            debugLog('Sending WebSocket message:', { type: message.type, text_length: message.text.length });
            this.ws.send(jsonMessage);
            return true;
        } else {
            debugError('WebSocket is not connected');
            return false;
        }
    }

    onMessage(handler) {
        this.messageHandlers.push(handler);
    }

    onError(handler) {
        this.errorHandlers.push(handler);
    }

    onClose(handler) {
        this.closeHandlers.push(handler);
    }

    onOpen(handler) {
        this.openHandlers.push(handler);
    }

    close() {
        if (this.ws) {
            this.reconnectAttempts = this.maxReconnectAttempts; // Prevent reconnection
            this.ws.close();
        }
    }

    isConnected() {
        return this.ws && this.ws.readyState === WebSocket.OPEN;
    }
}

function isDebugMode() {
    const devHostnames = ['localhost', '127.0.0.1', '::1'];
    return window.localStorage.getItem('chat_bot_debug') === '1'
        || devHostnames.includes(window.location.hostname);
}

function debugLog(...args) {
    if (isDebugMode()) {
        console.log(...args);
    }
}

function debugError(...args) {
    if (isDebugMode()) {
        console.error(...args);
    }
}

function getOrCreateClientId() {
    const storageKey = 'chat_bot_client_id';
    const existing = window.localStorage.getItem(storageKey);
    if (existing) {
        return existing;
    }

    const generated = window.crypto && window.crypto.randomUUID
        ? window.crypto.randomUUID()
        : `browser-${Date.now()}-${Math.random().toString(16).slice(2)}`;
    window.localStorage.setItem(storageKey, generated);
    return generated;
}

// Create WebSocket client instance
const wsClient = new WebSocketClient(`ws://${window.location.host}/ws`);
