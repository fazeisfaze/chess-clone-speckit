import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GameBoard } from "./GameBoard";
import { ClockDisplay } from "./ClockDisplay";

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

const START = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1";

describe("interactive chessboard", () => {
  it("submits a selected legal move and exposes the current turn", async () => {
    const user = userEvent.setup();
    const onMove = vi.fn();
    render(
      <GameBoard
        position={START}
        currentTurn="white"
        viewColor="white"
        inCheck={false}
        clock={{ white_remaining_ms: 180000, black_remaining_ms: 180000, active_color: "white" }}
        onMove={onMove}
      />,
    );

    expect(screen.getByText(/white to move/i)).toBeInTheDocument();
    expect(screen.getByText("White 3:00")).toBeInTheDocument();
    await user.click(screen.getByRole("gridcell", { name: /e2/ }));
    await user.click(screen.getByRole("gridcell", { name: /e4/ }));
    expect(onMove).toHaveBeenCalledWith("e2", "e4");
  });

  it("requires a promotion choice when a pawn reaches its last rank", async () => {
    const user = userEvent.setup();
    const onMove = vi.fn();
    render(
      <GameBoard
        position="4k3/P7/8/8/8/8/8/4K3 w - - 0 1"
        currentTurn="white"
        viewColor="white"
        inCheck={false}
        clock={{ white_remaining_ms: 180000, black_remaining_ms: 180000, active_color: "white" }}
        onMove={onMove}
      />,
    );
    await user.click(screen.getByRole("gridcell", { name: /a7/ }));
    await user.click(screen.getByRole("gridcell", { name: /a8/ }));
    expect(screen.getByRole("dialog", { name: /choose promotion/i })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Knight" }));
    expect(onMove).toHaveBeenCalledWith("a7", "a8", "n");
  });

  it("estimates display time from the received server timestamp instead of local clock skew", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-11-07T12:01:00Z"));
    render(
      <ClockDisplay clock={{
        white_remaining_ms: 180000,
        black_remaining_ms: 180000,
        active_color: "white",
        as_of: "2026-11-07T12:00:00Z",
      }} />,
    );
    expect(screen.getByLabelText("White clock")).toHaveTextContent("White 3:00");
    act(() => vi.advanceTimersByTime(1000));
    expect(screen.getByLabelText("White clock")).toHaveTextContent("White 2:59");
  });

  it("announces when the player to move is in check", () => {
    render(
      <GameBoard
        position={START}
        currentTurn="black"
        viewColor="black"
        inCheck
        clock={{ white_remaining_ms: 180000, black_remaining_ms: 180000, active_color: "black" }}
        onMove={() => undefined}
      />,
    );
    expect(screen.getByText("Check")).toBeInTheDocument();
  });

  it("styles each piece by side rather than inheriting the square color", () => {
    render(
      <GameBoard
        position={START}
        currentTurn="white"
        viewColor="white"
        inCheck={false}
        clock={{ white_remaining_ms: 180000, black_remaining_ms: 180000, active_color: "white" }}
        onMove={() => undefined}
      />,
    );
    const whitePiece = screen.getByRole("gridcell", { name: "a2 white pawn" });
    const blackPiece = screen.getByRole("gridcell", { name: "a7 black pawn" });
    expect(whitePiece.querySelector(".piece-white")).not.toBeNull();
    expect(blackPiece.querySelector(".piece-black")).not.toBeNull();
  });
});
