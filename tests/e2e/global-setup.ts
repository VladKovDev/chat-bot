import { runCompose, waitForStack } from './support/compose';

async function globalSetup() {
  if (process.env.E2E_SKIP_COMPOSE === '1') {
    await waitForStack();
    return;
  }

  runCompose(['down', '-v', '--remove-orphans']);
  runCompose(['up', '--build', '-d']);
  await waitForStack();
}

export default globalSetup;
