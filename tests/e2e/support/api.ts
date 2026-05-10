import { APIRequestContext, expect } from '@playwright/test';
import { decisionURL, websiteURL } from './env';

export type SessionResponse = {
  session_id: string;
  user_id: string;
  mode: string;
  active_topic: string | null;
  resumed: boolean;
};

export type MessageResponse = {
  session_id: string;
  user_message_id: string;
  bot_message_id: string;
  mode: string;
  active_topic: string | null;
  text: string;
  quick_replies?: QuickReply[];
  handoff?: Handoff | null;
};

export type QuickReply = {
  id: string;
  label: string;
  action: string;
  payload?: Record<string, unknown>;
};

export type Handoff = {
  handoff_id: string;
  session_id: string;
  status: string;
  reason?: string;
  operator_id?: string | null;
};

export async function startSession(request: APIRequestContext, clientID: string) {
  const response = await request.post(`${decisionURL}/api/v1/sessions`, {
    data: { channel: 'website', client_id: clientID },
  });
  await expect(response).toBeOK();
  return (await response.json()) as SessionResponse;
}

export async function sendMessage(
  request: APIRequestContext,
  sessionID: string,
  clientID: string,
  text: string,
  eventID?: string,
) {
  const response = await request.post(`${decisionURL}/api/v1/messages`, {
    data: {
      session_id: sessionID,
      channel: 'website',
      client_id: clientID,
      type: 'user_message',
      text,
      event_id: eventID,
    },
  });
  await expect(response).toBeOK();
  return (await response.json()) as MessageResponse;
}

export async function sendQuickReply(
  request: APIRequestContext,
  sessionID: string,
  clientID: string,
  quickReply: QuickReply,
  eventID?: string,
) {
  const response = await request.post(`${decisionURL}/api/v1/messages`, {
    data: {
      session_id: sessionID,
      channel: 'website',
      client_id: clientID,
      type: 'quick_reply.selected',
      text: String(quickReply.payload?.text ?? ''),
      quick_reply: quickReply,
      event_id: eventID,
    },
  });
  await expect(response).toBeOK();
  return (await response.json()) as MessageResponse;
}

export async function queueHandoff(request: APIRequestContext, sessionID: string, reason = 'manual_request') {
  const response = await request.post(`${decisionURL}/api/v1/operator/queue/${sessionID}/request`, {
    data: { reason },
  });
  await expect(response).toBeOK();
  const body = (await response.json()) as { handoff: Handoff };
  return body.handoff;
}

export async function acceptHandoff(request: APIRequestContext, handoffID: string, operatorID = 'OP-001') {
  const response = await request.post(`${decisionURL}/api/v1/operator/queue/${handoffID}/accept`, {
    data: { operator_id: operatorID },
  });
  await expect(response).toBeOK();
  const body = (await response.json()) as { handoff: Handoff };
  return body.handoff;
}

export async function closeHandoff(request: APIRequestContext, handoffID: string, operatorID = 'OP-001') {
  const response = await request.post(`${decisionURL}/api/v1/operator/queue/${handoffID}/close`, {
    data: { operator_id: operatorID },
  });
  await expect(response).toBeOK();
  const body = (await response.json()) as { handoff: Handoff };
  return body.handoff;
}

export async function operatorReply(
  request: APIRequestContext,
  sessionID: string,
  text: string,
  operatorID = 'OP-001',
) {
  const response = await request.post(`${decisionURL}/api/v1/operator/sessions/${sessionID}/messages`, {
    data: { operator_id: operatorID, text },
  });
  await expect(response).toBeOK();
  return response.json();
}

export async function websiteOperatorQueue(request: APIRequestContext, status = 'waiting') {
  const response = await request.get(`${websiteURL}/api/operator/queue?status=${encodeURIComponent(status)}`);
  await expect(response).toBeOK();
  return response.json();
}

export async function expectPublicError(responseStatus: number, body: unknown) {
  expect(responseStatus).toBeGreaterThanOrEqual(400);
  expect(JSON.stringify(body)).toMatch(/"error"/);
  expect(JSON.stringify(body)).not.toMatch(/SELECT|INSERT|pq:|pgconn|stack|panic|Traceback|provider unavailable/i);
}
