import { test, expect } from "@playwright/test";

const baseLocale = "/en";

test.describe("Forgot Password", () => {
  test("forgot password link is visible on sign-in page", async ({ page }) => {
    await page.goto(`${baseLocale}/sign-in`);
    await expect(page.locator('[data-test-id="forgot-password-link"]')).toBeVisible();
  });

  test("forgot password page loads", async ({ page }) => {
    await page.goto(`${baseLocale}/forgot-password`);
    await expect(page.locator('[data-test-id="forgot-password-page"]')).toBeVisible();
    await expect(page.locator('[data-test-id="forgot-password-heading"]')).toBeVisible();
  });

  test("submitting email shows success page regardless of email existence", async ({ page }) => {
    await page.goto(`${baseLocale}/forgot-password`);

    // Submit with non-existent email (timing attack protection - should still show success)
    await page.fill(
      '[data-test-id="forgot-password-email"]',
      `nonexistent-${Date.now()}@example.com`
    );
    await page.click('[data-test-id="forgot-password-submit"]');

    // Should show generic success message (security best practice)
    await expect(page.locator('[data-test-id="forgot-password-sent-page"]')).toBeVisible();
    await expect(page.locator('[data-test-id="forgot-password-sent-heading"]')).toBeVisible();
  });

  test("reset password page with missing token redirects to forgot-password", async ({ page }) => {
    await page.goto(`${baseLocale}/reset-password`);

    // Should redirect to forgot-password due to missing token
    await expect(page).toHaveURL(`${baseLocale}/forgot-password`);
  });

  test("reset password page with invalid token redirects to forgot-password", async ({ page }) => {
    await page.goto(`${baseLocale}/reset-password?token=invalid-token-12345`);

    // Should redirect to forgot-password due to invalid token
    await expect(page).toHaveURL(`${baseLocale}/forgot-password`);
  });

  test("reset password page with valid token format shows form", async ({ page }) => {
    // Use a properly formatted token (64 hex characters for SHA-256 hash)
    const validFormatToken = "a".repeat(64);
    await page.goto(`${baseLocale}/reset-password?token=${validFormatToken}`);

    // The form should be visible (token format is valid, though it may not exist in DB)
    await expect(page.locator('[data-test-id="reset-password-page"]')).toBeVisible();
    await expect(page.locator('[data-test-id="reset-password-heading"]')).toBeVisible();
  });

  test("reset password with mismatched passwords shows error", async ({ page }) => {
    // Use a properly formatted token
    const validFormatToken = "b".repeat(64);
    await page.goto(`${baseLocale}/reset-password?token=${validFormatToken}`);

    // Fill in mismatched passwords
    await page.fill("#password", "newpassword123");
    await page.fill("#password_confirm", "differentpassword456");
    await page.click('[data-test-id="reset-password-submit"]');

    // Should show error message about passwords not matching
    await expect(page.locator(".lmn-alert-danger")).toBeVisible();
  });

  test("forgot password flow with existing user", async ({ page }) => {
    // First, create an account
    const email = `forgot-pw-${Date.now()}@example.com`;
    const originalPassword = "originalpassword123";
    const newPassword = "newpassword456";

    await page.goto(`${baseLocale}/sign-up`);
    await page.fill("#email", email);
    await page.fill("#password", originalPassword);
    await page.fill("#password_confirm", originalPassword);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(`${baseLocale}/dashboard`);

    // Sign out
    await page.locator('form[action$="/sign-out"] button[type="submit"]').click();
    await expect(page).toHaveURL(baseLocale);

    // Navigate to forgot password page
    await page.goto(`${baseLocale}/forgot-password`);
    await expect(page.locator('[data-test-id="forgot-password-page"]')).toBeVisible();

    // Submit forgot password request
    await page.fill('[data-test-id="forgot-password-email"]', email);
    await page.click('[data-test-id="forgot-password-submit"]');

    // Should show success page
    await expect(page.locator('[data-test-id="forgot-password-sent-page"]')).toBeVisible();

    // Note: In a real test environment with email capture, we would:
    // 1. Extract the reset token from the captured email
    // 2. Navigate to the reset URL with the token
    // 3. Set a new password
    // 4. Verify we can sign in with the new password
    //
    // Since we can't access emails in this e2e test setup, we verify that:
    // - The forgot password form accepts the request
    // - The success page is shown
    // - The reset password page validates tokens correctly
  });

  test("navigating from sign-in to forgot-password and back", async ({ page }) => {
    await page.goto(`${baseLocale}/sign-in`);

    // Click forgot password link
    await page.click('[data-test-id="forgot-password-link"]');
    await expect(page).toHaveURL(`${baseLocale}/forgot-password`);
    await expect(page.locator('[data-test-id="forgot-password-page"]')).toBeVisible();

    // Click back to sign-in link
    await page.click("text=Back to sign in");
    await expect(page).toHaveURL(`${baseLocale}/sign-in`);
    await expect(page.locator('[data-test-id="sign-in-page"]')).toBeVisible();
  });

  test("forgot password page form validation - email is required", async ({ page }) => {
    await page.goto(`${baseLocale}/forgot-password`);

    // Try to submit without email
    await page.click('[data-test-id="forgot-password-submit"]');

    // HTML5 validation should prevent submission
    await expect(page.locator('[data-test-id="forgot-password-page"]')).toBeVisible();
  });

  test("reset password form validation - passwords are required", async ({ page }) => {
    const validFormatToken = "c".repeat(64);
    await page.goto(`${baseLocale}/reset-password?token=${validFormatToken}`);

    // Try to submit without filling passwords
    await page.click('[data-test-id="reset-password-submit"]');

    // HTML5 validation should prevent submission (required fields)
    await expect(page.locator('[data-test-id="reset-password-page"]')).toBeVisible();
  });
});
