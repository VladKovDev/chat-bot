import { spawnSync } from 'node:child_process';
import { composeEnv, composeProject, decisionURL, websiteURL } from './env';

type ComposeService = 'postgres' | 'decision-engine' | 'website' | 'nlp-service' | 'mock-external-services';

export function runCompose(args: string[], options: { allowFailure?: boolean } = {}) {
  const result = spawnSync('docker', ['compose', '-p', composeProject, ...args], {
    cwd: process.cwd(),
    env: composeEnv,
    encoding: 'utf8',
    stdio: options.allowFailure ? 'pipe' : 'inherit',
  });

  if (!options.allowFailure && result.status !== 0) {
    throw new Error(`docker compose ${args.join(' ')} failed with exit ${result.status}`);
  }

  return result;
}

export async function waitForHTTP(url: string, timeoutMs = 120_000) {
  const deadline = Date.now() + timeoutMs;
  let lastError = '';

  while (Date.now() < deadline) {
    try {
      const response = await fetch(url, { cache: 'no-store' });
      if (response.ok) {
        return;
      }
      lastError = `HTTP ${response.status}`;
    } catch (error) {
      lastError = error instanceof Error ? error.message : String(error);
    }
    await new Promise((resolve) => setTimeout(resolve, 2_000));
  }

  throw new Error(`Timed out waiting for ${url}: ${lastError}`);
}

export async function waitForStack() {
  await waitForHTTP(`${decisionURL}/api/v1/ready`);
  await waitForHTTP(`${websiteURL}/health`);
}

export async function restartServices(...services: ComposeService[]) {
  runCompose(['start', ...services]);
  if (services.includes('postgres') || services.includes('nlp-service') || services.includes('decision-engine')) {
    runCompose(['restart', 'decision-engine', 'website']);
  } else if (services.includes('website')) {
    runCompose(['restart', 'website']);
  }
  await waitForStack();
}

export function stopServices(...services: ComposeService[]) {
  runCompose(['stop', ...services]);
}
