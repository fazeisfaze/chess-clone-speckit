import { expect, test, type Page } from "@playwright/test";
import { createInvitation } from "./helpers";

type ServerClock = {
  white_remaining_ms: number;
  black_remaining_ms: number;
  active_color: "white" | "black" | null;
  as_of: string;
};

async function serverClock(page: Page): Promise<ServerClock> {
  return page.evaluate(async () => {
    const seat = JSON.parse(sessionStorage.getItem("online-chess-seat") ?? "null");
    const response = await fetch(`/matches/${encodeURIComponent(seat.match_id)}`, {
      headers: { Authorization: `Bearer ${seat.recovery_credential}` },
    });
    if (!response.ok) throw new Error(`clock snapshot failed: ${response.status}`);
    const state = await response.json();
    const receivedAt = Date.now();
    const elapsed = state.clock.active_color ? receivedAt - Date.parse(state.clock.as_of) : 0;
    return {
      white_remaining_ms: state.clock.white_remaining_ms - (state.clock.active_color === "white" ? elapsed : 0),
      black_remaining_ms: state.clock.black_remaining_ms - (state.clock.active_color === "black" ? elapsed : 0),
      active_color: state.clock.active_color,
      as_of: state.clock.as_of,
    };
  });
}

function parseClock(text: string | null) {
  const match = text?.match(/(\d+):(\d{2})/);
  if (!match) throw new Error(`invalid clock display: ${text}`);
  return (Number(match[1]) * 60 + Number(match[2])) * 1000;
}

test("clocks on both clients stay within one second of server time in 20 matches", async ({ browser }) => {
  for (let trial = 0; trial < 20; trial += 1) {
    const hostContext = await browser.newContext();
    const guestContext = await browser.newContext();
    const host = await hostContext.newPage();
    const guest = await guestContext.newPage();
    await host.goto("/");
    const invitation = await createInvitation(host);
    await guest.goto(invitation);
    await guest.getByRole("button", { name: "Join match" }).click();
    await expect(host.getByRole("grid", { name: "Chess board" })).toBeVisible();
    await expect(guest.getByRole("grid", { name: "Chess board" })).toBeVisible();

    const hostClock = await serverClock(host);
    const guestClock = await serverClock(guest);
    const readings = [
      [parseClock(await host.getByLabel("White clock").textContent()), hostClock.white_remaining_ms],
      [parseClock(await host.getByLabel("Black clock").textContent()), hostClock.black_remaining_ms],
      [parseClock(await guest.getByLabel("White clock").textContent()), guestClock.white_remaining_ms],
      [parseClock(await guest.getByLabel("Black clock").textContent()), guestClock.black_remaining_ms],
    ];
    for (const [displayed, authoritative] of readings) {
      expect(Math.abs(displayed - authoritative), `trial ${trial} clock drift`).toBeLessThanOrEqual(1000);
    }
    await hostContext.close();
    await guestContext.close();
  }
});
