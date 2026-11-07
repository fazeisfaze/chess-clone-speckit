import { expect, test, type Browser } from "@playwright/test";
import { createInvitation } from "./helpers";

async function invitedPlayers(browser: Browser) {
  const hostContext = await browser.newContext();
  const guestContext = await browser.newContext();
  const host = await hostContext.newPage();
  const guest = await guestContext.newPage();
  await host.goto("/");
  const invitation = await createInvitation(host);
  await guest.goto(invitation);
  await guest.getByRole("button", { name: "Join match" }).click();
  return { hostContext, guestContext, host, guest };
}

test("players can decline then accept a draw offer", async ({ browser }) => {
  const { hostContext, guestContext, host, guest } = await invitedPlayers(browser);
  await host.getByRole("button", { name: "Offer draw" }).click();
  await expect(guest.getByText("Draw offer from White")).toBeVisible();
  await guest.getByRole("button", { name: "Decline draw" }).click();
  await expect(host.getByText("Draw declined")).toBeVisible();
  await expect(host.getByText("White to move")).toBeVisible();

  await host.getByRole("button", { name: "Offer draw" }).click();
  await guest.getByRole("button", { name: "Accept draw" }).click();
  await expect(host.getByText("Game ended: 1/2-1/2")).toBeVisible();
  await expect(guest.getByText("Game ended: 1/2-1/2")).toBeVisible();
  await hostContext.close();
  await guestContext.close();
});

test("resigning awards the game to the opponent", async ({ browser }) => {
  const { hostContext, guestContext, host, guest } = await invitedPlayers(browser);
  await host.getByRole("button", { name: "Resign" }).click();
  await expect(host.getByText("Game ended: 0-1")).toBeVisible();
  await expect(guest.getByText("Game ended: 0-1")).toBeVisible();
  await hostContext.close();
  await guestContext.close();
});

test("a player can claim a threefold repetition draw", async ({ browser }) => {
  const { hostContext, guestContext, host, guest } = await invitedPlayers(browser);
  const play = async (page: typeof host, from: string, to: string) => {
    await page.getByRole("gridcell", { name: new RegExp(`${from}.*(knight|king|rook|bishop|queen|pawn)`) }).click();
    await page.getByRole("gridcell", { name: new RegExp(`${to}.*(empty|knight|king|rook|bishop|queen|pawn)`) }).click();
  };
  for (let repeat = 0; repeat < 2; repeat += 1) {
    await play(host, "g1", "f3");
    await play(guest, "g8", "f6");
    await play(host, "f3", "g1");
    await play(guest, "f6", "g8");
  }
  await expect(host.getByRole("button", { name: "Claim threefold repetition draw" })).toBeVisible();
  await host.getByRole("button", { name: "Claim threefold repetition draw" }).click();
  await expect(host.getByText("Game ended: 1/2-1/2")).toBeVisible();
  await expect(guest.getByText("Game ended: 1/2-1/2")).toBeVisible();
  await hostContext.close();
  await guestContext.close();
});
