export const composeProject = process.env.E2E_COMPOSE_PROJECT ?? 'chat-bot-e2e';
export const decisionURL = process.env.E2E_DECISION_URL ?? 'http://127.0.0.1:18080';
export const websiteURL = process.env.E2E_WEBSITE_URL ?? 'http://localhost:18081';
export const nlpURL = process.env.E2E_NLP_URL ?? 'http://127.0.0.1:18082';
export const mockExternalURL = process.env.E2E_MOCK_EXTERNAL_URL ?? 'http://127.0.0.1:18090';
export const databaseURL =
  process.env.E2E_DATABASE_URL ?? 'postgres://postgres:postgres@127.0.0.1:15442/chat_bot?sslmode=disable';

export const composeEnv: NodeJS.ProcessEnv = {
  ...process.env,
  COMPOSE_PROJECT_NAME: composeProject,
  POSTGRES_PORT: process.env.E2E_POSTGRES_PORT ?? '15442',
  DECISION_ENGINE_PORT: process.env.E2E_DECISION_PORT ?? '18080',
  WEBSITE_PORT: process.env.E2E_WEBSITE_PORT ?? '18081',
  NLP_PORT: process.env.E2E_NLP_PORT ?? '18082',
  MOCK_EXTERNAL_PORT: process.env.E2E_MOCK_EXTERNAL_PORT ?? '18090',
};
