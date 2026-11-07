import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App } from "../../App";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  window.sessionStorage.clear();
  window.history.replaceState(null, "", "/");
});

const activeMatch = {
  match_id: "match-1",
  status: "active",
  color: "white",
  position: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
  version: 1,
  time_control: "5+0",
  eligible_draws: ["ThreefoldRepetition"],
  moves: [{ sequence: 1, color: "white", from: "e2", to: "e4", notation: "e4" }],
  recovery_credential: "white-secret",
  clock: { white_remaining_ms: 300000, black_remaining_ms: 300000, active_color: "white", as_of: "2026-11-07T12:00:00Z" },
};

describe("match recovery and actions", () => {
  it("restores a guest seat from session storage after a page reload", async () => {
    window.sessionStorage.setItem("online-chess-seat", JSON.stringify({ match_id: "match-1", recovery_credential: "white-secret" }));
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => activeMatch,
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);

    expect(await screen.findByText("You", { exact: true })).toBeInTheDocument();
    expect(document.querySelector(".self-strip .player-identity strong")).toHaveTextContent("White");
    expect(await screen.findByRole("region", { name: "Move history" })).toHaveTextContent("e4");
    expect(fetchMock).toHaveBeenCalledWith(
      "/matches/match-1/recover",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ recovery_credential: "white-secret", known_version: 0 }),
      }),
    );
  });

  it("shows an empty history when recovery contains no accepted moves", async () => {
    window.sessionStorage.setItem("online-chess-seat", JSON.stringify({ match_id: "match-1", recovery_credential: "white-secret" }));
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, json: async () => ({ ...activeMatch, moves: [] }) }));
    render(<App />);
    expect(await screen.findByText("No moves yet")).toBeInTheDocument();
  });

  it("offers resign and draw actions in an active match", async () => {
    const user = userEvent.setup();
    window.history.replaceState(null, "", "/#token=invite-token");
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => activeMatch });
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);
    await user.click(screen.getByRole("button", { name: /join match/i }));

    expect(await screen.findByRole("button", { name: /offer draw/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /resign/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /claim threefold repetition draw/i })).toBeInTheDocument();
  });
});
