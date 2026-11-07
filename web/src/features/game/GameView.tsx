import type { MatchResponse } from "../match/matchClient";
import { GameBoard } from "./GameBoard";
import { useMatchSocket } from "./useMatchSocket";

export function GameView({ match, recoveryCredential }: { match: MatchResponse; recoveryCredential: string }) {
  const { state, connected, sendMove } = useMatchSocket(match, recoveryCredential);
  return (
    <section className="game-view" aria-label="Online match">
      <p className="connection-status" aria-live="polite">{connected ? "Connected" : "Connecting"}</p>
      <p className="player-color">You are playing {state.color === "white" ? "White" : "Black"}</p>
      <GameBoard
        position={state.position}
        currentTurn={state.currentTurn}
        viewColor={state.color}
        inCheck={state.inCheck}
        clock={state.clock}
        onMove={(from, to, promotion) => void sendMove(from, to, promotion)}
      />
      {state.error && <p role="alert">{state.error}</p>}
      {state.result && <p className="game-result" role="status">Game ended: {state.result}</p>}
    </section>
  );
}
