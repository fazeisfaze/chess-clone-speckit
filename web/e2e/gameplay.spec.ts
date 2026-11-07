import { expect, test } from "@playwright/test";
import { createInvitation } from "./helpers";

test("two players see accepted moves, turns, and clocks converge", async ({ browser }) => {
  const hostContext = await browser.newContext();
  const guestContext = await browser.newContext();
  const host = await hostContext.newPage();
  const guest = await guestContext.newPage();

  await host.goto("/");
  await host.getByLabel("Time control").selectOption("3+2");
  const invitation = await createInvitation(host);
  await guest.goto(invitation);
  await guest.getByRole("button", { name: "Join match" }).click();
  await expect(host.getByText("Connected", { exact: true })).toBeVisible();
  await expect(guest.getByText("Connected", { exact: true })).toBeVisible();

  const squareRects = await host.getByRole("gridcell").evaluateAll((cells) =>
    cells.map((cell) => {
      const rect = cell.getBoundingClientRect();
      return { width: rect.width, height: rect.height };
    }),
  );
  expect(squareRects).toHaveLength(64);
  for (const square of squareRects) {
    expect(Math.abs(square.width - square.height)).toBeLessThan(1);
  }

  const whitePawn = host.getByRole("gridcell", { name: /e2.*white pawn/i });
  await whitePawn.click();
  await host.getByRole("gridcell", { name: /e4.*empty/i }).click();

  await expect(guest.getByRole("gridcell", { name: /e4.*white pawn/i })).toBeVisible();
  for (const page of [host, guest]) {
    await expect(page.getByRole("gridcell", { name: /e2 empty/i })).toHaveClass(/last-move-from/);
    await expect(page.getByRole("gridcell", { name: /e4 white pawn/i })).toHaveClass(/last-move-to/);
    await expect(page.locator(".move-history-list li").last()).toHaveClass(/latest-move/);
  }
  await expect(guest.getByText("Black to move")).toBeVisible();
  await expect(host.getByLabel("White clock")).toHaveText(/^White 3:\d{2}$/);
  await expect(host.getByLabel("Black clock")).toHaveText(/^Black \d:\d{2}$/);
  await expect(guest.getByLabel("White clock")).toHaveText(/^White 3:\d{2}$/);

  await guest.getByRole("gridcell", { name: /e7.*black pawn/i }).click();
  await guest.getByRole("gridcell", { name: /e4.*white pawn/i }).click();
  await expect(guest.getByRole("alert")).toHaveText(/action not accepted|illegal move/i);
  await expect(host.getByRole("gridcell", { name: /e7.*black pawn/i })).toBeVisible();
  await expect(host.getByText("Black to move")).toBeVisible();

  await guest.getByRole("gridcell", { name: /e7.*black pawn/i }).click();
  await guest.getByRole("gridcell", { name: /e5.*empty/i }).click();
  await expect(host.getByRole("gridcell", { name: /e5.*black pawn/i })).toBeVisible();
  for (const page of [host, guest]) {
    await expect(page.getByRole("gridcell", { name: /e7 empty/i })).toHaveClass(/last-move-from/);
    await expect(page.getByRole("gridcell", { name: /e5 black pawn/i })).toHaveClass(/last-move-to/);
    await expect(page.locator(".move-history-list li").last()).toHaveClass(/latest-move/);
  }
  await expect(host.getByText("White to move")).toBeVisible();

  await hostContext.close();
  await guestContext.close();
});
