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
                console.log('WebSocket connected');
                this.reconnectAttempts = 0;
                this.openHandlers.forEach(handler => handler());
            };

            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    console.log('WebSocket message received:', data);
                    this.messageHandlers.forEach(handler => handler(data));
                } catch (error) {
                    console.error('Failed to parse WebSocket message:', error);
                }
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.errorHandlers.forEach(handler => handler(error));
            };

            this.ws.onclose = (event) => {
                console.log('WebSocket closed:', event.code, event.reason);
                this.closeHandlers.forEach(handler => handler(event));

                // Attempt to reconnect
                if (this.reconnectAttempts < this.maxReconnectAttempts) {
                    this.reconnectAttempts++;
                    console.log(`Reconnecting... Attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts}`);
                    setTimeout(() => this.connect(), this.reconnectDelay);
                }
            };
        } catch (error) {
            console.error('Failed to create WebSocket connection:', error);
        }
    }

    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            const jsonMessage = JSON.stringify(message);
            console.log('Sending WebSocket message:', jsonMessage);
            this.ws.send(jsonMessage);
            return true;
        } else {
            console.error('WebSocket is not connected');
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
