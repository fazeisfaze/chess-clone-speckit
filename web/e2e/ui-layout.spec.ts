import { expect, test } from "@playwright/test";
import { createInvitation } from "./helpers";

test("active match layout adapts from a board-and-sidebar desktop view to mobile", async ({ browser }) => {
  const hostContext = await browser.newContext({ viewport: { width: 1280, height: 900 } });
  const guestContext = await browser.newContext();
  const host = await hostContext.newPage();
  const guest = await guestContext.newPage();
  await host.goto("/");
  const invitation = await createInvitation(host);
  await guest.goto(invitation);
  await guest.getByRole("button", { name: "Join match" }).click();
  await expect(host.getByRole("complementary", { name: "Match controls" })).toBeVisible();

  const desktopColumns = await host.locator(".game-layout").evaluate((element) =>
    getComputedStyle(element).gridTemplateColumns.trim().split(/\s+/).length,
  );
  expect(desktopColumns).toBe(2);

  await host.setViewportSize({ width: 390, height: 844 });
  const mobileColumns = await host.locator(".game-layout").evaluate((element) =>
    getComputedStyle(element).gridTemplateColumns.trim().split(/\s+/).length,
  );
  expect(mobileColumns).toBe(1);
  await expect(host.getByRole("grid", { name: "Chess board" })).toBeVisible();

  await hostContext.close();
  await guestContext.close();
});
