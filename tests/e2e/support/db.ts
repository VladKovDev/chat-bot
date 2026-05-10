import { Pool, QueryResultRow } from 'pg';
import { databaseURL } from './env';

let pool: Pool | undefined;

export function db() {
  if (!pool) {
    pool = new Pool({ connectionString: databaseURL, max: 4 });
  }
  return pool;
}

export async function closeDB() {
  if (pool) {
    await pool.end();
    pool = undefined;
  }
}

export async function one<T extends QueryResultRow>(sql: string, values: unknown[] = []) {
  const result = await db().query<T>(sql, values);
  if (result.rowCount !== 1) {
    throw new Error(`Expected one row, got ${result.rowCount ?? 0}: ${sql}`);
  }
  return result.rows[0];
}

export async function many<T extends QueryResultRow>(sql: string, values: unknown[] = []) {
  const result = await db().query<T>(sql, values);
  return result.rows;
}

export async function count(sql: string, values: unknown[] = []) {
  const row = await one<{ count: string }>(sql, values);
  return Number(row.count);
}

export async function sessionSnapshot(sessionID: string) {
  return one<{
    id: string;
    client_id: string;
    active_topic: string;
    mode: string;
    last_intent: string;
    fallback_count: number;
    operator_status: string;
    status: string;
  }>(
    `SELECT id, client_id, active_topic, mode, last_intent, fallback_count, operator_status, status
       FROM sessions
      WHERE id = $1`,
    [sessionID],
  );
}

export async function latestDecision(sessionID: string) {
  return one<{
    intent: string;
    response_key: string;
    confidence: number | null;
    low_confidence: boolean;
    candidates: unknown;
  }>(
    `SELECT intent, response_key, confidence, low_confidence, candidates
       FROM decision_logs
      WHERE session_id = $1
      ORDER BY created_at DESC
      LIMIT 1`,
    [sessionID],
  );
}

export async function actionRows(sessionID: string, actionType?: string) {
  const values: unknown[] = [sessionID];
  const actionFilter = actionType ? 'AND action_type = $2' : '';
  if (actionType) {
    values.push(actionType);
  }
  return many<{
    action_type: string;
    request_payload: Record<string, unknown> | null;
    response_payload: Record<string, unknown> | null;
    error: string | null;
  }>(
    `SELECT action_type, request_payload, response_payload, error
       FROM actions_log
      WHERE session_id = $1 ${actionFilter}
      ORDER BY created_at DESC`,
    values,
  );
}

export async function messages(sessionID: string) {
  return many<{ sender_type: string; text: string; intent: string | null; created_at: Date }>(
    `SELECT sender_type, text, intent, created_at
       FROM messages
      WHERE session_id = $1
      ORDER BY created_at ASC`,
    [sessionID],
  );
}
