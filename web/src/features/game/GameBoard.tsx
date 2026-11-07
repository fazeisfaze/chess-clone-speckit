import { useState } from "react";
import { ClockDisplay, type ClockView } from "./ClockDisplay";

export type GameBoardProps = {
  position: string;
  currentTurn: "white" | "black";
  viewColor: "white" | "black";
  inCheck: boolean;
  clock: ClockView;
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

export function GameBoard({ position, currentTurn, viewColor, inCheck, clock, onMove }: GameBoardProps) {
  const [selected, setSelected] = useState<string | null>(null);
  const [promotion, setPromotion] = useState<{ from: string; to: string } | null>(null);
  const pieces = parsePosition(position);
  const files = viewColor === "white" ? "abcdefgh" : "hgfedcba";
  const ranks = viewColor === "white" ? "87654321" : "12345678";

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

  return (
    <section className="game-board-panel" aria-label="Chess game">
      <p className="turn-status" aria-live="polite">{currentTurn === "white" ? "White" : "Black"} to move</p>
      {inCheck && <p className="check-status" role="status">Check</p>}
      <ClockDisplay clock={clock} />
      <div className="chess-board" role="grid" aria-label="Chess board">
        {ranks.split("").flatMap((rank, row) =>
          files.split("").map((file, column) => {
            const square = `${file}${rank}`;
            const code = pieces.get(square);
            const piece = code ? PIECES[code] : undefined;
            const squareColor = (row + column) % 2 ? "dark" : "light";
            return (
              <button
                type="button"
                role="gridcell"
                className={`board-square ${squareColor}${selected === square ? " selected" : ""}`}
                aria-label={`${square} ${piece?.name ?? "empty"}`}
                aria-selected={selected === square}
                key={square}
                onClick={() => chooseSquare(square)}
              >
                {piece?.glyph ?? ""}
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
