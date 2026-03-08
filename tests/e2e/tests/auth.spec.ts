import { test, expect } from "@playwright/test";

const baseLocale = "/en";

test.describe("Authentication", () => {
  test("homepage loads", async ({ page }) => {
    await page.goto(baseLocale);
    await expect(page.locator("body")).toBeVisible();
  });

  test("sign up flow", async ({ page }) => {
    await page.goto(`${baseLocale}/sign-up`);
    await expect(page.locator('form[action$="/sign-up"]')).toBeVisible();

    const email = `test-${Date.now()}@example.com`;
    await page.fill("#email", email);
    await page.fill("#password", "password123");
    await page.fill("#password_confirm", "password123");
    await page.click('button[type="submit"]');

    // Should redirect to dashboard after successful sign-up
    await expect(page).toHaveURL(`${baseLocale}/dashboard`);
  });

  test("sign in flow", async ({ page }) => {
    // First, create an account
    const email = `signin-${Date.now()}@example.com`;
    await page.goto(`${baseLocale}/sign-up`);
    await page.fill("#email", email);
    await page.fill("#password", "password123");
    await page.fill("#password_confirm", "password123");
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(`${baseLocale}/dashboard`);

    // Sign out
    await page.locator('form[action$="/sign-out"] button[type="submit"]').click();
    await expect(page).toHaveURL(baseLocale);

    // Sign in
    await page.goto(`${baseLocale}/sign-in`);
    await page.fill('[data-test-id="sign-in-email"]', email);
    await page.fill("#password", "password123");
    await page.click('[data-test-id="sign-in-submit"]');
    await expect(page).toHaveURL(`${baseLocale}/dashboard`);
  });

  test("sign in with invalid credentials shows error", async ({ page }) => {
    await page.goto(`${baseLocale}/sign-in`);
    await page.fill('[data-test-id="sign-in-email"]', "wrong@example.com");
    await page.fill("#password", "wrongpassword");
    await page.click('[data-test-id="sign-in-submit"]');

    await expect(page.locator(".lmn-alert-danger")).toBeVisible();
  });
});
