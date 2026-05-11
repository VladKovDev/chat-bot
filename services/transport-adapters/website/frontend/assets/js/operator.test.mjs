import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import vm from 'node:vm';

const scriptPath = path.join(path.dirname(fileURLToPath(import.meta.url)), 'operator.js');
const scriptSource = fs.readFileSync(scriptPath, 'utf8');

test('operator console enables actions for selected demo operator state', () => {
    const api = loadOperatorAPI();

    assert.equal(api.canAccept({
        operatorId: 'operator-1',
        selected: { status: 'waiting' },
    }), true);
    assert.equal(api.canReply({
        operatorId: 'operator-1',
        selected: { status: 'accepted' },
    }), true);
    assert.equal(api.canClose({
        operatorId: 'operator-1',
        selected: { status: 'closed' },
    }), false);
});

test('operator console formats handoff context values', () => {
    const api = loadOperatorAPI();

    assert.equal(api.formatConfidence(0.734), '73%');
    assert.equal(api.formatConfidence(null), '-');
    assert.equal(api.shortId('11111111-2222-3333-4444-555555555555'), '11111111');
});

test('operator console renders dynamic queue, context, and history values as text', async () => {
    const maliciousPreview = '<img src=x onerror="globalThis.__xss = true"> "quoted" '.repeat(20);
    const maliciousIntent = 'intent"><script>globalThis.__xss = true</script>';
    const maliciousSummary = '<svg onload="globalThis.__xss = true">operator summary</svg>';
    const maliciousHistory = 'operator says <button onclick="globalThis.__xss = true">pay</button> and "quote"';
    const document = createOperatorDocument();
    const fetchCalls = [];
    const sandbox = {
        console,
        Date,
        Error,
        JSON,
        Math,
        Number,
        String,
        encodeURIComponent,
        setInterval() {
            return 1;
        },
        clearInterval() {},
        async fetch(path) {
            fetchCalls.push(path);
            if (path.startsWith('/api/operator/queue?')) {
                return createJSONResponse({
                    items: [{
                        handoff_id: 'handoff-1',
                        session_id: 'session-1',
                        status: 'waiting',
                        reason: '<script>reason()</script>',
                        preview: maliciousPreview,
                        last_intent: maliciousIntent,
                        active_topic: '<iframe srcdoc="<script>globalThis.__xss = true</script>"></iframe>',
                        confidence: 0.42,
                        fallback_count: 3,
                        action_summaries: [{
                            action_type: '<img src=x onerror="globalThis.__xss = true">',
                            status: '"done" <script>bad()</script>',
                            summary: maliciousSummary,
                        }],
                    }],
                });
            }
            if (path === '/api/operator/sessions/session-1/messages') {
                return createJSONResponse({
                    items: [{
                        sender_type: 'operator" onclick="globalThis.__xss = true',
                        timestamp: '2026-05-10T12:34:00Z',
                        text: maliciousHistory,
                    }],
                });
            }
            throw new Error(`unexpected fetch path: ${path}`);
        },
        document,
        window: {},
    };
    sandbox.globalThis = sandbox;

    vm.runInNewContext(scriptSource, sandbox, { filename: scriptPath });
    await flushAsync();

    const queueButton = document.elements.operatorQueue.children[0];
    assert.equal(queueButton.children[0].textContent, maliciousPreview);
    assert.equal(queueButton.children[1].textContent, `Новые - ${maliciousIntent}`);
    assert.equal(queueButton.children[2].textContent, 'session-1');
    assert.equal(countDescendantsByTag(document.elements.operatorQueue, 'IMG'), 0);

    queueButton.dispatchEvent({ type: 'click', target: queueButton });
    await flushAsync();

    assert.equal(document.elements.contextTopic.textContent, '<iframe srcdoc="<script>globalThis.__xss = true</script>"></iframe>');
    assert.equal(document.elements.contextIntent.textContent, maliciousIntent);
    assert.equal(document.elements.actionSummaries.children[0].children[0].textContent, '<img src=x onerror="globalThis.__xss = true">');
    assert.equal(document.elements.actionSummaries.children[0].children[2].textContent, maliciousSummary);
    assert.equal(document.elements.operatorHistory.children[0].children[1].textContent, maliciousHistory);
    assert.equal(countDescendantsByTag(document.elements.operatorHistory, 'BUTTON'), 0);
    assert.equal(sandbox.__xss, undefined);
    assert.deepEqual(fetchCalls, [
        '/api/operator/queue?status=waiting',
        '/api/operator/sessions/session-1/messages',
    ]);
});

