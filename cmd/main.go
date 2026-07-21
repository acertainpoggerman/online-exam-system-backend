package main

import (
	"context"
	"log"
	"net"
	"net/http"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/core/scripts"
	"github.com/acertainpoggerman/online-exam-system/internal/core/sessions"
	"github.com/acertainpoggerman/online-exam-system/internal/core/users"
	"github.com/acertainpoggerman/online-exam-system/internal/middleware"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {

	ctx := context.Background()

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.DbConnectionString == "" {
		log.Fatal("DB_CONNECTION_STRING is required (set it in .env.local)")
	}

	jwtSecretKey := []byte(cfg.JwtSecretKey)
	jwtExpiryTime := cfg.JwtExpiryTime

	// --------------------------------------------------------------------------
	// --- Instantiating DB & Store ---------------------------------------------
	// --------------------------------------------------------------------------

	pool, err := pgxpool.New(ctx, cfg.DbConnectionString)
	if err != nil {
		log.Fatal(err)
	}
	q := store.New(pool)

	// rdb := redis.NewClient(&redis.Options{
	// 	Addr:     "localhost:6379",
	// 	Password: "",
	// 	DB:       0,
	// })

	// --------------------------------------------------------------------------
	// --- Instatiating Routers -------------------------------------------------
	// --------------------------------------------------------------------------

	router := http.NewServeMux()

	unauthedRouter := http.NewServeMux()
	router.Handle("/auth/", http.StripPrefix("/auth", unauthedRouter))

	wsRouter := http.NewServeMux()
	router.Handle("/ws/", http.StripPrefix("/ws", wsRouter))

	authedRouter := http.NewServeMux()
	router.Handle("/", middleware.JWTAuth(jwtSecretKey)(authedRouter))

	// --------------------------------------------------------------------------
	// --- Instantiating Services -----------------------------------------------
	// --------------------------------------------------------------------------

	authService := users.NewAuthService(q, pool, jwtSecretKey, jwtExpiryTime)
	authHandler := users.NewAuthHandler(authService, int(jwtExpiryTime.Seconds()))
	authHandler.RegisterRoutes(unauthedRouter)

	userService := users.NewUserService(q, pool)
	userHandler := users.NewUserHandler(userService)
	userHandler.RegisterRoutes(authedRouter)

	scriptService := scripts.NewScriptService(q, pool)
	scriptHandler := scripts.NewScriptHandler(scriptService)
	scriptHandler.RegisterRoutes(authedRouter)

	hub := sessions.NewHub()

	sessionService := sessions.NewSessionService(q, pool, hub, scriptService)
	sessionHandler := sessions.NewSessionHandler(sessionService)
	sessionHandler.RegisterRoutes(authedRouter)

	wsHandler := sessions.NewWebsocketHandler(sessionService, hub, jwtSecretKey)
	wsHandler.RegisterRoutes(wsRouter)

	// --------------------------------------------------------------------------
	// --- Defining and Running Server ------------------------------------------
	// --------------------------------------------------------------------------

	addr := cfg.ServerAddr
	log.Printf("Starting server at %s", addr)

	server := http.Server{
		Addr:        addr,
		Handler:     middleware.CORS(chimiddleware.Logger(router)),
		BaseContext: func(l net.Listener) context.Context { return ctx },
	}
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Server error occured: %v\n", err)
	}
}
