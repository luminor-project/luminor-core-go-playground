import { test, expect } from "@playwright/test";

const locales = ["en", "de", "fr"] as const;

test.describe("Locale routing and metadata", () => {
  test("root redirects to a locale-prefixed URL", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveURL(/\/(en|de|fr)$/);
  });

  test("unprefixed content path redirects to locale-prefixed path", async ({ page }) => {
    await page.goto("/about");
    await expect(page).toHaveURL(/\/(en|de|fr)\/about$/);
  });

  for (const locale of locales) {
    test(`/${locale} sets lang, canonical, and localized links`, async ({ page }) => {
      await page.goto(`/${locale}`);

      await expect(page.locator("html")).toHaveAttribute("lang", locale);

      const canonical = page.locator('link[rel="canonical"]');
      await expect(canonical).toHaveAttribute("href", new RegExp(`/${locale}$`));

      for (const alt of locales) {
        await expect(page.locator(`link[rel="alternate"][hreflang="${alt}"]`)).toHaveAttribute(
          "href",
          new RegExp(`/${alt}$`)
        );
      }

      // Internal navigation should remain locale-prefixed.
      const internalLinks = page.locator('a[href^="/"]:not([href^="/static/"])');
      const count = await internalLinks.count();
      for (let i = 0; i < count; i++) {
        const href = await internalLinks.nth(i).getAttribute("href");
        expect(href).toBeTruthy();
        expect(href).toMatch(/^\/(en|de|fr)(\/|$)/);
      }
    });
  }
});
