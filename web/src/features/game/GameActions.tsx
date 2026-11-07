import { useState } from "react";
import { claimDraw, offerDraw, resolveDraw, resignMatch } from "../match/matchClient";

export type DrawOfferView = {
  offer_id: string;
  offering_color: "white" | "black";
};

export function GameActions({
  matchID,
  color,
  recoveryCredential,
  drawOffer,
  eligibleDraws,
}: {
  matchID: string;
  color: "white" | "black";
  recoveryCredential: string;
  drawOffer: DrawOfferView | null;
  eligibleDraws: string[];
}) {
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function run(action: () => Promise<unknown>) {
    setError("");
    setPending(true);
    try {
      await action();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Match action was not accepted.");
    } finally {
      setPending(false);
    }
  }

  return (
    <section className="game-actions" aria-label="Game actions">
      <button type="button" disabled={pending || Boolean(drawOffer)} onClick={() => void run(() => offerDraw(matchID, recoveryCredential))}>
        Offer draw
      </button>
      <button type="button" disabled={pending} onClick={() => void run(() => resignMatch(matchID, recoveryCredential))}>
        Resign
      </button>
      {eligibleDraws.includes("ThreefoldRepetition") && (
        <button type="button" disabled={pending} onClick={() => void run(() => claimDraw(matchID, recoveryCredential, "ThreefoldRepetition"))}>
          Claim threefold repetition draw
        </button>
      )}
      {eligibleDraws.includes("FiftyMoveRule") && (
        <button type="button" disabled={pending} onClick={() => void run(() => claimDraw(matchID, recoveryCredential, "FiftyMoveRule"))}>
          Claim fifty-move draw
        </button>
      )}
      {drawOffer && drawOffer.offering_color !== color && (
        <div className="draw-response" role="group" aria-label="Draw offer response">
          <p>Draw offer from {drawOffer.offering_color === "white" ? "White" : "Black"}</p>
          <button type="button" disabled={pending} onClick={() => void run(() => resolveDraw(matchID, drawOffer.offer_id, recoveryCredential, "accept"))}>
            Accept draw
          </button>
          <button type="button" disabled={pending} onClick={() => void run(() => resolveDraw(matchID, drawOffer.offer_id, recoveryCredential, "decline"))}>
            Decline draw
          </button>
        </div>
      )}
      {drawOffer && drawOffer.offering_color === color && <p role="status">Draw offer sent</p>}
      {error && <p role="alert">{error}</p>}
    </section>
  );
}
