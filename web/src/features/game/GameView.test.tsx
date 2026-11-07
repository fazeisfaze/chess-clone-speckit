import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import type { MatchResponse } from "../match/matchClient";
import { GameView } from "./GameView";

afterEach(cleanup);

vi.mock("./useMatchSocket", () => ({
  useMatchSocket: () => ({
    connected: true,
    sendMove: vi.fn(),
    state: {
      matchId: "match-1",
      status: "active",
      color: "white",
      position: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
      moves: [
        { sequence: 1, color: "white", from: "e2", to: "e4", notation: "e4" },
        { sequence: 2, color: "black", from: "e7", to: "e5", notation: "e5" },
      ],
      currentTurn: "white",
      version: 1,
      timeControl: "5+0",
      inCheck: false,
      eligibleDraws: [],
      clock: { white_remaining_ms: 300000, black_remaining_ms: 300000, active_color: "white" },
      result: "",
      drawOffer: null,
      notice: "",
      opponentConnected: true,
      error: "",
      syncRequired: false,
    },
  }),
}));

const match: MatchResponse = {
  match_id: "match-1",
  status: "active",
  color: "white",
  position: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
  version: 1,
  time_control: "5+0",
  recovery_credential: "white-secret",
  clock: { white_remaining_ms: 300000, black_remaining_ms: 300000, active_color: "white", as_of: "2026-11-07T12:00:00Z" },
};

describe("game screen layout", () => {
  it("presents the board alongside a labeled control sidebar", () => {
    render(<GameView match={match} recoveryCredential="white-secret" />);
    expect(screen.getByRole("heading", { name: "Online Chess" })).toBeInTheDocument();
    expect(screen.getByRole("complementary", { name: "Match controls" })).toBeInTheDocument();
    expect(screen.getByRole("grid", { name: "Chess board" })).toBeInTheDocument();
    expect(screen.getByText("White to move")).toBeInTheDocument();
    expect(screen.getByText("White to move")).toHaveAttribute("aria-live", "polite");
    expect(screen.getByLabelText("Match clocks")).toBeInTheDocument();
    expect(screen.getByText("Opponent")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Offer draw" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Resign" })).toBeInTheDocument();
  });

  it("shows accepted moves in numbered order with color and notation", () => {
    const { getByRole } = render(<GameView match={match} recoveryCredential="white-secret" />);
    const history = getByRole("region", { name: "Move history" });
    expect(history).toHaveTextContent("1");
    expect(history).toHaveTextContent("White");
    expect(history).toHaveTextContent("e4");
    expect(history).toHaveTextContent("Black");
    expect(history).toHaveTextContent("e5");
    expect(history.querySelectorAll("li")).toHaveLength(2);
  });
});
