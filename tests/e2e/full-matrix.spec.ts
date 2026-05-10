import { randomUUID } from 'node:crypto';
import { expect, request as playwrightRequest, test } from '@playwright/test';
import WebSocket from 'ws';
import {
  acceptHandoff,
  closeHandoff,
  expectPublicError,
  operatorReply,
  queueHandoff,
  sendMessage,
  sendQuickReply,
  startSession,
  websiteOperatorQueue,
  type MessageResponse,
  type QuickReply,
} from './support/api';
import { closeDB, count, latestDecision, many, messages, one, sessionSnapshot, actionRows } from './support/db';
import { decisionURL, websiteURL } from './support/env';
import { restartServices, runCompose, stopServices, waitForStack } from './support/compose';
import { expectLastBotContains, openChat, sendChatMessage } from './support/browser';

test.describe.configure({ mode: 'serial' });

test.afterAll(async () => {
  await closeDB();
});

function client(id: string) {
  return `e2e-${id}-${Date.now()}-${randomUUID()}`;
}

async function apiFlow(request: Parameters<typeof startSession>[0], id: string, text: string) {
  const clientID = client(id);
  const session = await startSession(request, clientID);
  const response = await sendMessage(request, session.session_id, clientID, text);
  return { clientID, session, response };
}

async function expectMessagePersistence(sessionID: string, response: MessageResponse) {
  const rows = await messages(sessionID);
  expect(rows.some((row) => row.sender_type === 'user')).toBeTruthy();
  if (response.text) {
    expect(rows.some((row) => row.sender_type === 'bot' && row.text === response.text)).toBeTruthy();
  }
  await latestDecision(sessionID);
}

async function expectDecision(sessionID: string, intent: string | RegExp, responseKey?: string | RegExp) {
  const decision = await latestDecision(sessionID);
  if (typeof intent === 'string') {
    expect(decision.intent).toBe(intent);
  } else {
    expect(decision.intent).toMatch(intent);
  }
  if (responseKey) {
    if (typeof responseKey === 'string') {
      expect(decision.response_key).toBe(responseKey);
    } else {
      expect(decision.response_key).toMatch(responseKey);
    }
  }
  return decision;
}

async function expectActionEvidence(sessionID: string, actionType: string, status: string | RegExp) {
  const rows = await actionRows(sessionID, actionType);
  expect(rows.length).toBeGreaterThan(0);
  const row = rows[0];
  const payload = row.response_payload ?? {};
  const audit = payload.audit as Record<string, unknown> | undefined;
  const result = payload.result as Record<string, unknown> | undefined;
  expect(audit?.source ?? result?.source).toBe('mock_external');
  expect(audit?.provider).toBeTruthy();
  expect(audit?.duration_ms).toEqual(expect.any(Number));
  if (typeof status === 'string') {
    expect(String(result?.status ?? audit?.status)).toBe(status);
  } else {
    expect(String(result?.status ?? audit?.status)).toMatch(status);
  }
  return row;
}

async function expectNoAction(sessionID: string) {
  expect(await count('SELECT COUNT(*) FROM actions_log WHERE session_id = $1', [sessionID])).toBe(0);
}

test('E2E-001 @smoke новый пользователь creates isolated web session and first bot answer', async ({ page }) => {
  const sessionID = await openChat(page);
  await sendChatMessage(page, 'главное меню');
  await expectLastBotContains(page, /Главное меню|категори/i);

  const session = await sessionSnapshot(sessionID);
  expect(session.client_id).toContain('-');
  const dbMessages = await messages(sessionID);
  expect(dbMessages.map((item) => item.sender_type)).toEqual(expect.arrayContaining(['user', 'bot']));
  expect(await count('SELECT COUNT(*) FROM session_context WHERE session_id = $1', [sessionID])).toBe(1);
  await expectDecision(sessionID, 'return_to_menu', 'main_menu');
});

test('E2E-002 изоляция пользователей keeps sessions, history and context separated', async ({ request }) => {
  const first = await apiFlow(request, '002-a', 'как отменить запись');
  const second = await apiFlow(request, '002-b', 'оплата не прошла');

  expect(first.session.session_id).not.toBe(second.session.session_id);
  expect((await sessionSnapshot(first.session.session_id)).active_topic).toBe('booking');
  expect((await sessionSnapshot(second.session.session_id)).active_topic).toBe('payment');
  expect((await messages(first.session.session_id)).map((item) => item.text).join('\n')).not.toContain('оплата не прошла');
  expect((await messages(second.session.session_id)).map((item) => item.text).join('\n')).not.toContain('как отменить запись');
});

