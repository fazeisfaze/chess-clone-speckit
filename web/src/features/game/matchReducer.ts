import type { ClockView } from "./ClockDisplay";

export type MatchState = {
  matchId: string;
  status: "waiting" | "active" | "completed" | "unavailable";
  color: "white" | "black";
  position: string;
  currentTurn: "white" | "black";
  version: number;
  timeControl: "3+2" | "5+0" | "10+0";
  inCheck: boolean;
  clock: ClockView;
  result: string;
  error: string;
  syncRequired: boolean;
};

type MoveEvent = {
  type: "move_accepted";
  version: number;
  position: string;
  next_turn: "white" | "black";
  clock: ClockView;
  result: string;
  in_check: boolean;
};

type RealtimeEvent = MoveEvent
	| { type: "action_rejected"; message: string }
	| {
		type: "match_snapshot";
		match_id: string;
		status: MatchState["status"];
		color: MatchState["color"];
		position: string;
		next_turn: MatchState["currentTurn"];
		version: number;
		time_control: MatchState["timeControl"];
		in_check: boolean;
		clock: ClockView;
		result?: string;
	}
	| { type: "game_ended"; version: number; result: string; clock: ClockView }
	| { type: "clock_expired"; version: number; result: string; clock: ClockView };

export function matchReducer(state: MatchState, event: RealtimeEvent): MatchState {
	if (event.type === "action_rejected") {
		return { ...state, error: event.message };
	}
	if (event.type === "match_snapshot") {
		return {
			matchId: event.match_id,
			status: event.status,
			color: event.color,
			position: event.position,
			currentTurn: event.next_turn,
			version: event.version,
			timeControl: event.time_control,
			inCheck: event.in_check,
			clock: event.clock,
			result: event.result ?? "",
			error: "",
			syncRequired: false,
		};
	}
	if (event.type === "game_ended" || event.type === "clock_expired") {
		if (event.version < state.version) return state;
		return { ...state, status: "completed", result: event.result, clock: event.clock, version: event.version };
	}
	if (event.version <= state.version) return state;
	if (event.version !== state.version + 1) return { ...state, syncRequired: true };
	return {
		...state,
		position: event.position,
		inCheck: event.in_check,
		currentTurn: event.next_turn,
		version: event.version,
		clock: event.clock,
		result: event.result,
		status: event.result ? "completed" : "active",
		error: "",
		syncRequired: false,
	};
}
