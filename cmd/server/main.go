package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"shortlink/internal/db"
	"shortlink/internal/httplog"
	link "shortlink/internal/links"
	"shortlink/internal/users"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://shortlink:dev_password@localhost:5432/shortlink"
	}
	if err := db.Init(ctx, dsn); err != nil {
		log.Fatal("db init:", err)

	}
	if err := db.RunMigrations(ctx, "migrations"); err != nil {
		log.Fatal("migrations:", err)

	}
	linkStore := link.NewStore(db.Pool)
	linkHandlers := link.NewHandlers(linkStore)
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(httplog.Middleware(logger))
	r.Route("/api", func(r chi.Router) {
		r.Post("/signup", users.SignupHandler)
		r.Post("/login", users.LoginHandler)

		r.Group(func(r chi.Router) {
			r.Use(users.RequireAuth)
			r.Post("/logout", users.LogoutHandler)
			r.Get("/me", users.MeHandler)
			r.Post("/shorten", linkHandlers.Shorten)
			r.Get("/links", linkHandlers.ListMine)
		})
	})
	r.Get("/{code}", linkHandlers.Redirect)
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	logger.Info("server starting", "addr", ":8080")
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