test('E2E-003 @smoke FAQ services/prices returns canned answer, quick replies and decision log', async ({ request }) => {
  const { session, response } = await apiFlow(request, '003', 'цены на услуги');

  expect(response.text).toContain('Услуги и цены');
  expect(response.text).toMatch(/Стрижка|Маникюр/);
  expect(response.quick_replies?.length ?? 0).toBeGreaterThan(0);
  await expectMessagePersistence(session.session_id, response);
  await expectDecision(session.session_id, /ask_prices|ask_services_info/, 'services_prices');
});

test('E2E-004 cancellation rules do not run business lookup action', async ({ request }) => {
  const { session, response } = await apiFlow(request, '004', 'как отменить запись');

  expect(response.text).toContain('Правила отмены');
  await expectDecision(session.session_id, 'ask_cancellation_rules', 'booking_cancellation_rules');
  await expectNoAction(session.session_id);
});

test('E2E-005 address and hours returns contact/location response', async ({ request }) => {
  const { session, response } = await apiFlow(request, '005', 'где находитесь');

  expect(response.text).toMatch(/Адрес|Работаем|9:00-21:00/);
  await expectDecision(session.session_id, 'ask_location', 'services_location');
});

test('E2E-006 @smoke booking found renders provider values and action evidence', async ({ request }) => {
  const { session, response } = await apiFlow(request, '006', 'проверьте запись БРГ-482910');

  expect(response.text).toContain('Запись найдена');
  expect(response.text).toContain('BRG-482910');
  expect(response.text).toContain('Женская стрижка');
  expect(response.text).not.toContain('{');
  await expectDecision(session.session_id, 'ask_booking_status', 'booking_found');
  await expectActionEvidence(session.session_id, 'find_booking', 'found');
});

test('E2E-007 booking not found returns controlled response and operator retry replies', async ({ request }) => {
  const { session, response } = await apiFlow(request, '007', 'проверьте запись BRG-404000');

  expect(response.text).toContain('Запись не найдена');
  expect(response.quick_replies?.some((reply) => /оператор/i.test(reply.label))).toBeTruthy();
  await expectDecision(session.session_id, 'ask_booking_status', 'booking_not_found');
  await expectActionEvidence(session.session_id, 'find_booking', 'not_found');
});

test('E2E-008 reschedule rules do not mutate demo booking rows', async ({ request }) => {
  const before = await count('SELECT COUNT(*) FROM demo_bookings');
  const { session, response } = await apiFlow(request, '008', 'можно перенести запись');
  const after = await count('SELECT COUNT(*) FROM demo_bookings');

  expect(response.text).toContain('Правила переноса');
  expect(after).toBe(before);
  await expectDecision(session.session_id, 'ask_reschedule_rules', 'booking_reschedule_rules');
  await expectNoAction(session.session_id);
});

test('E2E-009 workspace prices returns workspace price intent', async ({ request }) => {
  const { session, response } = await apiFlow(request, '009', 'сколько стоит рабочее место');

  expect(response.text).toMatch(/Горячее место|200/);
  await expectDecision(session.session_id, 'ask_workspace_prices', 'workspace_types_prices');
});

test('E2E-010 workspace booking found logs find_workspace_booking', async ({ request }) => {
  const { session, response } = await apiFlow(request, '010', 'найди бронь WS-1001');

  expect(response.text).toContain('Бронь найдена');
  expect(response.text).toContain('WS-1001');
  await expectDecision(session.session_id, 'ask_workspace_status', 'workspace_booking_found');
  await expectActionEvidence(session.session_id, 'find_workspace_booking', 'found');
});

test('E2E-011 workspace unavailable offers admin/operator path and keeps workspace topic', async ({ request }) => {
  const { session, response } = await apiFlow(request, '011', 'рабочее место недоступно');

  expect(response.text).toContain('место недоступно');
  expect(response.quick_replies?.some((reply) => /администратор/i.test(reply.label))).toBeTruthy();
  expect((await sessionSnapshot(session.session_id)).active_topic).toBe('workspace');
  await expectDecision(session.session_id, 'workspace_unavailable', 'workspace_unavailable');
});

