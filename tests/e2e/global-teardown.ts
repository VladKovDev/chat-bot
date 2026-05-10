import { runCompose } from './support/compose';

async function globalTeardown() {
  if (process.env.E2E_KEEP_COMPOSE === '1' || process.env.E2E_SKIP_COMPOSE === '1') {
    return;
  }

  runCompose(['down', '-v', '--remove-orphans'], { allowFailure: true });
}

export default globalTeardown;
