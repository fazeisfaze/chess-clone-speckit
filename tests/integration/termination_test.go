package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"chess_clone/internal/config"
	"chess_clone/internal/store/postgres"
	httpapi "chess_clone/internal/transport/http"
)

func TestDrawOfferDeclineAcceptanceAndResignation(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for termination integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	_, currentFile, _, _ := runtime.Caller(0)
	if err := postgres.ApplyMigrations(ctx, pool, filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpapi.NewApplicationHandler(config.Config{AllowedOrigin: "http://localhost:5173"}, pool))
	defer server.Close()

	createMatch := func() (string, string, string) {
		createBody, _ := json.Marshal(map[string]string{"time_control": "5+0"})
		createdResponse, err := http.Post(server.URL+"/matches", "application/json", bytes.NewReader(createBody))
		if err != nil {
			t.Fatal(err)
		}
		var created struct {
			ID         string `json:"match_id"`
			InviteURL  string `json:"invite_url"`
			Credential string `json:"recovery_credential"`
		}
		if err := json.NewDecoder(createdResponse.Body).Decode(&created); err != nil {
			createdResponse.Body.Close()
			t.Fatal(err)
		}
		createdResponse.Body.Close()
		if createdResponse.StatusCode != http.StatusCreated {
			t.Fatalf("create status = %d", createdResponse.StatusCode)
		}
		token := strings.TrimPrefix(strings.SplitN(created.InviteURL, "#token=", 2)[1], "")
		joinBody, _ := json.Marshal(map[string]string{"invite_token": token})
		joinedResponse, err := http.Post(server.URL+"/matches/join", "application/json", bytes.NewReader(joinBody))
		if err != nil {
			t.Fatal(err)
		}
		var joined struct {
			Credential string `json:"recovery_credential"`
		}
		if err := json.NewDecoder(joinedResponse.Body).Decode(&joined); err != nil {
			joinedResponse.Body.Close()
			t.Fatal(err)
		}
		joinedResponse.Body.Close()
		if joinedResponse.StatusCode != http.StatusOK {
			t.Fatalf("join status = %d", joinedResponse.StatusCode)
		}
		return created.ID, created.Credential, joined.Credential
	}

	post := func(path, credential string, body any) (*http.Response, error) {
		data, _ := json.Marshal(body)
		request, _ := http.NewRequest(http.MethodPost, server.URL+path, bytes.NewReader(data))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+credential)
		return http.DefaultClient.Do(request)
	}

	matchID, whiteCredential, blackCredential := createMatch()
	offerResponse, err := post("/matches/"+matchID+"/draw-offers", whiteCredential, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	var offer struct {
		ID     string `json:"offer_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(offerResponse.Body).Decode(&offer); err != nil {
		offerResponse.Body.Close()
		t.Fatal(err)
	}
	offerResponse.Body.Close()
	if offerResponse.StatusCode != http.StatusCreated || offer.Status != "pending" || offer.ID == "" {
		t.Fatalf("draw offer status=%d response=%+v", offerResponse.StatusCode, offer)
	}
	decline, err := post(fmt.Sprintf("/matches/%s/draw-offers/%s/resolve", matchID, offer.ID), blackCredential, map[string]string{"decision": "decline"})
	if err != nil {
		t.Fatal(err)
	}
	decline.Body.Close()
	if decline.StatusCode != http.StatusOK {
		t.Fatalf("decline status = %d", decline.StatusCode)
	}
	offerResponse, err = post("/matches/"+matchID+"/draw-offers", whiteCredential, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewDecoder(offerResponse.Body).Decode(&offer); err != nil {
		offerResponse.Body.Close()
		t.Fatal(err)
	}
	offerResponse.Body.Close()
	accept, err := post(fmt.Sprintf("/matches/%s/draw-offers/%s/resolve", matchID, offer.ID), blackCredential, map[string]string{"decision": "accept"})
	if err != nil {
		t.Fatal(err)
	}
	var drawn struct {
		Status      string `json:"status"`
		MatchStatus string `json:"match_status"`
		Result      string `json:"result"`
	}
	if err := json.NewDecoder(accept.Body).Decode(&drawn); err != nil {
		accept.Body.Close()
		t.Fatal(err)
	}
	accept.Body.Close()
	if accept.StatusCode != http.StatusOK || drawn.Status != "accepted" || drawn.MatchStatus != "completed" || drawn.Result != "1/2-1/2" {
		t.Fatalf("accepted draw status=%d response=%+v", accept.StatusCode, drawn)
	}

	secondMatchID, secondWhite, _ := createMatch()
	resign, err := post("/matches/"+secondMatchID+"/resign", secondWhite, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	var resigned struct {
		Status string `json:"status"`
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resign.Body).Decode(&resigned); err != nil {
		resign.Body.Close()
		t.Fatal(err)
	}
	resign.Body.Close()
	if resign.StatusCode != http.StatusOK || resigned.Status != "completed" || resigned.Result != "0-1" {
		t.Fatalf("resign status=%d response=%+v", resign.StatusCode, resigned)
	}

	drawMatchID, drawWhite, drawBlack := createMatch()
	playMove := func(credential string, version int, from, to string) {
		t.Helper()
		response, err := post(fmt.Sprintf("/matches/%s/moves", drawMatchID), credential, map[string]any{
			"expected_version": version, "from": from, "to": to, "promotion": nil,
		})
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("move %s%s status=%d", from, to, response.StatusCode)
		}
	}
	moves := []struct {
		credential string
		from, to   string
	}{
		{drawWhite, "g1", "f3"}, {drawBlack, "g8", "f6"},
		{drawWhite, "f3", "g1"}, {drawBlack, "f6", "g8"},
		{drawWhite, "g1", "f3"}, {drawBlack, "g8", "f6"},
		{drawWhite, "f3", "g1"}, {drawBlack, "f6", "g8"},
	}
	for index, move := range moves {
		playMove(move.credential, index+1, move.from, move.to)
	}
	stateRequest, _ := http.NewRequest(http.MethodGet, server.URL+"/matches/"+drawMatchID, nil)
	stateRequest.Header.Set("Authorization", "Bearer "+drawWhite)
	stateResponse, err := http.DefaultClient.Do(stateRequest)
	if err != nil {
		t.Fatal(err)
	}
	var drawState struct {
		EligibleDraws []string `json:"eligible_draws"`
	}
	if err := json.NewDecoder(stateResponse.Body).Decode(&drawState); err != nil {
		stateResponse.Body.Close()
		t.Fatal(err)
	}
	stateResponse.Body.Close()
	foundThreefold := false
	for _, method := range drawState.EligibleDraws {
		if method == "ThreefoldRepetition" {
			foundThreefold = true
		}
	}
	if !foundThreefold {
		t.Fatalf("claimable draw methods = %v", drawState.EligibleDraws)
	}
	claim, err := post("/matches/"+drawMatchID+"/draw-claims", drawWhite, map[string]string{"method": "ThreefoldRepetition"})
	if err != nil {
		t.Fatal(err)
	}
	var claimed struct {
		Status string `json:"status"`
		Result string `json:"result"`
	}
	if err := json.NewDecoder(claim.Body).Decode(&claimed); err != nil {
		claim.Body.Close()
		t.Fatal(err)
	}
	claim.Body.Close()
	if claim.StatusCode != http.StatusOK || claimed.Status != "completed" || claimed.Result != "1/2-1/2" {
		t.Fatalf("draw claim status=%d response=%+v", claim.StatusCode, claimed)
	}
}