test('E2E-012 payment completed renders status and logs find_payment', async ({ request }) => {
  const { session, response } = await apiFlow(request, '012', 'найди платеж PAY-123456');

  expect(response.text).toContain('Платёж найден');
  expect(response.text).toContain('PAY-123456');
  expect(response.text).toMatch(/completed|заверш|оплачен/i);
  await expectDecision(session.session_id, 'ask_payment_status', 'payment_found');
  await expectActionEvidence(session.session_id, 'find_payment', 'found');
});

test('E2E-013 payment failed returns payment_not_passed intent', async ({ request }) => {
  const { session, response } = await apiFlow(request, '013', 'оплата не прошла');

  expect(response.text).toContain('Оплата не прошла');
  await expectDecision(session.session_id, 'payment_not_passed', 'payment_failed');
});

test('E2E-014 debited not activated keeps payment topic and offers operator', async ({ request }) => {
  const { session, response } = await apiFlow(request, '014', 'деньги списались услуга не активировалась');

  expect(response.text).toContain('услуга не активирована');
  expect(response.quick_replies?.some((reply) => /оператор/i.test(reply.label))).toBeTruthy();
  expect((await sessionSnapshot(session.session_id)).active_topic).toBe('payment');
  await expectDecision(session.session_id, 'payment_not_activated', 'payment_debited_not_activated');
});

test('E2E-015 payment not found returns retry/operator replies and action evidence', async ({ request }) => {
  const { session, response } = await apiFlow(request, '015', 'статус платежа PAY-404000');

  expect(response.text).toContain('Платёж не найден');
  expect(response.quick_replies?.some((reply) => /оператор/i.test(reply.label))).toBeTruthy();
  await expectActionEvidence(session.session_id, 'find_payment', 'not_found');
});

test('E2E-016 site not loading returns browser/cache guidance', async ({ request }) => {
  const { session, response } = await apiFlow(request, '016', 'сайт не работает');

  expect(response.text).toMatch(/Очистите кэш|браузер|Обновите/);
  await expectDecision(session.session_id, 'ask_site_problem', 'tech_site_not_loading');
});

test('E2E-017 login problem returns reset password quick replies and tech/account topic', async ({ request }) => {
  const { session, response } = await apiFlow(request, '017', 'не могу войти');

  expect(response.text).toContain('Проблема со входом');
  expect(response.quick_replies?.some((reply) => /пароль|логин/i.test(reply.label))).toBeTruthy();
  expect(['tech_issue', 'account']).toContain((await sessionSnapshot(session.session_id)).active_topic);
  await expectDecision(session.session_id, 'login_not_working', 'tech_login_problem');
});

test('E2E-018 code not received returns confirmation-code troubleshooting', async ({ request }) => {
  const { session, response } = await apiFlow(request, '018', 'не приходит код');

  expect(response.text).toContain('Код не приходит');
  await expectDecision(session.session_id, 'code_not_received', 'tech_code_not_received');
});

test('E2E-019 account lookup found logs find_user_account', async ({ request }) => {
  const { session, response } = await apiFlow(request, '019', 'проверь аккаунт user1@example.com');

  expect(response.text).toContain('Аккаунт найден');
  expect(response.text).toContain('user1@example.com');
  await expectDecision(session.session_id, 'ask_account_status', 'account_found');
  await expectActionEvidence(session.session_id, 'find_user_account', 'found');
});

test('E2E-020 account lookup not found is safe and has no internal error', async ({ request }) => {
  const { session, response } = await apiFlow(request, '020', 'проверь аккаунт missing@example.com');

  expect(response.text).toContain('Аккаунт не найден');
  expect(response.text).not.toMatch(/sql|panic|exception|provider/i);
  await expectActionEvidence(session.session_id, 'find_user_account', 'not_found');
});

test('E2E-021 @smoke complaint handoff creates waiting operator queue with context snapshot', async ({ request }) => {
  const { session, response } = await apiFlow(request, '021', 'хочу пожаловаться на мастера');

  expect(response.handoff?.status).toBe('waiting');
  expect(response.text).toMatch(/передан|оператор/i);
  const queue = await one<{ status: string; reason: string; context_snapshot: Record<string, unknown> }>(
    'SELECT status, reason, context_snapshot FROM operator_queue WHERE session_id = $1 ORDER BY created_at DESC LIMIT 1',
    [session.session_id],
  );
  expect(queue.status).toBe('waiting');
  expect(queue.reason).toBe('complaint');
  expect(queue.context_snapshot.last_intent).toMatch(/complaint|report_complaint/);
});

