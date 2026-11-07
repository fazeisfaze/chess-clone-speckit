package httpapi

import (
	"context"
	"net/http"

	"chess_clone/internal/config"
	"chess_clone/internal/match"
	"chess_clone/internal/store/postgres"
	"chess_clone/internal/transport/realtime"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Application struct {
	Handler    http.Handler
	Hub        *realtime.Hub
	MatchStore *postgres.MatchStore
}

func NewApplication(cfg config.Config, pool *pgxpool.Pool) *Application {
	mux := http.NewServeMux()
	store := postgres.NewMatchStore(pool)
	service := match.NewService(store)
	hub := realtime.NewHub()
	endpoints := matchEndpoints{service: service, origin: cfg.AllowedOrigin, hub: hub}
	mux.HandleFunc("POST /matches", endpoints.create)
	mux.HandleFunc("POST /matches/join", endpoints.join)
	mux.HandleFunc("GET /matches/{match_id}", endpoints.get)
	mux.HandleFunc("POST /matches/{match_id}/recover", endpoints.recover)
	mux.HandleFunc("POST /matches/{match_id}/moves", endpoints.move)
	mux.HandleFunc("POST /matches/{match_id}/resign", endpoints.resign)
	mux.HandleFunc("POST /matches/{match_id}/draw-offers", endpoints.offerDraw)
	mux.HandleFunc("POST /matches/{match_id}/draw-offers/{offer_id}/resolve", endpoints.resolveDraw)
	mux.HandleFunc("POST /matches/{match_id}/draw-claims", endpoints.claimDraw)
	mux.Handle("GET /matches/{match_id}/events", realtime.NewSessionHandler(realtime.SessionOptions{
		AllowedOrigin: cfg.AllowedOrigin,
		Authorize: func(ctx context.Context, matchID, credential string) bool {
			_, err := service.Read(ctx, matchID, credential)
			return err == nil
		},
		Hub:       hub,
		Snapshot:  endpoints.snapshot,
		OnMessage: endpoints.socketMessage,
	}))
	mux.Handle("/", NewHandler(cfg))
	return &Application{Handler: mux, Hub: hub, MatchStore: store}
}

func NewApplicationHandler(cfg config.Config, pool *pgxpool.Pool) http.Handler {
	return NewApplication(cfg, pool).Handler
}
