import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GameBoard } from "./GameBoard";

const START = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1";

describe("chessboard keyboard and assistive technology behavior", () => {
  it("moves keyboard focus to the adjacent square and submits a move with Enter", async () => {
    const user = userEvent.setup();
    const onMove = vi.fn();
    render(
      <GameBoard
        position={START}
        currentTurn="white"
        viewColor="white"
        inCheck={false}
        onMove={onMove}
      />,
    );
    const e2 = screen.getByRole("gridcell", { name: "e2 white pawn" });
    e2.focus();
    await user.keyboard("{ArrowRight}");
    const f2 = screen.getByRole("gridcell", { name: "f2 white pawn" });
    expect(f2).toHaveFocus();
    e2.focus();
    await user.keyboard("{Enter}");
    screen.getByRole("gridcell", { name: "e4 empty" }).focus();
    await user.keyboard("{Enter}");
    expect(onMove).toHaveBeenCalledWith("e2", "e4");
  });

  it("announces check state through a live status region", () => {
    render(
      <GameBoard
        position={START}
        currentTurn="black"
        viewColor="black"
        inCheck
        onMove={() => undefined}
      />,
    );
    expect(screen.getByText("Check")).toHaveAttribute("role", "status");
  });
});
