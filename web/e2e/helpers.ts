import type { Page } from "@playwright/test";

export async function createInvitation(page: Page): Promise<string> {
  const responsePromise = page.waitForResponse((response) =>
    response.url().endsWith("/matches") && response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Create match" }).click();
  const response = await responsePromise;
  const body = await response.json() as { invite_url?: string };
  if (!response.ok() || !body.invite_url) throw new Error("host did not receive an invitation link");
  return body.invite_url;
}
