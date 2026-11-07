import { useEffect, useMemo, useState } from "react";
import { createMatch, joinMatch, readMatch, type MatchResponse } from "./matchClient";
import { GameView } from "../game/GameView";

const INITIAL_PIECES = [
  ["♜", "♞", "♝", "♛", "♚", "♝", "♞", "♜"],
  ["♟", "♟", "♟", "♟", "♟", "♟", "♟", "♟"],
  ["", "", "", "", "", "", "", ""],
  ["", "", "", "", "", "", "", ""],
  ["", "", "", "", "", "", "", ""],
  ["", "", "", "", "", "", "", ""],
  ["♙", "♙", "♙", "♙", "♙", "♙", "♙", "♙"],
  ["♖", "♘", "♗", "♕", "♔", "♗", "♘", "♖"],
];

function tokenFromLocation() {
  return new URLSearchParams(window.location.hash.slice(1)).get("token") ?? "";
}

function formatClock(milliseconds: number) {
  const totalSeconds = Math.max(0, Math.floor(milliseconds / 1000));
  return `${Math.floor(totalSeconds / 60)}:${String(totalSeconds % 60).padStart(2, "0")}`;
}

function StartingBoard() {
  return (
    <div className="chess-board" role="grid" aria-label="Chess board">
      {INITIAL_PIECES.flatMap((rank, row) =>
        rank.map((piece, column) => (
          <div
            className={`board-square ${(row + column) % 2 ? "dark" : "light"}`}
            role="gridcell"
            aria-label={`${String.fromCharCode(97 + column)}${8 - row}${piece ? ` ${piece}` : " empty"}`}
            key={`${row}-${column}`}
          >
            {piece}
          </div>
        )),
      )}
    </div>
  );
}

function ActiveMatch({ match, recoveryCredential }: { match: MatchResponse; recoveryCredential: string }) {
  return <GameView match={match} recoveryCredential={recoveryCredential} />;
}

export function MatchLobby() {
  const [timeControl, setTimeControl] = useState<MatchResponse["time_control"]>("5+0");
  const [match, setMatch] = useState<MatchResponse | null>(null);
  const [inviteURL, setInviteURL] = useState("");
  const [recoveryCredential, setRecoveryCredential] = useState("");
  const [error, setError] = useState("");
  const inviteToken = useMemo(tokenFromLocation, []);

  useEffect(() => {
    if (match?.status !== "waiting" || !recoveryCredential) return;
    let mounted = true;
    const refresh = async () => {
      try {
        const latest = await readMatch(match.match_id, recoveryCredential);
        if (mounted && latest.status !== "waiting") setMatch(latest);
      } catch {
        // A temporary refresh failure does not invalidate the active invitation.
      }
    };
    const timer = window.setInterval(() => void refresh(), 500);
    return () => {
      mounted = false;
      window.clearInterval(timer);
    };
  }, [match?.match_id, match?.status, recoveryCredential]);

  async function handleCreate() {
    setError("");
    try {
      const created = await createMatch(timeControl);
      setMatch(created);
      setInviteURL(created.invite_url ?? "");
      setRecoveryCredential(created.recovery_credential);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Unable to create match.");
    }
  }

  async function handleJoin() {
    setError("");
    try {
      const joined = await joinMatch(inviteToken);
      setMatch(joined);
      setRecoveryCredential(joined.recovery_credential);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Unable to join match.");
    }
  }

  if (match?.status === "active") return <ActiveMatch match={match} recoveryCredential={recoveryCredential} />;

  return (
    <section className="match-lobby" aria-labelledby="lobby-title">
      <h1 id="lobby-title">Online Chess</h1>
      {inviteToken ? (
        <button type="button" onClick={handleJoin}>Join match</button>
      ) : (
        <div className="create-match">
          <label htmlFor="time-control">Time control</label>
          <select
            id="time-control"
            value={timeControl}
            onChange={(event) => setTimeControl(event.target.value as MatchResponse["time_control"])}
          >
            <option value="3+2">3 minutes + 2 seconds (3+2)</option>
            <option value="5+0">5 minutes, no increment (5+0)</option>
            <option value="10+0">10 minutes, no increment (10+0)</option>
          </select>
          <button type="button" onClick={handleCreate}>Create match</button>
        </div>
      )}
      {match?.status === "waiting" && (
        <div className="waiting-match">
          <p>Waiting for an opponent</p>
          <a href={inviteURL}>Invitation link</a>
          <p>White {formatClock(match.clock.white_remaining_ms)}</p>
          <StartingBoard />
        </div>
      )}
      {error && <p role="alert">{error}</p>}
    </section>
  );
}
