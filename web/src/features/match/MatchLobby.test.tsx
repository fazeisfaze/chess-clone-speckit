import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App } from "../../App";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  Reflect.deleteProperty(navigator, "clipboard");
  window.sessionStorage.clear();
  window.history.replaceState(null, "", "/");
});

describe("online match lobby", () => {
  it("copies the host invitation without leaving the waiting lobby", async () => {
    const user = userEvent.setup();
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          match_id: "match-1",
          status: "waiting",
          color: "white",
          time_control: "3+2",
          invite_url: "http://localhost:5173/#token=invite-token",
          recovery_credential: "seat-secret",
          version: 0,
          clock: { white_remaining_ms: 180000, black_remaining_ms: 180000, active_color: null },
        }),
        { status: 201, headers: { "Content-Type": "application/json" } },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);
    expect(screen.queryByRole("button", { name: /copy invitation link/i })).not.toBeInTheDocument();
    const preset = screen.getByRole("combobox", { name: /time control/i });
    expect(preset).toHaveDisplayValue(/5 minutes/);
    await user.selectOptions(preset, "3+2");
    await user.click(screen.getByRole("button", { name: /create match/i }));

    expect(fetchMock).toHaveBeenCalledWith(
      "/matches",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ time_control: "3+2" }),
      }),
    );
    const copyButton = await screen.findByRole("button", { name: /copy invitation link/i });
    await user.click(copyButton);
    expect(await screen.findByRole("status")).toHaveTextContent(/copied/i);
    expect(writeText).toHaveBeenCalledWith("http://localhost:5173/#token=invite-token");
    expect(copyButton).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /invitation link/i })).not.toBeInTheDocument();
    expect(screen.getByText(/waiting for an opponent/i)).toBeInTheDocument();
  });

  it("reports clipboard failure without announcing a successful copy", async () => {
    const user = userEvent.setup();
    const writeText = vi.fn().mockRejectedValue(new Error("clipboard denied"));
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
      new Response(JSON.stringify({
        match_id: "match-1", status: "waiting", color: "white", time_control: "5+0",
        invite_url: "https://example.test/#token=secret", recovery_credential: "seat-secret",
        version: 0, clock: { white_remaining_ms: 300000, black_remaining_ms: 300000, active_color: null },
      }), { status: 201, headers: { "Content-Type": "application/json" } }),
    ));

    render(<App />);
    await user.click(screen.getByRole("button", { name: /create match/i }));
    await user.click(await screen.findByRole("button", { name: /copy invitation link/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent(/could not copy/i);
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it("joins an invitation as Black and displays the same starting board and active clock", async () => {
    const user = userEvent.setup();
    window.history.replaceState(null, "", "/#token=invite-token");
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          match_id: "match-1",
          status: "active",
          color: "black",
          position: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
          time_control: "5+0",
          version: 0,
          recovery_credential: "black-secret",
          clock: { white_remaining_ms: 300000, black_remaining_ms: 300000, active_color: "white" },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const { container } = render(<App />);
    await user.click(screen.getByRole("button", { name: /join match/i }));

    expect(fetchMock).toHaveBeenCalledWith(
      "/matches/join",
      expect.objectContaining({ body: JSON.stringify({ invite_token: "invite-token" }) }),
    );
    expect(await screen.findByText("You", { exact: true })).toBeInTheDocument();
    expect(container.querySelector(".self-strip .player-identity strong")).toHaveTextContent("Black");
    expect(screen.getByRole("grid", { name: /chess board/i })).toBeInTheDocument();
    expect(screen.getByLabelText("White clock")).toHaveTextContent(/White\s*5:00/);
    expect(screen.getByLabelText("Black clock")).toHaveTextContent(/Black\s*5:00/);
  });
});
