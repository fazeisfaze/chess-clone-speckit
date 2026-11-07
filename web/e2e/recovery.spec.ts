import { expect, test } from "@playwright/test";

test("a guest reload restores the current board and running clock", async ({ browser }) => {
  const hostContext = await browser.newContext();
  const guestContext = await browser.newContext();
  const host = await hostContext.newPage();
  const guest = await guestContext.newPage();

  await host.goto("/");
  await host.getByRole("button", { name: "Create match" }).click();
  const invitation = await host.getByRole("link", { name: "Invitation link" }).getAttribute("href");
  if (!invitation) throw new Error("invitation link missing");
  await guest.goto(invitation);
  await guest.getByRole("button", { name: "Join match" }).click();

  await host.getByRole("gridcell", { name: /e2 white pawn/i }).click();
  await host.getByRole("gridcell", { name: /e4 empty/i }).click();
  await expect(guest.getByRole("gridcell", { name: /e4 white pawn/i })).toBeVisible();

  await guest.reload();
  await expect(guest.getByRole("gridcell", { name: /e4 white pawn/i })).toBeVisible();
  await expect(guest.getByText("Black to move")).toBeVisible();
  await expect(guest.getByText(/^Black \d:\d{2}$/)).toBeVisible();
  await expect(guest.getByText("You are playing Black")).toBeVisible();

  await hostContext.close();
  await guestContext.close();
});