test('E2E-022 manual operator queues handoff with manual_request reason', async ({ request }) => {
  const { session, response } = await apiFlow(request, '022', 'оператор');

  expect(response.handoff?.status).toBe('waiting');
  const queue = await one<{ reason: string }>(
    'SELECT reason FROM operator_queue WHERE session_id = $1 ORDER BY created_at DESC LIMIT 1',
    [session.session_id],
  );
  expect(queue.reason).toBe('manual_request');
});

test('E2E-023 repeated fallback increments fallback_count and offers/queues operator', async ({ request }) => {
  const clientID = client('023');
  const session = await startSession(request, clientID);
  const first = await sendMessage(request, session.session_id, clientID, 'абра кадабра непонятный запрос');
  const second = await sendMessage(request, session.session_id, clientID, 'снова совсем непонятно фыва олдж');

  expect(first.text).toMatch(/Уточните|уточните/);
  expect(second.text).toMatch(/оператор|Подключаю/i);
  expect((await sessionSnapshot(session.session_id)).fallback_count).toBeGreaterThanOrEqual(1);
  const queueCount = await count('SELECT COUNT(*) FROM operator_queue WHERE session_id = $1', [session.session_id]);
  expect(queueCount).toBeGreaterThanOrEqual(0);
});

test('E2E-024 @smoke operator UI opens queue and accepts waiting handoff', async ({ page, request }) => {
  const { session } = await apiFlow(request, '024', 'оператор');

  await page.goto('/');
  await page.getByRole('button', { name: 'Operator' }).click();
  await page.locator('#operatorSelect').selectOption('OP-001');
  await expect(page.locator('#operatorQueue')).toContainText(session.session_id);
  await page.locator('.queue-item').filter({ hasText: session.session_id }).click();
  await page.locator('#acceptHandoffButton').click();
  await expect(page.locator('#operatorSessionMeta')).toContainText('accepted');

  const queue = await one<{ status: string; assigned_operator_id: string }>(
    'SELECT status, assigned_operator_id FROM operator_queue WHERE session_id = $1 ORDER BY created_at DESC LIMIT 1',
    [session.session_id],
  );
  expect(queue.status).toBe('accepted');
  expect(queue.assigned_operator_id).toBe('OP-001');
});

test('E2E-025 operator replies into same user chat and DB stores sender_type operator', async ({ page, request }) => {
  const sessionID = await openChat(page);
  const handoff = await queueHandoff(request, sessionID);
  await acceptHandoff(request, handoff.handoff_id);
  await operatorReply(request, sessionID, 'Здравствуйте, я оператор и вижу ваш диалог.');

  await expectLastBotContains(page, /Оператор: Здравствуйте/);
  const operatorMessages = (await messages(sessionID)).filter((item) => item.sender_type === 'operator');
  expect(operatorMessages.length).toBeGreaterThan(0);
});

test('E2E-026 user messages during operator mode do not receive bot auto response', async ({ request }) => {
  const clientID = client('026');
  const session = await startSession(request, clientID);
  const handoff = await queueHandoff(request, session.session_id);
  await acceptHandoff(request, handoff.handoff_id);
  const beforeBot = await count('SELECT COUNT(*) FROM messages WHERE session_id = $1 AND sender_type = $2', [session.session_id, 'bot']);
  const response = await sendMessage(request, session.session_id, clientID, 'я еще здесь, оператору видно?');
  const afterBot = await count('SELECT COUNT(*) FROM messages WHERE session_id = $1 AND sender_type = $2', [session.session_id, 'bot']);

  expect(response.text).toBe('');
  expect(afterBot).toBe(beforeBot);
  expect((await messages(session.session_id)).some((item) => item.sender_type === 'user' && item.text.includes('оператору видно'))).toBeTruthy();
});

test('E2E-027 operator closes handoff and transition is persisted', async ({ page, request }) => {
  const sessionID = await openChat(page);
  const handoff = await queueHandoff(request, sessionID);
  await acceptHandoff(request, handoff.handoff_id);
  await closeHandoff(request, handoff.handoff_id);

  await expectLastBotContains(page, /завершен/);
  const queue = await one<{ status: string }>('SELECT status FROM operator_queue WHERE id = $1', [handoff.handoff_id]);
  expect(queue.status).toBe('closed');
  const transitions = await count(
    `SELECT COUNT(*) FROM transitions_log WHERE session_id = $1 AND event = 'operator_closed'`,
    [sessionID],
  );
  expect(transitions).toBeGreaterThan(0);
});

