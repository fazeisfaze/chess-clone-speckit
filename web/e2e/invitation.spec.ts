import { expect, test } from "@playwright/test";
import { createInvitation } from "./helpers";

test("two guests create and join an invitation match", async ({ browser }) => {
  const hostContext = await browser.newContext();
  const guestContext = await browser.newContext();
  await hostContext.grantPermissions(["clipboard-read", "clipboard-write"]);
  const host = await hostContext.newPage();
  const guest = await guestContext.newPage();

  await host.goto("/");
  await host.getByLabel("Time control").selectOption("5+0");
  const href = await createInvitation(host);
  const copyInvitation = host.getByRole("button", { name: "Copy invitation link" });
  await expect(copyInvitation).toBeVisible();
  await copyInvitation.click();
  await expect(host.getByRole("status")).toHaveText("Invitation link copied.");
  expect(await host.evaluate(() => navigator.clipboard.readText())).toBe(href);
  await expect(host.getByText("Waiting for an opponent")).toBeVisible();
  await guest.goto(href);
  await guest.getByRole("button", { name: "Join match" }).click();
  await expect(host.getByText("Waiting for an opponent")).toHaveCount(0);
  await expect(host.getByRole("grid", { name: "Chess board" })).toBeVisible();
  await expect(guest.getByRole("grid", { name: "Chess board" })).toBeVisible();
  await expect(host.locator(".self-strip .player-identity strong")).toHaveText("White");
  await expect(guest.locator(".self-strip .player-identity strong")).toHaveText("Black");
  await expect(host.getByText(/^White [45]:\d{2}$/)).toBeVisible();
  await expect(guest.getByText(/^White [45]:\d{2}$/)).toBeVisible();

  await hostContext.close();
  await guestContext.close();
});
