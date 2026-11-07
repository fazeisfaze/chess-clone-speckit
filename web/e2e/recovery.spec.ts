import { expect, test } from "@playwright/test";
import { createInvitation } from "./helpers";

test("a guest reload restores the current board and running clock", async ({ browser }) => {
  const hostContext = await browser.newContext();
  const guestContext = await browser.newContext();
  const host = await hostContext.newPage();
  const guest = await guestContext.newPage();

  await host.goto("/");
  const invitation = await createInvitation(host);
  await guest.goto(invitation);
  await guest.getByRole("button", { name: "Join match" }).click();

  await host.getByRole("gridcell", { name: /e2 white pawn/i }).click();
  await host.getByRole("gridcell", { name: /e4 empty/i }).click();
  await expect(guest.getByRole("gridcell", { name: /e4 white pawn/i })).toBeVisible();

  await guest.reload();
  await expect(guest.getByRole("gridcell", { name: /e4 white pawn/i })).toBeVisible();
  await expect(guest.getByText("Black to move")).toBeVisible();
  await expect(guest.getByText(/^Black \d:\d{2}$/)).toBeVisible();
  await expect(guest.locator(".self-strip .player-identity strong")).toHaveText("Black");

  await hostContext.close();
  await guestContext.close();
});
