import { useEffect, useState } from "react";

export type ClockView = {
  white_remaining_ms: number;
  black_remaining_ms: number;
  active_color: "white" | "black" | null;
  as_of?: string;
  client_received_at?: number;
};

function format(milliseconds: number) {
  const seconds = Math.max(0, Math.floor(milliseconds / 1000));
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;
}

export function ClockDisplay({ clock }: { clock: ClockView }) {
  const [now, setNow] = useState(() => Date.now());
  const [receivedAt, setReceivedAt] = useState(() => clock.client_received_at ?? Date.now());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 200);
    return () => window.clearInterval(timer);
  }, []);
  useEffect(() => {
    setReceivedAt(clock.client_received_at ?? Date.now());
  }, [clock.as_of, clock.client_received_at]);

  const serverAnchor = clock.as_of ? Date.parse(clock.as_of) : Number.NaN;
  const offset = Number.isFinite(serverAnchor) ? serverAnchor - receivedAt : 0;
  const estimatedServerNow = now + offset;
  const elapsed = clock.active_color && Number.isFinite(serverAnchor)
    ? Math.max(0, estimatedServerNow - serverAnchor)
    : 0;
  const white = clock.white_remaining_ms - (clock.active_color === "white" ? elapsed : 0);
  const black = clock.black_remaining_ms - (clock.active_color === "black" ? elapsed : 0);

  return (
    <div className="clocks" aria-label="Match clocks">
      <output className={`clock-row ${clock.active_color === "white" ? "active" : ""}`} aria-label="White clock">
        <span>White</span><strong>{" "}{format(white)}</strong>
      </output>
      <output className={`clock-row ${clock.active_color === "black" ? "active" : ""}`} aria-label="Black clock">
        <span>Black</span><strong>{" "}{format(black)}</strong>
      </output>
    </div>
  );
}
