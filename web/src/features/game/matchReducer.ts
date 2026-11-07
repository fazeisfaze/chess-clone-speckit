import type { ClockView } from "./ClockDisplay";
import type { MoveHistoryEntry } from "../match/matchClient";

export type MatchState = {
  matchId: string;
  status: "waiting" | "active" | "completed" | "unavailable";
  color: "white" | "black";
  position: string;
  currentTurn: "white" | "black";
  version: number;
  timeControl: "3+2" | "5+0" | "10+0";
  inCheck: boolean;
  eligibleDraws: string[];
  moves: MoveHistoryEntry[];
  clock: ClockView;
  result: string;
  drawOffer: { offer_id: string; offering_color: "white" | "black" } | null;
  notice: string;
  opponentConnected: boolean;
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
  eligible_draws: string[];
  move: MoveHistoryEntry;
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
		eligible_draws: string[];
		moves?: MoveHistoryEntry[];
		clock: ClockView;
		result?: string;
	}
	| { type: "game_ended"; version: number; result: string; clock: ClockView }
	| { type: "clock_expired"; version: number; result: string; clock: ClockView }
	| { type: "draw_offer"; offer_id: string; offering_color: "white" | "black" }
	| { type: "draw_resolved"; offer_id: string; status: "accepted" | "declined"; version: number }
	| { type: "opponent_connection"; color: "white" | "black"; connected: boolean };

export function matchReducer(state: MatchState, event: RealtimeEvent): MatchState {
	if (event.type === "action_rejected") {
		return { ...state, error: event.message };
	}
	if (event.type === "draw_offer") {
		return { ...state, drawOffer: { offer_id: event.offer_id, offering_color: event.offering_color }, notice: "" };
	}
	if (event.type === "draw_resolved") {
		return {
			...state,
			drawOffer: null,
			notice: event.status === "declined" ? "Draw declined" : "Draw accepted",
		};
	}
	if (event.type === "opponent_connection") {
		if (event.color === state.color) return state;
		return { ...state, opponentConnected: event.connected };
	}
	if (event.type === "match_snapshot") {
		if (event.version < state.version) return state;
		return {
			matchId: event.match_id,
			status: event.status,
			color: event.color,
			position: event.position,
			currentTurn: event.next_turn,
			version: event.version,
			timeControl: event.time_control,
			inCheck: event.in_check,
			eligibleDraws: event.eligible_draws,
			moves: event.moves ?? [],
			clock: event.clock,
			result: event.result ?? "",
			drawOffer: null,
			notice: "",
			opponentConnected: false,
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
		moves: [...state.moves, event.move],
		inCheck: event.in_check,
		eligibleDraws: event.eligible_draws,
		currentTurn: event.next_turn,
		version: event.version,
		clock: event.clock,
		result: event.result,
		status: event.result ? "completed" : "active",
		error: "",
		syncRequired: false,
	};
}
