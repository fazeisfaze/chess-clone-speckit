import { useState, type KeyboardEvent } from "react";
import type { MoveHistoryEntry } from "../match/matchClient";

export type GameBoardProps = {
  position: string;
  currentTurn: "white" | "black";
  viewColor: "white" | "black";
  inCheck: boolean;
  lastMove?: MoveHistoryEntry | null;
  onMove: (from: string, to: string, promotion?: string) => void;
};

const PIECES: Record<string, { color: "white" | "black"; name: string; glyph: string }> = {
  K: { color: "white", name: "white king", glyph: "♔" },
  Q: { color: "white", name: "white queen", glyph: "♕" },
  R: { color: "white", name: "white rook", glyph: "♖" },
  B: { color: "white", name: "white bishop", glyph: "♗" },
  N: { color: "white", name: "white knight", glyph: "♘" },
  P: { color: "white", name: "white pawn", glyph: "♙" },
  k: { color: "black", name: "black king", glyph: "♚" },
  q: { color: "black", name: "black queen", glyph: "♛" },
  r: { color: "black", name: "black rook", glyph: "♜" },
  b: { color: "black", name: "black bishop", glyph: "♝" },
  n: { color: "black", name: "black knight", glyph: "♞" },
  p: { color: "black", name: "black pawn", glyph: "♟" },
};

function parsePosition(fen: string) {
  const rows = fen.split(" ")[0]?.split("/") ?? [];
  const pieces = new Map<string, string>();
  rows.forEach((row, rowIndex) => {
    let file = 0;
    for (const char of row) {
      if (/\d/.test(char)) {
        file += Number(char);
      } else {
        pieces.set(`${String.fromCharCode(97 + file)}${8 - rowIndex}`, char);
        file += 1;
      }
    }
  });
  return pieces;
}

export function GameBoard({ position, currentTurn, viewColor, inCheck, lastMove, onMove }: GameBoardProps) {
  const [selected, setSelected] = useState<string | null>(null);
  const [promotion, setPromotion] = useState<{ from: string; to: string } | null>(null);
  const pieces = parsePosition(position);
  const files = viewColor === "white" ? "abcdefgh" : "hgfedcba";
  const ranks = viewColor === "white" ? "87654321" : "12345678";
  const checkedKing = inCheck
    ? [...pieces.entries()].find(([, piece]) => piece === (currentTurn === "white" ? "K" : "k"))?.[0]
    : undefined;

  function chooseSquare(square: string) {
    const piece = pieces.get(square);
    if (!selected) {
      if (piece && PIECES[piece].color === viewColor && currentTurn === viewColor) setSelected(square);
      return;
    }
    if (square === selected) {
      setSelected(null);
      return;
    }
    if (piece && PIECES[piece].color === viewColor) {
      setSelected(square);
      return;
    }
    const movingPiece = pieces.get(selected);
    const lastRank = viewColor === "white" ? "8" : "1";
    if (movingPiece?.toLowerCase() === "p" && square[1] === lastRank) {
      setPromotion({ from: selected, to: square });
      setSelected(null);
      return;
    }
    const from = selected;
    setSelected(null);
    onMove(from, square);
  }

  function moveGridFocus(event: KeyboardEvent<HTMLButtonElement>) {
    const deltas: Record<string, number> = { ArrowRight: 1, ArrowLeft: -1, ArrowDown: 8, ArrowUp: -8 };
    const delta = deltas[event.key];
    if (delta === undefined) return;
    const grid = event.currentTarget.parentElement;
    const cells = grid ? Array.from(grid.querySelectorAll<HTMLButtonElement>('[role="gridcell"]')) : [];
    const currentIndex = cells.indexOf(event.currentTarget);
    const nextIndex = currentIndex + delta;
    if (nextIndex < 0 || nextIndex >= cells.length) return;
    if ((event.key === "ArrowRight" || event.key === "ArrowLeft") && Math.floor(currentIndex / 8) !== Math.floor(nextIndex / 8)) return;
    event.preventDefault();
    cells[nextIndex]?.focus();
  }

  return (
    <section className="game-board-panel" aria-label="Chessboard">
      {inCheck && <p className="check-status" role="status">Check</p>}
      <div className="chess-board" role="grid" aria-label="Chess board">
        {ranks.split("").flatMap((rank, row) =>
          files.split("").map((file, column) => {
            const square = `${file}${rank}`;
            const code = pieces.get(square);
            const piece = code ? PIECES[code] : undefined;
            const squareColor = (row + column) % 2 ? "dark" : "light";
            const lastMoveClass = lastMove?.from === square ? " last-move-from" : lastMove?.to === square ? " last-move-to" : "";
            return (
              <button
                type="button"
                role="gridcell"
                className={`board-square ${squareColor}${selected === square ? " selected" : ""}${checkedKing === square ? " checked-king" : ""}${lastMoveClass}`}
                aria-label={`${square} ${piece?.name ?? "empty"}${checkedKing === square ? ", in check" : ""}`}
                aria-selected={selected === square}
                key={square}
                onClick={() => chooseSquare(square)}
                onKeyDown={moveGridFocus}
              >
                {piece && <span className={`piece-${piece.color}`} aria-hidden="true">{piece.glyph}</span>}
              </button>
            );
          }),
        )}
      </div>
      {promotion && (
        <div role="dialog" aria-modal="true" aria-label="Choose promotion" className="promotion-dialog">
          <p>Choose promotion</p>
          {([ ["q", "Queen"], ["r", "Rook"], ["b", "Bishop"], ["n", "Knight"] ] as const).map(([code, name]) => (
            <button
              type="button"
              key={code}
              onClick={() => {
                onMove(promotion.from, promotion.to, code);
                setPromotion(null);
              }}
            >
              {name}
            </button>
          ))}
        </div>
      )}
    </section>
  );
}