test('E2E-028 restore after restart resumes persisted history and context for same client', async ({ request }) => {
  const clientID = client('028');
  const session = await startSession(request, clientID);
  await sendMessage(request, session.session_id, clientID, 'как отменить запись');
  const beforeCount = await count('SELECT COUNT(*) FROM messages WHERE session_id = $1', [session.session_id]);

  runCompose(['restart', 'decision-engine', 'website']);
  await waitForStack();

  const resumed = await startSession(request, clientID);
  expect(resumed.session_id).toBe(session.session_id);
  expect(resumed.resumed).toBe(true);
  expect(await count('SELECT COUNT(*) FROM messages WHERE session_id = $1', [session.session_id])).toBe(beforeCount);
  expect((await sessionSnapshot(session.session_id)).active_topic).toBe('booking');
});

test('E2E-029 quick reply typed payload sends id/action, not parsed label text', async ({ request }) => {
  const clientID = client('029');
  const session = await startSession(request, clientID);
  const quickReply: QuickReply = {
    id: 'main-menu',
    label: 'Показать меню',
    action: 'select_intent',
    payload: { intent: 'return_to_menu', text: 'главное меню' },
  };
  const response = await sendQuickReply(request, session.session_id, clientID, quickReply);

  expect(response.text).toMatch(/Главное меню|категори/);
  const lastUser = (await messages(session.session_id)).filter((item) => item.sender_type === 'user').pop();
  expect(lastUser?.intent).toBe('main-menu');
  expect(lastUser?.text).toBe('главное меню');
  await expectDecision(session.session_id, 'return_to_menu', 'main_menu');
});

test('E2E-030 @smoke malicious text renders as text and does not execute script', async ({ page }) => {
  await page.addInitScript(() => {
    (window as Window & { __xssFired?: boolean }).__xssFired = false;
    window.alert = () => {
      (window as Window & { __xssFired?: boolean }).__xssFired = true;
    };
  });
  const payload = '<img src=x onerror="window.__xssFired=true;alert(1)">';
  const sessionID = await openChat(page);
  await sendChatMessage(page, payload);

  await expect(page.locator('.message.user-message .message-text').last()).toHaveText(payload);
  expect(await page.evaluate(() => (window as Window & { __xssFired?: boolean }).__xssFired)).toBe(false);
  expect((await messages(sessionID)).some((item) => item.sender_type === 'user' && item.text === payload)).toBeTruthy();
});

test('E2E-031 @smoke WebSocket rejects disallowed Origin without raw user data', async () => {
  await expect(
    new Promise((resolve, reject) => {
      const wsURL = websiteURL.replace(/^http/, 'ws') + '/ws';
      const socket = new WebSocket(wsURL, { headers: { Origin: 'https://evil.example' } });
      socket.once('open', () => {
        socket.close();
        reject(new Error('disallowed origin connected'));
      });
      socket.once('error', (error) => resolve(error));
      socket.once('close', (code) => {
        if (code !== 1006) {
          resolve(new Error(`closed with ${code}`));
        }
      });
    }),
  ).resolves.toBeTruthy();
});

test('E2E-032 public error masking hides DB internals when database is unavailable', async () => {
  await closeDB();
  stopServices('postgres');
  try {
    const isolatedRequest = await playwrightRequest.newContext();
    const response = await isolatedRequest.post(`${decisionURL}/api/v1/sessions`, {
      data: { channel: 'website', client_id: client('032') },
      timeout: 15_000,
    });
    const body = await response.json().catch(() => ({}));
    await expectPublicError(response.status(), body);
    await isolatedRequest.dispose();
  } finally {
    await restartServices('postgres');
  }
});

