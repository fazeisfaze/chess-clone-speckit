import { useCallback, useEffect, useReducer, useRef, useState } from "react";
import { readMatch, submitMove, type MatchResponse } from "../match/matchClient";
import { matchReducer, type MatchState } from "./matchReducer";

function initialState(match: MatchResponse): MatchState {
  const turn = match.position.split(" ")[1] === "b" ? "black" : "white";
  return {
    matchId: match.match_id,
    status: match.status,
    color: match.color,
    position: match.position,
    currentTurn: turn,
    version: match.version,
    timeControl: match.time_control,
    inCheck: match.in_check ?? false,
    clock: { ...match.clock, client_received_at: Date.now() },
    result: "",
    error: "",
    syncRequired: false,
  };
}

export function useMatchSocket(match: MatchResponse, recoveryCredential: string) {
  const [state, dispatch] = useReducer(matchReducer, match, initialState);
  const [connected, setConnected] = useState(false);
  const socket = useRef<WebSocket | null>(null);

  useEffect(() => {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const connection = new WebSocket(`${protocol}//${window.location.host}/matches/${encodeURIComponent(match.match_id)}/events`);
    socket.current = connection;
    connection.onopen = () => {
      setConnected(true);
      connection.send(JSON.stringify({ type: "authenticate", recovery_credential: recoveryCredential }));
    };
    connection.onmessage = (message) => {
      try {
        const event = JSON.parse(String(message.data));
        if (event.clock?.as_of) event.clock.client_received_at = Date.now();
        if (event.type === "action_rejected") {
          dispatch({ type: "action_rejected", message: event.message ?? event.error ?? "Action not accepted" });
          return;
        }
        dispatch(event);
      } catch {
        dispatch({ type: "action_rejected", message: "Received an invalid match update" });
      }
    };
    connection.onclose = () => setConnected(false);
    connection.onerror = () => setConnected(false);
    return () => {
      socket.current = null;
      connection.close();
    };
  }, [match.match_id, recoveryCredential]);

  useEffect(() => {
    if (!state.syncRequired) return;
    let cancelled = false;
    void readMatch(match.match_id, recoveryCredential).then((snapshot) => {
      if (!cancelled) {
        dispatch({
          type: "match_snapshot",
          match_id: snapshot.match_id,
          status: snapshot.status,
          color: snapshot.color,
          position: snapshot.position,
          next_turn: snapshot.position.split(" ")[1] === "b" ? "black" : "white",
          version: snapshot.version,
          time_control: snapshot.time_control,
          in_check: snapshot.in_check ?? false,
          clock: snapshot.clock,
        });
      }
    }).catch((error: unknown) => {
      if (!cancelled) dispatch({ type: "action_rejected", message: error instanceof Error ? error.message : "Unable to synchronize match" });
    });
    return () => { cancelled = true; };
  }, [state.syncRequired, match.match_id, recoveryCredential]);

  const sendMove = useCallback(async (from: string, to: string, promotion?: string) => {
    const currentSocket = socket.current;
    if (currentSocket?.readyState === WebSocket.OPEN) {
      currentSocket.send(JSON.stringify({ type: "move", expected_version: state.version, from, to, promotion: promotion ?? null }));
      return;
    }
    try {
      const accepted = await submitMove(match.match_id, recoveryCredential, state.version, from, to, promotion);
      dispatch({ type: "move_accepted", ...accepted, result: accepted.result ?? "" });
    } catch (error) {
      dispatch({ type: "action_rejected", message: error instanceof Error ? error.message : "Move was not accepted" });
    }
  }, [match.match_id, recoveryCredential, state.version]);

  return { state, connected, sendMove };
}
