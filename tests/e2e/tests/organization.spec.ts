import { test, expect } from "@playwright/test";

const baseLocale = "/en";

test.describe("Organization", () => {
  test("new user gets default organization", async ({ page }) => {
    const email = `org-${Date.now()}@example.com`;

    // Sign up
    await page.goto(`${baseLocale}/sign-up`);
    await page.fill("#email", email);
    await page.fill("#password", "password123");
    await page.fill("#password_confirm", "password123");
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(`${baseLocale}/dashboard`);

    // Navigate to organization page
    await page.goto(`${baseLocale}/organization`);
    await expect(page).toHaveURL(`${baseLocale}/organization`);

    // Should have organization dashboard structure
    await expect(page.locator('form[action$="/organization/create"]')).toBeVisible();
  });

  test("can create new organization", async ({ page }) => {
    const email = `orgcreate-${Date.now()}@example.com`;
    const organizationName = `My Test Organization ${Date.now()}-${Math.floor(
      Math.random() * 100000
    )}`;

    // Sign up
    await page.goto(`${baseLocale}/sign-up`);
    await page.fill("#email", email);
    await page.fill("#password", "password123");
    await page.fill("#password_confirm", "password123");
    await page.click('button[type="submit"]');

    // Go to org page and create new org
    await page.goto(`${baseLocale}/organization`);
    await page.fill('form[action$="/organization/create"] input[name="name"]', organizationName);
    await page.click('form[action$="/organization/create"] button[type="submit"]');

    // Should return to org dashboard without an error flash
    await expect(page).toHaveURL(`${baseLocale}/organization`);
    await expect(page.locator(".lmn-alert-danger")).toHaveCount(0);
  });
});