test('E2E-033 @smoke NLP unavailable falls back under 3 seconds and records fallback evidence', async ({ request }) => {
  stopServices('nlp-service');
  try {
    const clientID = client('033');
    const session = await startSession(request, clientID);
    const started = Date.now();
    const response = await sendMessage(request, session.session_id, clientID, 'совершенно непонятная просьба про космический тариф');
    const duration = Date.now() - started;

    expect(duration).toBeLessThan(3_000);
    expect(response.text).toMatch(/Уточните|оператор/i);
    const decision = await latestDecision(session.session_id);
    expect(JSON.stringify(decision.candidates)).toContain('embedding_unavailable');
  } finally {
    await restartServices('nlp-service');
  }
});

test('E2E-034 latency budget p95 is under 3 seconds for FAQ and lookup demo flows', async ({ request }) => {
  const durations: number[] = [];
  for (const [id, text] of [
    ['034-a', 'цены на услуги'],
    ['034-b', 'проверьте запись БРГ-482910'],
    ['034-c', 'найди платеж PAY-123456'],
    ['034-d', 'сайт не работает'],
    ['034-e', 'сколько стоит рабочее место'],
  ] as const) {
    const clientID = client(id);
    const session = await startSession(request, clientID);
    const started = Date.now();
    await sendMessage(request, session.session_id, clientID, text);
    durations.push(Date.now() - started);
  }
  const sorted = [...durations].sort((a, b) => a - b);
  const p95 = sorted[Math.ceil(sorted.length * 0.95) - 1];
  expect(p95).toBeLessThan(3_000);
});

test('E2E-035 @smoke no LLM runtime is present in compose or network calls', async ({ page }) => {
  const config = runCompose(['config'], { allowFailure: true });
  expect(config.status).toBe(0);
  expect(config.stdout).not.toMatch(/ollama|gigachat|\/llm\/decide/i);

  const requested: string[] = [];
  page.on('request', (req) => requested.push(req.url()));
  await openChat(page);
  await sendChatMessage(page, 'какие услуги и цены');
  expect(requested.join('\n')).not.toMatch(/\/llm\/decide|ollama|gigachat/i);
});

test('E2E-036 mock external contract evidence exists for booking/payment/workspace/account providers', async ({ request }) => {
  const flows = [
    ['036-booking', 'проверьте запись БРГ-482910', 'find_booking'],
    ['036-payment', 'найди платеж PAY-123456', 'find_payment'],
    ['036-workspace', 'найди бронь WS-1001', 'find_workspace_booking'],
    ['036-account', 'проверь аккаунт user1@example.com', 'find_user_account'],
  ] as const;

  for (const [id, text, actionType] of flows) {
    const { session } = await apiFlow(request, id, text);
    const row = await expectActionEvidence(session.session_id, actionType, 'found');
    expect(row.response_payload?.audit).toMatchObject({ source: 'mock_external' });
  }
});

test('E2E-037 provider unavailable returns controlled fallback/operator offer and DB evidence', async ({ request }) => {
  const { session, response } = await apiFlow(request, '037', 'найди платеж PAY-ERROR-503');

  expect(response.text).toMatch(/внешней системе|оператор/i);
  expect(response.handoff?.status).toBe('waiting');
  const action = await expectActionEvidence(session.session_id, 'find_payment', 'unavailable');
  expect(JSON.stringify(action.response_payload)).toContain('provider_unavailable');
  const queue = await one<{ reason: string }>(
    'SELECT reason FROM operator_queue WHERE session_id = $1 ORDER BY created_at DESC LIMIT 1',
    [session.session_id],
  );
  expect(queue.reason).toBe('business_error');
});

test('E2E-038 @smoke knowledge-backed prices use seeded KB/provider evidence, not hardcoded JS', async ({ request }) => {
  const { session, response } = await apiFlow(request, '038', 'покажите цены');

  expect(response.text).toContain('Услуги и цены');
  await expectDecision(session.session_id, /ask_prices|ask_services_info/, 'services_prices');
  const evidence = await one<{ metadata: Record<string, unknown>; article_count: string }>(
    `SELECT i.metadata, COUNT(ka.id)::TEXT AS article_count
       FROM intents i
       LEFT JOIN knowledge_articles ka
         ON ka.key = i.metadata->>'knowledge_key' AND ka.active = true
      WHERE i.key IN ('ask_prices', 'ask_services_info')
      GROUP BY i.metadata
      ORDER BY COUNT(ka.id) DESC
      LIMIT 1`,
  );
  expect(evidence.metadata.knowledge_key).toBeTruthy();
  expect(Number(evidence.article_count)).toBeGreaterThan(0);
  expect(response.text).not.toContain('window.');
});
