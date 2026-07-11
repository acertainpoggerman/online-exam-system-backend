package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/core/scripts"
	"github.com/acertainpoggerman/online-exam-system/internal/core/sessions"
	"github.com/acertainpoggerman/online-exam-system/internal/core/submissions"
	"github.com/acertainpoggerman/online-exam-system/internal/core/users"
	"github.com/acertainpoggerman/online-exam-system/internal/middleware"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {

	ctx := context.Background()

	jwtSecretKey := []byte("thisissecretkey")
	jwtExpiryTime := 30 * time.Minute

	// --------------------------------------------------------------------------
	// --- Instantiating DB & Store ---------------------------------------------
	// --------------------------------------------------------------------------

	pool, err := pgxpool.New(ctx, "user=postgres password=mysecretpassword dbname=examdb")
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

	submissionService := submissions.NewSubmissionService(q, pool, scriptService)
	submissionHandler := submissions.NewSubmissionHandler(submissionService)
	submissionHandler.RegisterRoutes(authedRouter)

	hub := sessions.NewHub()

	sessionService := sessions.NewSessionService(q, pool, hub, submissionService)
	sessionHandler := sessions.NewSessionHandler(sessionService)
	sessionHandler.RegisterRoutes(authedRouter)

	wsHandler := sessions.NewWebsocketHandler(sessionService, hub, jwtSecretKey)
	wsHandler.RegisterRoutes(wsRouter)

	// --------------------------------------------------------------------------
	// --- Defining and Running Server ------------------------------------------
	// --------------------------------------------------------------------------

	addr := ":3000"
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
