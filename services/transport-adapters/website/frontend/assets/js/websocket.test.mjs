import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';

const scriptPath = path.resolve('services/transport-adapters/website/frontend/assets/js/websocket.js');
const scriptSource = fs.readFileSync(scriptPath, 'utf8');

test('buildWebSocketURL chooses wss for https pages', () => {
    const api = loadWebSocketAPI({
        protocol: 'https:',
        host: 'chat.example.test',
        hostname: 'chat.example.test',
    });

    assert.equal(
        api.buildWebSocketURL({ protocol: 'https:', host: 'chat.example.test' }),
        'wss://chat.example.test/ws',
    );
});

test('buildWebSocketURL chooses ws for http pages', () => {
    const api = loadWebSocketAPI({
        protocol: 'http:',
        host: 'localhost:8081',
        hostname: 'localhost',
    });

    assert.equal(
        api.buildWebSocketURL({ protocol: 'http:', host: 'localhost:8081' }),
        'ws://localhost:8081/ws',
    );
});

function loadWebSocketAPI(location) {
    const storage = new Map();

    class FakeWebSocket {
        static OPEN = 1;

        constructor(url) {
            this.url = url;
            this.readyState = FakeWebSocket.OPEN;
        }

        send() {}

        close() {}
    }

    const sandbox = {
        console,
        Date,
        Math,
        Set,
        Map,
        JSON,
        encodeURIComponent,
        decodeURIComponent,
        setTimeout,
        clearTimeout,
        WebSocket: FakeWebSocket,
        window: {
            location,
            localStorage: {
                getItem(key) {
                    return storage.has(key) ? storage.get(key) : null;
                },
                setItem(key, value) {
                    storage.set(key, value);
                },
            },
            crypto: {
                randomUUID() {
                    return '00000000-0000-4000-8000-000000000000';
                },
            },
        },
    };
    sandbox.globalThis = sandbox;

    vm.runInNewContext(scriptSource, sandbox, { filename: scriptPath });

    return sandbox.window.ChatBotWebSocket;
}
