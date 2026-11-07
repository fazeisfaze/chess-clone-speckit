import { expect, test } from "@playwright/test";

test("two guests create and join an invitation match", async ({ browser }) => {
  const hostContext = await browser.newContext();
  const guestContext = await browser.newContext();
  const host = await hostContext.newPage();
  const guest = await guestContext.newPage();

  await host.goto("/");
  await host.getByLabel("Time control").selectOption("5+0");
  await host.getByRole("button", { name: "Create match" }).click();
  const invitation = host.getByRole("link", { name: "Invitation link" });
  await expect(invitation).toBeVisible();
  const href = await invitation.getAttribute("href");
  if (!href) throw new Error("host did not receive an invitation link");

  await guest.goto(href);
  await guest.getByRole("button", { name: "Join match" }).click();
  await expect(host.getByText("Waiting for an opponent")).toHaveCount(0);
  await expect(host.getByRole("grid", { name: "Chess board" })).toBeVisible();
  await expect(guest.getByRole("grid", { name: "Chess board" })).toBeVisible();
  await expect(host.getByText("You are playing White")).toBeVisible();
  await expect(guest.getByText("You are playing Black")).toBeVisible();
  await expect(host.getByText(/^White [45]:\d{2}$/)).toBeVisible();
  await expect(guest.getByText(/^White [45]:\d{2}$/)).toBeVisible();

  await hostContext.close();
  await guestContext.close();
});
