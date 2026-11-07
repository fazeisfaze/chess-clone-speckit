import type { MatchResponse } from "../match/matchClient";
import { GameBoard } from "./GameBoard";
import { GameActions } from "./GameActions";
import { useMatchSocket } from "./useMatchSocket";

export function GameView({ match, recoveryCredential }: { match: MatchResponse; recoveryCredential: string }) {
  const { state, connected, sendMove } = useMatchSocket(match, recoveryCredential);
  return (
    <section className="game-view" aria-label="Online match">
      <p className="connection-status" aria-live="polite">{connected ? "Connected" : "Connecting"}</p>
      <p className="player-color">You are playing {state.color === "white" ? "White" : "Black"}</p>
      <p className="opponent-status" aria-live="polite">
        Opponent {state.opponentConnected ? "connected" : "disconnected"}
      </p>
      {state.notice && <p role="status">{state.notice}</p>}
      <GameBoard
        position={state.position}
        currentTurn={state.currentTurn}
        viewColor={state.color}
        inCheck={state.inCheck}
        clock={state.clock}
        onMove={(from, to, promotion) => void sendMove(from, to, promotion)}
      />
      {state.status === "active" && (
        <GameActions
          matchID={state.matchId}
          color={state.color}
          recoveryCredential={recoveryCredential}
          drawOffer={state.drawOffer}
          eligibleDraws={state.eligibleDraws}
        />
      )}
      {state.error && <p role="alert">{state.error}</p>}
      {state.result && <p className="game-result" role="status">Game ended: {state.result}</p>}
    </section>
  );
}
