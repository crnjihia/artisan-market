import { test, expect } from '@playwright/test';
import { execSync } from 'child_process';

/**
 * Helper to get the container ID of the notification service.
 */
function getNotificationContainerId(): string {
  try {
    const out = execSync('docker ps --filter "name=notification" --format "{{.ID}}"', { encoding: 'utf8' }).trim();
    return out.split('\n')[0] || '';
  } catch {
    return '';
  }
}

/**
 * Returns the recent logs from the notification container.
 */
function getNotificationLogs(): string {
  const containerId = getNotificationContainerId();
  if (!containerId) return '';
  try {
    const logs = execSync(`docker logs ${containerId} --since 1m`, { encoding: 'utf8' });
    return logs;
  } catch {
    return '';
  }
}

test('full checkout flow shows success and triggers mock notifications', async ({ page }) => {
  // 1️⃣ Load home page
  await page.goto('/');

  // 2️⃣ Add first product to cart
  const addButton = page.locator('text=Add to Cart').first();
  await expect(addButton).toBeVisible();
  await addButton.click();

  // 3️⃣ Open cart drawer (click cart toggle button)
  await page.click('#cart-toggle');

  // 4️⃣ Click Checkout button (fixed position button)
  await page.click('button:has-text("Checkout")');

  // 5️⃣ Confirm checkout in modal
  await page.click('button:has-text("Confirm")');

  // 6️⃣ Wait for success message
  await expect(page.locator('text=Payment succeeded!')).toBeVisible({ timeout: 30000 });

  // 7️⃣ Verify that mock SMS and Email were emitted (allow some time for background consumers)
  await page.waitForTimeout(3000);
  const logs = getNotificationLogs();
  expect(logs).toContain('[Mock SMS]');
  expect(logs).toContain('[Mock Email]');
});
