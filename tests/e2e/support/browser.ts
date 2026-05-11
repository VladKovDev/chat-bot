import { expect, Page } from '@playwright/test';

export async function openChat(page: Page) {
  await page.goto('/');
  await expect(page.locator('#statusText')).toHaveText(/Connected|Онлайн/);
  await page.waitForFunction(() => Boolean((window as Window & { currentSessionId?: string }).currentSessionId));
  return page.evaluate(() => (window as Window & { currentSessionId: string }).currentSessionId);
}

export async function sendChatMessage(page: Page, text: string) {
  await page.locator('#messageInput').fill(text);
  await page.locator('#sendButton').click();
  await expect(page.locator('.message.user-message .message-text').last()).toContainText(text);
}

export async function expectLastBotContains(page: Page, text: string | RegExp) {
  await expect(page.locator('.message.bot-message .message-text').last()).toContainText(text);
}
