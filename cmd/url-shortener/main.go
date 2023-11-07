package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/exp/slog"
	"net/http"
	"os"
	"url-shortener/internal/config"
	"url-shortener/internal/http-server/handlers/redirect"
	"url-shortener/internal/http-server/handlers/url/delete"
	"url-shortener/internal/http-server/handlers/url/save"
	mwLogger "url-shortener/internal/http-server/middleware/logger"
	"url-shortener/internal/lib/logger/handlers/slogpretty"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage/sqlite"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	//configs
	cfg := config.MustLoad()

	//loggers
	logger := setupLogger(cfg.Env)

	logger.Info("starting app", slog.String("env", cfg.Env))
	logger.Debug("debug messages are enabled")

	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		logger.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	//id, err := storage.SaveUrl("https://google.com", "google")
	//if err != nil {
	//	logger.Error("failed to same url", sl.Err(err))
	//	os.Exit(1)
	//}
	//
	//logger.Info("saved url", slog.Int64("id", id))
	//
	//err = storage.DeleteUrl("google1")
	//if err != nil {
	//	logger.Error("failed to remove url", sl.Err(err))
	//	os.Exit(1)
	//}

	//router
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(mwLogger.New(logger))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/url", func(r chi.Router) {
		r.Use(middleware.BasicAuth("url-shortener", map[string]string{
			cfg.HTTPServer.User: cfg.HTTPServer.Password,
		}))

		r.Post("/", save.New(logger, storage))
		r.Delete("/{alias}", delete.New(logger, storage))
	})

	router.Get("/{alias}", redirect.New(logger, storage))
	logger.Info("starting server", slog.String("address", cfg.Address))

	port := os.Getenv("PORT")
	if port == "" {
		logger.Error("$PORT must be set")
	}

	srv := &http.Server{
		Addr:         cfg.Address + ":" + port,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		logger.Error("failed to start server")
	}

	logger.Info("server stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}
	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
