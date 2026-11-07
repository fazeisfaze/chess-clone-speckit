import type { MatchResponse } from "../match/matchClient";
import { GameBoard } from "./GameBoard";
import { ClockDisplay } from "./ClockDisplay";
import { GameActions } from "./GameActions";
import { useMatchSocket } from "./useMatchSocket";

export function GameView({ match, recoveryCredential }: { match: MatchResponse; recoveryCredential: string }) {
  const { state, connected, sendMove } = useMatchSocket(match, recoveryCredential);
  const opponentColor = state.color === "white" ? "black" : "white";
  const latestMove = state.moves.at(-1) ?? null;
  return (
    <section className="game-view" aria-label="Online match">
      <header className="game-header">
        <div className="brand-lockup">
          <span className="brand-icon" aria-hidden="true">♞</span>
          <div>
            <p className="eyebrow">ONLINE MATCH</p>
            <h1>Online Chess</h1>
          </div>
        </div>
        <div className="connection-stack">
          <span className={`connection-pill ${connected ? "online" : "connecting"}`} aria-live="polite">
            <span className="connection-dot" />{connected ? "Connected" : "Connecting"}
          </span>
          <span className="opponent-status" aria-live="polite">
            Opponent {state.opponentConnected ? "connected" : "disconnected"}
          </span>
        </div>
      </header>

      <div className="game-layout">
        <section className="board-column" aria-label="Chess board and players">
          <div className={`player-strip opponent-strip ${state.currentTurn === opponentColor ? "active-player" : ""}`}>
            <span className={`player-avatar ${opponentColor}`} aria-hidden="true">{opponentColor === "white" ? "♙" : "♟"}</span>
            <span className="player-identity"><span>Opponent</span><strong>{opponentColor === "white" ? "White" : "Black"}</strong></span>
            <span className="player-presence">{state.opponentConnected ? "Online" : "Offline"}</span>
          </div>
          <GameBoard
            position={state.position}
            currentTurn={state.currentTurn}
            viewColor={state.color}
            inCheck={state.inCheck}
            lastMove={latestMove}
            onMove={(from, to, promotion) => void sendMove(from, to, promotion)}
          />
          <div className={`player-strip self-strip ${state.currentTurn === state.color ? "active-player" : ""}`}>
            <span className={`player-avatar ${state.color}`} aria-hidden="true">{state.color === "white" ? "♙" : "♟"}</span>
            <span className="player-identity"><span>You</span><strong>{state.color === "white" ? "White" : "Black"}</strong></span>
            <span className="player-presence">{connected ? "Online" : "Reconnecting"}</span>
          </div>
        </section>

        <aside className="game-sidebar" aria-label="Match controls">
          <section className={`turn-card ${state.inCheck ? "in-check" : ""}`}>
            <p className="eyebrow">{state.currentTurn === state.color ? "YOUR TURN" : "OPPONENT'S TURN"}</p>
            <h2 aria-live="polite" aria-atomic="true">{state.currentTurn === "white" ? "White" : "Black"} to move</h2>
            <p>{state.inCheck ? "King in check — respond with a legal move." : "Choose a piece and select its destination."}</p>
            {state.inCheck && <span className="check-badge" role="status">Check</span>}
          </section>

          <section className="clock-panel" aria-label="Time control">
            <div className="panel-heading">
              <div><p className="eyebrow">TIME CONTROL</p><h2>{state.timeControl}</h2></div>
              <span className="time-control-type">{state.timeControl.startsWith("3+") ? "BLITZ" : "RAPID"}</span>
            </div>
            <ClockDisplay clock={state.clock} />
          </section>

          <GameActions
            matchID={state.matchId}
            color={state.color}
            recoveryCredential={recoveryCredential}
            drawOffer={state.drawOffer}
            eligibleDraws={state.eligibleDraws}
          />
          <section className="move-history" aria-label="Move history">
            <div className="panel-heading">
              <div><p className="eyebrow">MATCH RECORD</p><h2>Move history</h2></div>
              <span className="move-count">{state.moves.length}</span>
            </div>
            {state.moves.length === 0 ? (
              <p className="move-history-empty">No moves yet</p>
            ) : (
              <ol className="move-history-list">
                {state.moves.map((move) => (
                  <li key={move.sequence} className={move.sequence === latestMove?.sequence ? "latest-move" : undefined}>
                    <span className="move-number">{Math.ceil(move.sequence / 2)}.</span>
                    <span className={`move-color ${move.color}`}>{move.color === "white" ? "White" : "Black"}</span>
                    <strong>{move.notation}</strong>
                  </li>
                ))}
              </ol>
            )}
          </section>
          {state.notice && <p className="action-notice" role="status">{state.notice}</p>}
          {state.error && <p className="error-notice" role="alert">{state.error}</p>}
          {state.result && <p className="game-result" role="status">Game ended: {state.result}</p>}
        </aside>
      </div>
    </section>
  );
}
