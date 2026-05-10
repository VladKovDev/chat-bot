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
        this.reconnectDelay = 2000;
        this.clientId = getOrCreateClientId();
    }

    connect() {
        try {
            this.ws = new WebSocket(this.url);

            this.ws.onopen = () => {
                debugLog('WebSocket connected');
                this.reconnectAttempts = 0;
                this.openHandlers.forEach((handler) => handler());
            };

            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    debugLog('WebSocket message received:', data);
                    this.messageHandlers.forEach((handler) => handler(data));
                } catch (_error) {
                    debugError('Failed to parse WebSocket message');
                }
            };

            this.ws.onerror = (error) => {
                debugError('WebSocket error');
                this.errorHandlers.forEach((handler) => handler(error));
            };

            this.ws.onclose = (event) => {
                debugLog('WebSocket closed:', event.code, event.reason);
                this.closeHandlers.forEach((handler) => handler(event));

                if (this.reconnectAttempts < this.maxReconnectAttempts) {
                    this.reconnectAttempts += 1;
                    debugLog(`Reconnecting... Attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts}`);
                    setTimeout(() => this.connect(), this.reconnectDelay);
                }
            };
        } catch (_error) {
            debugError('Failed to create WebSocket connection');
        }
    }

    send(message) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            debugError('WebSocket is not connected');
            return false;
        }

        const jsonMessage = JSON.stringify(message);
        debugLog('Sending WebSocket message:', {
            type: message.type,
            session_id: message.session_id,
        });
        this.ws.send(jsonMessage);
        return true;
    }

    createEvent(type, payload = {}) {
        const eventId = createEventId();
        return {
            type,
            event_id: eventId,
            correlation_id: eventId,
            timestamp: new Date().toISOString(),
            ...payload,
        };
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
            this.reconnectAttempts = this.maxReconnectAttempts;
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

    const generated = createEventId();
    window.localStorage.setItem(storageKey, generated);
    return generated;
}

function createEventId() {
    if (window.crypto && window.crypto.randomUUID) {
        return window.crypto.randomUUID();
    }
    return `browser-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
const wsClient = new WebSocketClient(`${protocol}://${window.location.host}/ws`);
