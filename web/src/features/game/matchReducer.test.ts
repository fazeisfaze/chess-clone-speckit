import { describe, expect, it } from "vitest";
import { matchReducer, type MatchState } from "./matchReducer";

const initial: MatchState = {
  matchId: "match-1",
  status: "active",
  color: "white",
  position: "start-fen",
  currentTurn: "white",
  version: 1,
  timeControl: "3+2",
  inCheck: false,
  eligibleDraws: [],
  moves: [],
  clock: {
    white_remaining_ms: 180000,
    black_remaining_ms: 180000,
    active_color: "white",
    as_of: "2026-11-07T12:00:00Z",
  },
  result: "",
  drawOffer: null,
  notice: "",
  opponentConnected: false,
  error: "",
  syncRequired: false,
};

describe("match realtime reducer", () => {
  it("applies the next versioned move and its clock snapshot atomically", () => {
    const next = matchReducer(initial, {
      type: "move_accepted",
      version: 2,
      position: "after-e4",
      next_turn: "black",
      clock: {
        white_remaining_ms: 171000,
        black_remaining_ms: 180000,
        active_color: "black",
        as_of: "2026-11-07T12:00:09Z",
      },
      result: "",
      in_check: false,
      eligible_draws: [],
      move: { sequence: 1, color: "white", from: "e2", to: "e4", notation: "e4" },
    });
    expect(next.position).toBe("after-e4");
    expect(next.currentTurn).toBe("black");
    expect(next.clock.white_remaining_ms).toBe(171000);
    expect(next.version).toBe(2);
    expect(next.moves).toEqual([{ sequence: 1, color: "white", from: "e2", to: "e4", notation: "e4" }]);
  });

  it("ignores duplicate events and requests synchronization on a version gap", () => {
    const duplicate = matchReducer(initial, {
      type: "move_accepted", version: 1, position: "stale", next_turn: "black",
      clock: initial.clock, result: "", in_check: false, eligible_draws: [],
      move: { sequence: 1, color: "white", from: "e2", to: "e4", notation: "e4" },
    });
    expect(duplicate).toBe(initial);
    const gap = matchReducer(initial, {
      type: "move_accepted", version: 3, position: "unknown", next_turn: "black",
      clock: initial.clock, result: "", in_check: false, eligible_draws: [],
      move: { sequence: 3, color: "white", from: "d2", to: "d4", notation: "d4" },
    });
    expect(gap.syncRequired).toBe(true);
    expect(gap.position).toBe(initial.position);
    expect(gap.moves).toEqual([]);
  });

  it("records a server rejection without mutating board or clocks", () => {
    const rejected = matchReducer(initial, { type: "action_rejected", message: "illegal move" });
    expect(rejected.error).toBe("illegal move");
    expect(rejected.position).toBe(initial.position);
    expect(rejected.clock).toBe(initial.clock);
    expect(rejected.moves).toEqual([]);
  });

  it("restores an empty history when an older match snapshot omits moves", () => {
    const restored = matchReducer(initial, {
      type: "match_snapshot",
      match_id: "match-1",
      status: "active",
      color: "white",
      position: "start-fen",
      next_turn: "white",
      version: 1,
      time_control: "3+2",
      in_check: false,
      eligible_draws: [],
      clock: initial.clock,
    });
    expect(restored.moves).toEqual([]);
  });

  it("applies a terminal event for the version already applied by the final move", () => {
    const afterFinalMove = { ...initial, position: "final-position", version: 2 };
    const ended = matchReducer(afterFinalMove, {
      type: "game_ended", version: 2, result: "1-0",
      clock: { ...initial.clock, active_color: null },
    });
    expect(ended.status).toBe("completed");
    expect(ended.result).toBe("1-0");
    expect(ended.clock.active_color).toBeNull();
  });

  it("handles a timeout event as a terminal state rather than a board move", () => {
    const ended = matchReducer(initial, {
      type: "clock_expired", version: 2, result: "0-1",
      clock: { ...initial.clock, white_remaining_ms: 0, active_color: null },
    });
    expect(ended.status).toBe("completed");
    expect(ended.result).toBe("0-1");
    expect(ended.clock.white_remaining_ms).toBe(0);
  });

  it("shows an incoming draw offer and clears it when declined", () => {
    const offered = matchReducer(initial, {
      type: "draw_offer", offer_id: "offer-1", offering_color: "black",
    } as never) as MatchState & { drawOffer?: { offer_id: string; offering_color: string } };
    expect(offered.drawOffer).toEqual({ offer_id: "offer-1", offering_color: "black" });
    const declined = matchReducer(offered, {
      type: "draw_resolved", offer_id: "offer-1", status: "declined", version: 1,
    } as never) as MatchState & { drawOffer?: unknown; notice?: string };
    expect(declined.drawOffer).toBeNull();
    expect(declined.notice).toBe("Draw declined");
  });

  it("updates opponent connection state but ignores its own presence event", () => {
    const connected = matchReducer(initial, {
      type: "opponent_connection", color: "black", connected: true,
    } as never) as MatchState & { opponentConnected?: boolean };
    expect(connected.opponentConnected).toBe(true);
    const own = matchReducer(connected, {
      type: "opponent_connection", color: "white", connected: false,
    } as never) as MatchState & { opponentConnected?: boolean };
    expect(own.opponentConnected).toBe(true);
  });
});
