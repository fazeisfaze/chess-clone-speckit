export type ClockSnapshot = {
  white_remaining_ms: number;
  black_remaining_ms: number;
  active_color: "white" | "black" | null;
  as_of: string;
  client_received_at?: number;
};

export type MatchResponse = {
  match_id: string;
  status: "waiting" | "active" | "completed" | "unavailable";
  color: "white" | "black";
  position: string;
  invite_url?: string;
  recovery_credential: string;
  version: number;
  time_control: "3+2" | "5+0" | "10+0";
  clock: ClockSnapshot;
  in_check?: boolean;
  eligible_draws?: string[];
};

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const result = await response.json();
  if (!response.ok) {
    throw new Error(result?.error?.message ?? "The request could not be completed.");
  }
  if (result?.clock?.as_of) result.clock.client_received_at = Date.now();
  return result as T;
}

export function createMatch(timeControl: MatchResponse["time_control"]) {
  return postJSON<MatchResponse>("/matches", { time_control: timeControl });
}

export function joinMatch(inviteToken: string) {
  return postJSON<MatchResponse>("/matches/join", { invite_token: inviteToken });
}

export function recoverMatch(matchID: string, recoveryCredential: string, knownVersion = 0) {
  return postJSON<MatchResponse>(`/matches/${encodeURIComponent(matchID)}/recover`, {
    recovery_credential: recoveryCredential,
    known_version: knownVersion,
  });
}

export async function readMatch(matchID: string, recoveryCredential: string) {
  const response = await fetch(`/matches/${encodeURIComponent(matchID)}`, {
    headers: { Authorization: `Bearer ${recoveryCredential}` },
  });
  const result = await response.json();
  if (!response.ok) {
    throw new Error(result?.error?.message ?? "The match could not be refreshed.");
  }
  if (result?.clock?.as_of) result.clock.client_received_at = Date.now();
  return result as MatchResponse;
}

export type MoveAccepted = {
  type?: "move_accepted";
  version: number;
  position: string;
  next_turn: "white" | "black";
  clock: ClockSnapshot;
  result?: string;
  in_check: boolean;
  eligible_draws: string[];
};

export async function submitMove(
  matchID: string,
  recoveryCredential: string,
  expectedVersion: number,
  from: string,
  to: string,
  promotion?: string,
) {
  const response = await fetch(`/matches/${encodeURIComponent(matchID)}/moves`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${recoveryCredential}`,
    },
    body: JSON.stringify({ expected_version: expectedVersion, from, to, promotion: promotion ?? null }),
  });
  const result = await response.json();
  if (!response.ok) throw new Error(result?.error?.message ?? "Move was not accepted.");
  if (result?.clock?.as_of) result.clock.client_received_at = Date.now();
  return result as MoveAccepted;
}

async function authorizedPost<T>(path: string, recoveryCredential: string, body: unknown = {}) {
  const response = await fetch(path, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${recoveryCredential}`,
    },
    body: JSON.stringify(body),
  });
  const result = await response.json();
  if (!response.ok) throw new Error(result?.error?.message ?? "Match action was not accepted.");
  if (result?.clock?.as_of) result.clock.client_received_at = Date.now();
  return result as T;
}

export function resignMatch(matchID: string, recoveryCredential: string) {
  return authorizedPost<{ status: string; result: string }>(`/matches/${encodeURIComponent(matchID)}/resign`, recoveryCredential);
}

export function offerDraw(matchID: string, recoveryCredential: string) {
  return authorizedPost<{ offer_id: string; status: string; offering_color: "white" | "black" }>(
    `/matches/${encodeURIComponent(matchID)}/draw-offers`, recoveryCredential,
  );
}

export function resolveDraw(matchID: string, offerID: string, recoveryCredential: string, decision: "accept" | "decline") {
  return authorizedPost<{ status: string; result: string }>(
    `/matches/${encodeURIComponent(matchID)}/draw-offers/${encodeURIComponent(offerID)}/resolve`,
    recoveryCredential,
    { decision },
  );
}

export function claimDraw(matchID: string, recoveryCredential: string, method: "ThreefoldRepetition" | "FiftyMoveRule") {
  return authorizedPost<{ status: string; result: string }>(
    `/matches/${encodeURIComponent(matchID)}/draw-claims`, recoveryCredential, { method },
  );
}
