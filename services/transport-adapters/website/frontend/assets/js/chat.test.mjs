import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';

const scriptPath = path.resolve('services/transport-adapters/website/frontend/assets/js/chat.js');
const scriptSource = fs.readFileSync(scriptPath, 'utf8');

test('quick reply buttons render malicious labels as text and send typed selection', () => {
    const { elements, wsClient, sandbox } = loadChat();
    const maliciousLabel = '<img src=x onerror="globalThis.__xss = true">Связаться';
    const quickReply = {
        id: 'dangerous-label',
        label: maliciousLabel,
        action: 'select_intent',
        payload: {
            intent: 'return_to_menu',
            text: 'главное меню',
        },
        order: 7,
    };

    wsClient.handlers.message({
        type: 'session.started',
        session_id: '11111111-1111-1111-1111-111111111111',
        mode: 'standard',
    });
    wsClient.handlers.message({
        type: 'message.bot',
        text: 'Выберите действие',
        quick_replies: [quickReply],
    });

    const button = findByClass(elements.messages, 'option-button');
    assert.ok(button, 'quick reply button should be rendered');
    assert.equal(button.textContent, maliciousLabel);
    assert.equal(sandbox.__xss, undefined);

    elements.messages.dispatchEvent({ type: 'click', target: button });

    const sent = wsClient.sent.at(-1);
    assert.equal(sent.type, 'quick_reply.selected');
    assert.equal(sent.session_id, '11111111-1111-1111-1111-111111111111');
    assert.equal(sent.quick_reply.id, quickReply.id);
    assert.equal(sent.quick_reply.label, quickReply.label);
    assert.equal(sent.quick_reply.action, quickReply.action);
    assert.equal(sent.quick_reply.payload.intent, quickReply.payload.intent);
    assert.equal(sent.quick_reply.payload.text, quickReply.payload.text);
    assert.equal(sent.quick_reply.order, quickReply.order);
    assert.equal(sent.text, undefined);
    assert.equal(sandbox.__xss, undefined);
}
);

test('chat renderer inserts user, bot, and operator text with quotes and HTML as text nodes', () => {
    const { elements, wsClient, sandbox } = loadChat();
    const maliciousUserText = 'hello "quoted" <img src=x onerror="globalThis.__xss = true">'.repeat(10);
    const maliciousBotText = '<script>globalThis.__xss = true</script> bot answer';
    const maliciousOperatorText = '<button onclick="globalThis.__xss = true">operator</button> says hi';

    wsClient.handlers.message({
        type: 'session.started',
        session_id: '22222222-2222-2222-2222-222222222222',
        mode: 'standard',
    });
    elements.messageInput.value = maliciousUserText;
    elements.messageForm.dispatchEvent({
        type: 'submit',
        preventDefault() {},
    });
    wsClient.handlers.message({
        type: 'message.bot',
        text: maliciousBotText,
        quick_replies: [],
    });
    wsClient.handlers.message({
        type: 'message.operator',
        text: maliciousOperatorText,
    });

    assert.equal(elements.messages.children[0].children[0].children[0].textContent, maliciousUserText);
    assert.equal(elements.messages.children[1].children[0].children[0].textContent, maliciousBotText);
    assert.equal(elements.messages.children[2].children[0].children[0].textContent, `Оператор: ${maliciousOperatorText}`);
    assert.equal(wsClient.sent.at(-1).text, maliciousUserText);
    assert.equal(countDescendantsByTag(elements.messages, 'IMG'), 0);
    assert.equal(countDescendantsByTag(elements.messages, 'SCRIPT'), 0);
    assert.equal(countDescendantsByTag(elements.messages, 'BUTTON'), 0);
    assert.equal(sandbox.__xss, undefined);
});

test('chat renderer does not serialize quick replies into HTML attributes', () => {
    assert.equal(scriptSource.includes('data-quick-reply'), false);
    assert.equal(scriptSource.includes('onclick'), false);
    assert.equal(scriptSource.includes('innerHTML'), false);
    assert.equal(scriptSource.includes('insertAdjacentHTML'), false);
    assert.equal(scriptSource.includes('outerHTML'), false);
});

