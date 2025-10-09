package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/acertainpoggerman/online-exam-system/internal/handlers"
	"github.com/acertainpoggerman/online-exam-system/internal/middleware"
	"github.com/acertainpoggerman/online-exam-system/internal/repository"
	"github.com/acertainpoggerman/online-exam-system/internal/services"
	"github.com/acertainpoggerman/online-exam-system/internal/tools"
	"github.com/acertainpoggerman/online-exam-system/serverconfig"
)

func main() {

	ctx := context.Background()

	// [Config, Logging]

	cfg, err := serverconfig.LoadConfig(".env.local")
	if err != nil {
		slog.Error("Error loading configuration", "error", err.Error())
	}

	slog.SetLogLoggerLevel(cfg.LogLevel)

	// [Mongo Client]

	client, err := tools.ConnectDB(cfg.DatabaseURI)
	if err != nil {
		slog.Error("Error connecting to MongoDB Client", "error", err.Error())
	}

	repo := repository.NewMongoRepository(client, cfg.DatabaseName)
	svc := services.NewService(repo, cfg.JwtSecretKey, cfg.JwtExpiryTime)
	httpHandler := handlers.NewHandler(svc)
	// taskHandler := tasks.NewHandler(svc)

	// [Routing]

	router := http.NewServeMux()

	// --- Non-Authenticated Routes
	// [Auth]
	router.HandleFunc("POST /register", httpHandler.HandleRegister)
	router.HandleFunc("POST /login", httpHandler.HandleLogin)
	// ---

	authRouter := http.NewServeMux()
	router.Handle("/", middleware.IsAuthenticated(cfg.JwtSecretKey)(authRouter))

	// --- Authenticated Routes
	// [Me/Users]
	authRouter.HandleFunc("GET /me", httpHandler.HandleGetMe)
	authRouter.HandleFunc("DELETE /me", httpHandler.HandleDeleteMe)
	// [Scripts]
	authRouter.HandleFunc("GET /scripts/{id}", httpHandler.HandleGetScriptByID)
	authRouter.HandleFunc("POST /scripts", httpHandler.HandleCreateScript)
	authRouter.HandleFunc("DELETE /scripts/{id}", httpHandler.HandleDeleteScriptByID)
	authRouter.HandleFunc("PUT /scripts/{id}", httpHandler.HandleUpdateScriptByID)
	// [Sessions]
	authRouter.HandleFunc("GET /sessions/{id}", httpHandler.HandleGetSessionByID)
	authRouter.HandleFunc("POST /sessions", httpHandler.HandleCreateSession)
	authRouter.HandleFunc("DELETE /sessions/{id}", httpHandler.HandleDeleteSessionByID)
	authRouter.HandleFunc("PUT /sessions/{id}", httpHandler.HandleUpdateSessionByID)
	// ---

	// [Server]

	server := http.Server{
		Addr:        cfg.ServerAddr,
		Handler:     middleware.Logging(router),
		BaseContext: func(l net.Listener) context.Context { return ctx },
	}

	slog.Info("Starting server at " + cfg.ServerAddr)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Server error occured", "error", err.Error())
	}
}
