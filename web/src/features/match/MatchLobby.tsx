import { useEffect, useMemo, useState } from "react";
import { createMatch, joinMatch, readMatch, recoverMatch, type MatchResponse } from "./matchClient";
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
  const [copyFeedback, setCopyFeedback] = useState<"success" | "failure" | "">("");
  const [savedSeat] = useState(() => {
    try {
      const raw = window.sessionStorage.getItem("online-chess-seat");
      return raw ? JSON.parse(raw) as { match_id: string; recovery_credential: string } : null;
    } catch {
      return null;
    }
  });
  const [recoveryCredential, setRecoveryCredential] = useState(savedSeat?.recovery_credential ?? "");
  const [isRestoring, setIsRestoring] = useState(Boolean(savedSeat));
  const [error, setError] = useState("");
  const inviteToken = useMemo(tokenFromLocation, []);

  useEffect(() => {
    if (!savedSeat) return;
    let mounted = true;
    void recoverMatch(savedSeat.match_id, savedSeat.recovery_credential).then((restored) => {
      if (mounted) setMatch(restored);
    }).catch((reason: unknown) => {
      if (mounted) setError(reason instanceof Error ? reason.message : "Unable to recover match.");
    }).finally(() => {
      if (mounted) setIsRestoring(false);
    });
    return () => { mounted = false; };
  }, [savedSeat]);

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
    setCopyFeedback("");
    try {
      const created = await createMatch(timeControl);
      setMatch(created);
      setInviteURL(created.invite_url ?? "");
      setRecoveryCredential(created.recovery_credential);
      window.sessionStorage.setItem("online-chess-seat", JSON.stringify({
        match_id: created.match_id,
        recovery_credential: created.recovery_credential,
      }));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Unable to create match.");
    }
  }

  async function handleCopyInvitation() {
    setCopyFeedback("");
    try {
      if (!navigator.clipboard?.writeText) throw new Error("Clipboard unavailable");
      await navigator.clipboard.writeText(inviteURL);
      setCopyFeedback("success");
    } catch {
      setCopyFeedback("failure");
    }
  }

  async function handleJoin() {
    setError("");
    try {
      const joined = await joinMatch(inviteToken);
      setMatch(joined);
      setRecoveryCredential(joined.recovery_credential);
      window.sessionStorage.setItem("online-chess-seat", JSON.stringify({
        match_id: joined.match_id,
        recovery_credential: joined.recovery_credential,
      }));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Unable to join match.");
    }
  }

  if (match?.status === "active") return <ActiveMatch match={match} recoveryCredential={recoveryCredential} />;

  if (isRestoring) return <section className="match-lobby" aria-live="polite">Recovering match…</section>;

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
          {inviteURL && (
            <div className="invitation-copy">
              <button type="button" onClick={handleCopyInvitation}>Copy invitation link</button>
              {copyFeedback === "success" && <p role="status">Invitation link copied.</p>}
              {copyFeedback === "failure" && <p role="alert">Could not copy invitation link. Check clipboard permissions and try again.</p>}
            </div>
          )}
          <p>White {formatClock(match.clock.white_remaining_ms)}</p>
          <StartingBoard />
        </div>
      )}
      {error && <p role="alert">{error}</p>}
    </section>
  );
}
