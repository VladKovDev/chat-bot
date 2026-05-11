(function () {
    const state = {
        operatorId: '',
        status: 'waiting',
        queue: [],
        selected: null,
        historyTimer: null,
    };

    function initOperatorConsole() {
        const operatorSurface = document.getElementById('operatorSurface');
        if (!operatorSurface) {
            return;
        }

        bindSurfaceSwitch();
        bindOperatorControls();
        refreshQueue();
    }

    function bindSurfaceSwitch() {
        document.querySelectorAll('.surface-tab').forEach((button) => {
            button.addEventListener('click', () => {
                const view = button.dataset.view;
                document.querySelectorAll('.surface-tab').forEach((tab) => {
                    tab.classList.toggle('active', tab === button);
                });
                document.getElementById('chatSurface').classList.toggle('hidden', view !== 'chat');
                document.getElementById('operatorSurface').classList.toggle('hidden', view !== 'operator');
            });
        });
    }

    function bindOperatorControls() {
        const operatorSelect = document.getElementById('operatorSelect');
        operatorSelect.addEventListener('change', () => {
            state.operatorId = operatorSelect.value;
            document.getElementById('operatorStatusText').textContent = state.operatorId
                ? `Вы: ${state.operatorId}`
                : 'Выберите оператора';
            renderSelectedSession();
        });

        document.getElementById('queueFilters').addEventListener('click', (event) => {
            const button = event.target.closest('.filter-button');
            if (!button) {
                return;
            }
            state.status = button.dataset.status;
            document.querySelectorAll('.filter-button').forEach((item) => {
                item.classList.toggle('active', item === button);
            });
            state.selected = null;
            stopHistoryPolling();
            renderSelectedSession();
            refreshQueue();
        });

        document.getElementById('acceptHandoffButton').addEventListener('click', acceptSelectedHandoff);
        document.getElementById('closeHandoffButton').addEventListener('click', closeSelectedHandoff);
        document.getElementById('operatorReplyForm').addEventListener('submit', (event) => {
            event.preventDefault();
            sendOperatorReply();
        });
    }

    async function refreshQueue() {
        try {
            const data = await fetchJSON(`/api/operator/queue?status=${encodeURIComponent(state.status)}`);
            state.queue = Array.isArray(data.items) ? data.items : [];
            renderQueue();
        } catch (error) {
            renderQueueError(error.message);
        }
    }

    function renderQueue() {
        const queue = document.getElementById('operatorQueue');
        clearElement(queue);

        if (state.queue.length === 0) {
            queue.appendChild(createQueueEmptyState('Пусто', 'По этому фильтру пусто'));
            return;
        }

        state.queue.forEach((item) => {
            const button = document.createElement('button');
            button.type = 'button';
            button.className = `queue-item${state.selected && state.selected.handoff_id === item.handoff_id ? ' active' : ''}`;
            button.appendChild(createTextElement('strong', '', item.preview || item.reason || item.session_id));
            button.appendChild(createTextElement('span', '', `${localizeStatus(item.status || state.status)} - ${item.last_intent || 'нет'}`));
            button.appendChild(createTextElement('span', '', item.session_id));
            button.addEventListener('click', () => selectQueueItem(item));
            queue.appendChild(button);
        });
    }

    function renderQueueError(message) {
        const queue = document.getElementById('operatorQueue');
        clearElement(queue);
        queue.appendChild(createQueueEmptyState('Очередь недоступна', message));
    }

    function selectQueueItem(item) {
        state.selected = item;
        renderQueue();
        renderSelectedSession();
        loadSessionHistory();
        startHistoryPolling();
    }

    function renderSelectedSession() {
        const item = state.selected;
        document.getElementById('operatorSessionTitle').textContent = item
            ? `Диалог ${shortId(item.session_id)}`
            : 'Ничего не выбрано';
        document.getElementById('operatorSessionMeta').textContent = item
            ? `${item.reason || 'Диалог'} - ${localizeStatus(item.status || state.status)}`
            : 'Выберите диалог';

        document.getElementById('contextTopic').textContent = item && item.active_topic ? item.active_topic : '-';
        document.getElementById('contextIntent').textContent = item && item.last_intent ? item.last_intent : '-';
        document.getElementById('contextConfidence').textContent = formatConfidence(item && item.confidence);
        document.getElementById('contextFallbacks').textContent = String((item && item.fallback_count) || 0);
        renderActionSummaries((item && item.action_summaries) || []);

        document.getElementById('acceptHandoffButton').disabled = !canAccept(state);
        document.getElementById('closeHandoffButton').disabled = !canClose(state);
        document.getElementById('operatorReplyInput').disabled = !canReply(state);
        document.getElementById('operatorReplyButton').disabled = !canReply(state);
    }

    function renderActionSummaries(items) {
        const list = document.getElementById('actionSummaries');
        clearElement(list);
        if (!Array.isArray(items) || items.length === 0) {
            list.appendChild(createTextElement('div', 'action-summary', 'Действий нет'));
            return;
        }
        items.forEach((item) => {
            const summary = document.createElement('div');
            summary.className = 'action-summary';
            summary.appendChild(createTextElement('strong', '', item.action_type || 'Шаг'));
            summary.appendChild(createTextElement('div', '', item.status || 'неизв.'));
            if (item.summary) {
                summary.appendChild(createTextElement('div', '', item.summary));
            }
            list.appendChild(summary);
        });
    }

    async function loadSessionHistory() {
        if (!state.selected) {
            renderHistory([]);
            return;
        }
        try {
            const data = await fetchJSON(`/api/operator/sessions/${encodeURIComponent(state.selected.session_id)}/messages`);
            renderHistory(Array.isArray(data.items) ? data.items : []);
        } catch (error) {
            renderHistoryError(error.message);
        }
    }

    function renderHistory(items) {
        const history = document.getElementById('operatorHistory');
        clearElement(history);
        if (!Array.isArray(items) || items.length === 0) {
            history.appendChild(createHistoryMessage('history', 'История', 'Сообщений нет'));
            return;
        }
        items.forEach((item) => {
            history.appendChild(createHistoryMessage(
                item.sender_type || 'bot',
                `${localizeSender(item.sender_type)} - ${formatTime(item.timestamp)}`,
                item.text || '',
            ));
        });
        history.scrollTop = history.scrollHeight;
    }

    function renderHistoryError(message) {
        const history = document.getElementById('operatorHistory');
        clearElement(history);
        history.appendChild(createHistoryMessage('error', 'Ошибка', message));
    }

    function startHistoryPolling() {
        stopHistoryPolling();
        state.historyTimer = setInterval(loadSessionHistory, 2000);
    }

    function stopHistoryPolling() {
        if (state.historyTimer) {
            clearInterval(state.historyTimer);
            state.historyTimer = null;
        }
    }

    async function acceptSelectedHandoff() {
        if (!canAccept(state)) {
            return;
        }
        const data = await fetchJSON(`/api/operator/queue/${encodeURIComponent(state.selected.handoff_id)}/accept`, {
            method: 'POST',
            body: JSON.stringify({ operator_id: state.operatorId }),
        });
        state.selected = {
            ...state.selected,
            status: data.handoff.status,
            operator_id: data.handoff.operator_id || state.operatorId,
        };
        renderSelectedSession();
        refreshQueue();
    }

    async function closeSelectedHandoff() {
        if (!canClose(state)) {
            return;
        }
        const data = await fetchJSON(`/api/operator/queue/${encodeURIComponent(state.selected.handoff_id)}/close`, {
            method: 'POST',
            body: JSON.stringify({ operator_id: state.operatorId }),
        });
        state.selected = {
            ...state.selected,
            status: data.handoff.status,
            operator_id: data.handoff.operator_id || state.operatorId,
        };
        renderSelectedSession();
        refreshQueue();
    }

    async function sendOperatorReply() {
        if (!canReply(state)) {
            return;
        }
        const input = document.getElementById('operatorReplyInput');
        const text = input.value.trim();
        if (!text) {
            return;
        }
        input.value = '';
        await fetchJSON(`/api/operator/sessions/${encodeURIComponent(state.selected.session_id)}/messages`, {
            method: 'POST',
            body: JSON.stringify({ operator_id: state.operatorId, text }),
        });
        loadSessionHistory();
    }

    async function fetchJSON(path, options = {}) {
        const response = await fetch(path, {
            headers: { 'Content-Type': 'application/json' },
            ...options,
        });
        const data = await response.json();
        if (!response.ok) {
            const message = data && data.error && data.error.message ? data.error.message : `HTTP ${response.status}`;
            throw new Error(message);
        }
        return data;
    }

    function canAccept(current) {
        return Boolean(current.operatorId && current.selected && current.selected.status === 'waiting');
    }

    function canClose(current) {
        return Boolean(current.operatorId && current.selected && ['waiting', 'accepted'].includes(current.selected.status));
    }

    function canReply(current) {
        return Boolean(current.operatorId && current.selected && current.selected.status === 'accepted');
    }

    function formatConfidence(value) {
        return typeof value === 'number' ? `${Math.round(value * 100)}%` : '-';
    }

    function localizeStatus(value) {
        switch (value) {
            case 'waiting':
                return 'Новые';
            case 'accepted':
                return 'В работе';
            case 'closed':
                return 'Закрыты';
            default:
                return value || '-';
        }
    }

    function localizeSender(value) {
        switch (value) {
            case 'user':
                return 'Клиент';
            case 'bot':
                return 'Бот';
            case 'operator':
                return 'Оператор';
            case 'system':
                return 'Система';
            default:
                return value || 'Сообщение';
        }
    }

    function formatTime(value) {
        if (!value) {
            return '';
        }
        const parsed = new Date(value);
        if (Number.isNaN(parsed.getTime())) {
            return value;
        }
        return parsed.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }

    function shortId(value) {
        return value ? value.slice(0, 8) : '-';
    }

    function clearElement(element) {
        while (element.firstChild) {
            element.removeChild(element.firstChild);
        }
    }

    function createTextElement(tagName, className, text) {
        const element = document.createElement(tagName);
        if (className) {
            element.className = className;
        }
        element.textContent = text == null ? '' : String(text);
        return element;
    }

    function createQueueEmptyState(title, detail) {
        const item = document.createElement('div');
        item.className = 'queue-item';
        item.appendChild(createTextElement('strong', '', title));
        item.appendChild(createTextElement('span', '', detail));
        return item;
    }

    function createHistoryMessage(senderType, meta, text) {
        const item = document.createElement('div');
        item.className = `history-message ${toClassToken(senderType, 'message')}`;
        item.appendChild(createTextElement('div', 'meta', meta));
        item.appendChild(createTextElement('div', '', text));
        return item;
    }

    function toClassToken(value, fallback) {
        const token = String(value || '')
            .toLowerCase()
            .replace(/[^a-z0-9_-]/g, '-')
            .replace(/-+/g, '-')
            .replace(/^-|-$/g, '');
        return token || fallback;
    }

    window.OperatorConsole = {
        canAccept,
        canClose,
        canReply,
        formatConfidence,
        shortId,
    };

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initOperatorConsole);
    } else {
        initOperatorConsole();
    }
}());
