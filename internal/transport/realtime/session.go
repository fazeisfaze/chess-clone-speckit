package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

type SessionOptions struct {
	AllowedOrigin         string
	AuthenticationTimeout time.Duration
	MaxMessageBytes       int64
	Authorize             func(context.Context, string, string) bool
	Hub                   *Hub
	Snapshot              func(context.Context, string, string) ([]byte, error)
	OnMessage             func(context.Context, string, string, []byte) error
}

func NewSessionHandler(options SessionOptions) http.Handler {
	if options.AuthenticationTimeout <= 0 {
		options.AuthenticationTimeout = 5 * time.Second
	}
	if options.MaxMessageBytes <= 0 {
		options.MaxMessageBytes = 4096
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" || origin != options.AllowedOrigin {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{options.AllowedOrigin},
		})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "session ended")
		conn.SetReadLimit(options.MaxMessageBytes)

		authCtx, cancel := context.WithTimeout(r.Context(), options.AuthenticationTimeout)
		defer cancel()
		messageType, data, err := conn.Read(authCtx)
		if err != nil {
			status := websocket.StatusPolicyViolation
			if authCtx.Err() != nil && r.Context().Err() != nil {
				status = websocket.StatusGoingAway
			}
			_ = conn.Close(status, "authentication required")
			return
		}
		var message struct {
			Type               string `json:"type"`
			RecoveryCredential string `json:"recovery_credential"`
		}
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.DisallowUnknownFields()
		if messageType != websocket.MessageText || decoder.Decode(&message) != nil || message.Type != "authenticate" || message.RecoveryCredential == "" || options.Authorize == nil {
			_ = conn.Close(websocket.StatusPolicyViolation, "invalid authentication")
			return
		}
		matchID := r.PathValue("match_id")
		if !options.Authorize(r.Context(), matchID, message.RecoveryCredential) {
			_ = conn.Close(websocket.StatusPolicyViolation, "authentication failed")
			return
		}
		if options.Hub == nil {
			_ = conn.Write(r.Context(), websocket.MessageText, []byte("{\"type\":\"authenticated\"}"))
			return
		}
		events, unsubscribe := options.Hub.Subscribe(matchID)
		defer unsubscribe()
		var snapshot []byte
		if options.Snapshot != nil {
			snapshot, err = options.Snapshot(r.Context(), matchID, message.RecoveryCredential)
			if err != nil {
				_ = conn.Close(websocket.StatusPolicyViolation, "match unavailable")
				return
			}
		} else {
			snapshot = []byte("{\"type\":\"authenticated\"}")
		}
		if err := conn.Write(r.Context(), websocket.MessageText, snapshot); err != nil {
			return
		}
		type incomingMessage struct {
			data []byte
			err  error
		}
		incoming := make(chan incomingMessage, 1)
		go func() {
			for {
				messageType, data, err := conn.Read(r.Context())
				if err != nil {
					incoming <- incomingMessage{err: err}
					return
				}
				if messageType != websocket.MessageText {
					incoming <- incomingMessage{err: errors.New("only text messages are supported")}
					return
				}
				select {
				case incoming <- incomingMessage{data: data}:
				case <-r.Context().Done():
					return
				}
			}
		}()
		for {
			select {
			case event, ok := <-events:
				if !ok {
					_ = conn.Close(websocket.StatusTryAgainLater, "reconnect to resynchronize")
					return
				}
				if err := conn.Write(r.Context(), websocket.MessageText, event); err != nil {
					return
				}
			case received := <-incoming:
				if received.err != nil {
					return
				}
				if options.OnMessage == nil || options.OnMessage(r.Context(), matchID, message.RecoveryCredential, received.data) != nil {
					_ = conn.Write(r.Context(), websocket.MessageText, []byte("{\"type\":\"action_rejected\",\"error\":\"action not accepted\"}"))
				}
			case <-r.Context().Done():
				return
			}
		}
	})
}