function loadChat() {
    const elements = {
        messages: new FakeElement('div'),
        messageForm: new FakeElement('form'),
        messageInput: new FakeElement('input'),
        sendButton: new FakeElement('button'),
        statusDot: new FakeElement('span'),
        statusText: new FakeElement('span'),
        typingIndicator: new FakeElement('div'),
    };
    elements.messages.id = 'messages';
    elements.messageForm.id = 'messageForm';
    elements.messageInput.id = 'messageInput';
    elements.sendButton.id = 'sendButton';
    elements.statusDot.id = 'statusDot';
    elements.statusText.id = 'statusText';
    elements.typingIndicator.id = 'typingIndicator';

    const wsClient = {
        clientId: 'browser-a',
        handlers: {},
        sent: [],
        onOpen(callback) {
            this.handlers.open = callback;
        },
        onClose(callback) {
            this.handlers.close = callback;
        },
        onError(callback) {
            this.handlers.error = callback;
        },
        onMessage(callback) {
            this.handlers.message = callback;
        },
        connect() {},
        createEvent(type, payload) {
            return { type, ...payload };
        },
        send(event) {
            this.sent.push(event);
            return true;
        },
    };

    const document = {
        readyState: 'complete',
        getElementById(id) {
            return elements[id];
        },
        createElement(tagName) {
            return new FakeElement(tagName);
        },
        addEventListener() {},
    };

    const sandbox = {
        console,
        Date: FixedDate,
        window: {},
        document,
        wsClient,
    };
    sandbox.globalThis = sandbox;
    vm.runInNewContext(scriptSource, sandbox, { filename: scriptPath });
    return { elements, wsClient, sandbox };
}

class FakeElement {
    constructor(tagName) {
        this.tagName = tagName.toUpperCase();
        this.children = [];
        this.parentNode = null;
        this.listeners = {};
        this.style = {};
        this.disabled = false;
        this.value = '';
        this.scrollTop = 0;
        this.scrollHeight = 0;
        this._className = '';
        this.textContent = '';
    }

    set className(value) {
        this._className = value;
    }

    get className() {
        return this._className;
    }

    get childElementCount() {
        return this.children.length;
    }

    appendChild(child) {
        child.parentNode = this;
        this.children.push(child);
        return child;
    }

    insertBefore(child, reference) {
        child.parentNode = this;
        const index = this.children.indexOf(reference);
        if (index === -1) {
            this.children.push(child);
        } else {
            this.children.splice(index, 0, child);
        }
        return child;
    }

    querySelector(selector) {
        if (!selector.startsWith('.')) {
            return null;
        }
        return findByClass(this, selector.slice(1));
    }

    addEventListener(type, callback) {
        this.listeners[type] = callback;
    }

    dispatchEvent(event) {
        this.listeners[event.type]?.(event);
    }

    closest(selector) {
        if (!selector.startsWith('.')) {
            return null;
        }
        const className = selector.slice(1);
        for (let node = this; node; node = node.parentNode) {
            if (node.className.split(/\s+/).includes(className)) {
                return node;
            }
        }
        return null;
    }

    focus() {}

    set innerHTML(_value) {
        throw new Error('innerHTML must not be used in chat renderer tests');
    }

    get innerHTML() {
        throw new Error('innerHTML must not be read in chat renderer tests');
    }
}

function findByClass(node, className) {
    if (node.className.split(/\s+/).includes(className)) {
        return node;
    }
    for (const child of node.children) {
        const match = findByClass(child, className);
        if (match) {
            return match;
        }
    }
    return null;
}

function countDescendantsByTag(node, tagName) {
    let count = node.tagName === tagName ? 1 : 0;
    for (const child of node.children) {
        count += countDescendantsByTag(child, tagName);
    }
    return count;
}

function FixedDate() {
    return new Date('2026-05-10T12:34:00Z');
}

FixedDate.now = Date.now;
FixedDate.UTC = Date.UTC;
FixedDate.parse = Date.parse;