test('operator console frontend source does not use dynamic HTML sinks or inline handlers', () => {
    assert.equal(scriptSource.includes('innerHTML'), false);
    assert.equal(scriptSource.includes('insertAdjacentHTML'), false);
    assert.equal(scriptSource.includes('outerHTML'), false);
    assert.equal(scriptSource.includes('onclick'), false);
});

function loadOperatorAPI() {
    const sandbox = {
        console,
        Date,
        Error,
        JSON,
        Math,
        Number,
        String,
        encodeURIComponent,
        setInterval() {
            return 1;
        },
        clearInterval() {},
        fetch() {
            throw new Error('fetch should not run during helper tests');
        },
        document: {
            readyState: 'loading',
            addEventListener() {},
        },
        window: {},
    };
    sandbox.globalThis = sandbox;

    vm.runInNewContext(scriptSource, sandbox, { filename: scriptPath });

    return sandbox.window.OperatorConsole;
}

function createOperatorDocument() {
    const elements = {
        operatorSurface: new FakeElement('section'),
        chatSurface: new FakeElement('div'),
        operatorSelect: new FakeElement('select'),
        operatorStatusText: new FakeElement('p'),
        queueFilters: new FakeElement('div'),
        operatorQueue: new FakeElement('div'),
        acceptHandoffButton: new FakeElement('button'),
        closeHandoffButton: new FakeElement('button'),
        operatorReplyForm: new FakeElement('form'),
        operatorSessionTitle: new FakeElement('h2'),
        operatorSessionMeta: new FakeElement('p'),
        contextTopic: new FakeElement('dd'),
        contextIntent: new FakeElement('dd'),
        contextConfidence: new FakeElement('dd'),
        contextFallbacks: new FakeElement('dd'),
        actionSummaries: new FakeElement('div'),
        operatorHistory: new FakeElement('section'),
        operatorReplyInput: new FakeElement('input'),
        operatorReplyButton: new FakeElement('button'),
    };
    Object.entries(elements).forEach(([id, element]) => {
        element.id = id;
    });

    const surfaceTabs = [
        createButtonWithClass('surface-tab active', { view: 'chat' }),
        createButtonWithClass('surface-tab', { view: 'operator' }),
    ];
    const filterButtons = [
        createButtonWithClass('filter-button active', { status: 'waiting' }),
        createButtonWithClass('filter-button', { status: 'accepted' }),
        createButtonWithClass('filter-button', { status: 'closed' }),
    ];
    filterButtons.forEach((button) => elements.queueFilters.appendChild(button));

    return {
        elements,
        readyState: 'complete',
        getElementById(id) {
            return elements[id];
        },
        createElement(tagName) {
            return new FakeElement(tagName);
        },
        querySelectorAll(selector) {
            if (selector === '.surface-tab') {
                return surfaceTabs;
            }
            if (selector === '.filter-button') {
                return filterButtons;
            }
            return [];
        },
        addEventListener() {},
    };
}

function createButtonWithClass(className, dataset) {
    const button = new FakeElement('button');
    button.className = className;
    button.dataset = dataset;
    return button;
}

function createJSONResponse(data, ok = true, status = 200) {
    return {
        ok,
        status,
        async json() {
            return data;
        },
    };
}

async function flushAsync() {
    await Promise.resolve();
    await Promise.resolve();
    await new Promise((resolve) => setImmediate(resolve));
}

class FakeElement {
    constructor(tagName) {
        this.tagName = tagName.toUpperCase();
        this.children = [];
        this.parentNode = null;
        this.listeners = {};
        this.dataset = {};
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

    get classList() {
        return {
            toggle: (className, force) => {
                const classes = new Set(this.className.split(/\s+/).filter(Boolean));
                if (force) {
                    classes.add(className);
                } else {
                    classes.delete(className);
                }
                this.className = [...classes].join(' ');
            },
        };
    }

    get firstChild() {
        return this.children[0] || null;
    }

    appendChild(child) {
        child.parentNode = this;
        this.children.push(child);
        return child;
    }

    removeChild(child) {
        const index = this.children.indexOf(child);
        if (index !== -1) {
            this.children.splice(index, 1);
            child.parentNode = null;
        }
        return child;
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

    set innerHTML(_value) {
        throw new Error('innerHTML must not be used in operator renderer tests');
    }

    get innerHTML() {
        throw new Error('innerHTML must not be read in operator renderer tests');
    }
}

function countDescendantsByTag(node, tagName) {
    let count = node.tagName === tagName ? 1 : 0;
    for (const child of node.children) {
        count += countDescendantsByTag(child, tagName);
    }
    return count;
}
