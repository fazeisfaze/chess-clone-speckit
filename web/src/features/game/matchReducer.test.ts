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
  clock: {
    white_remaining_ms: 180000,
    black_remaining_ms: 180000,
    active_color: "white",
    as_of: "2026-11-07T12:00:00Z",
  },
  result: "",
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
    });
    expect(next.position).toBe("after-e4");
    expect(next.currentTurn).toBe("black");
    expect(next.clock.white_remaining_ms).toBe(171000);
    expect(next.version).toBe(2);
  });

  it("ignores duplicate events and requests synchronization on a version gap", () => {
    const duplicate = matchReducer(initial, {
      type: "move_accepted", version: 1, position: "stale", next_turn: "black",
      clock: initial.clock, result: "", in_check: false,
    });
    expect(duplicate).toBe(initial);
    const gap = matchReducer(initial, {
      type: "move_accepted", version: 3, position: "unknown", next_turn: "black",
      clock: initial.clock, result: "", in_check: false,
    });
    expect(gap.syncRequired).toBe(true);
    expect(gap.position).toBe(initial.position);
  });

  it("records a server rejection without mutating board or clocks", () => {
    const rejected = matchReducer(initial, { type: "action_rejected", message: "illegal move" });
    expect(rejected.error).toBe("illegal move");
    expect(rejected.position).toBe(initial.position);
    expect(rejected.clock).toBe(initial.clock);
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
});
