package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"shmanki/internal/card"
	"shmanki/internal/deck"
	"shmanki/internal/fsrs"
	"shmanki/internal/generate"
	"shmanki/internal/platform/config"
	platformdb "shmanki/internal/platform/db"
	platformmiddleware "shmanki/internal/platform/middleware"
	"shmanki/internal/platform/response"
	"shmanki/internal/platform/token"
	"shmanki/internal/review"
	"shmanki/internal/user"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	log.Printf("loaded config from backend/.env")

	log.Printf("connecting to database")
	dbPool, err := platformdb.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer dbPool.Close()

	tokenManager := token.NewManager(cfg.JWTSecret, cfg.JWTTTLHours)
	scheduler := fsrs.NewScheduler(fsrs.DefaultWeights, fsrs.DefaultConfig)

	userRepo := user.NewRepository(dbPool)
	userService := user.NewService(userRepo, tokenManager, cfg.DefaultLanguage)
	userHandler := user.NewHandler(userService)

	deckRepo := deck.NewRepository(dbPool)
	deckService := deck.NewService(deckRepo, userRepo, cfg.DefaultLanguage)
	deckHandler := deck.NewHandler(deckService)

	cardRepo := card.NewRepository(dbPool)
	cardService := card.NewService(cardRepo)
	cardHandler := card.NewHandler(cardService)

	reviewRepo := review.NewRepository(dbPool)
	reviewService := review.NewService(reviewRepo, scheduler)
	reviewHandler := review.NewHandler(reviewService)

	generateRepo := generate.NewRepository(dbPool)
	generateClient := generate.NewClient(cfg.LLMAPIURL, cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMProvider, cfg.LLMTimeoutSeconds)
	generateService := generate.NewService(generateRepo, generateClient, cfg.DefaultLanguage)
	generateHandler := generate.NewHandler(generateService)

	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	router.Use(platformmiddleware.Recovery)
	router.Use(platformmiddleware.Logger)

	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	router.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", userHandler.Register)
			r.Post("/login", userHandler.Login)
		})

		r.Group(func(r chi.Router) {
			r.Use(platformmiddleware.Auth(tokenManager))
			r.Route("/decks", func(r chi.Router) {
				r.Get("/", deckHandler.List)
				r.Post("/", deckHandler.Create)
				r.Get("/{id}", deckHandler.Get)
				r.Put("/{id}", deckHandler.Update)
				r.Delete("/{id}", deckHandler.Delete)
				r.Get("/{deckID}/objects", cardHandler.ListInfoObjects)
				r.Post("/{deckID}/objects", cardHandler.CreateInfoObject)
			})

			r.Route("/objects", func(r chi.Router) {
				r.Get("/{id}", cardHandler.GetInfoObject)
				r.Put("/{id}", cardHandler.UpdateInfoObject)
				r.Delete("/{id}", cardHandler.DeleteInfoObject)
				r.Post("/{objectID}/cards", cardHandler.CreateCard)
			})

			r.Route("/cards", func(r chi.Router) {
				r.Put("/{id}", cardHandler.UpdateCard)
				r.Delete("/{id}", cardHandler.DeleteCard)
			})

			r.Route("/review", func(r chi.Router) {
				r.Get("/session", reviewHandler.GetSession)
				r.Post("/submit", reviewHandler.SubmitReview)
			})

			r.Route("/stats", func(r chi.Router) {
				r.Get("/deck/{id}", reviewHandler.GetDeckStats)
				r.Get("/card/{id}", reviewHandler.GetCardStats)
			})

			r.Route("/generate", func(r chi.Router) {
				r.Post("/suggest", generateHandler.Suggest)
				r.Post("/edit", generateHandler.Edit)
				r.Post("/save", generateHandler.Save)
			})
		})
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown server: %v", err)
		}
	}()

	log.Printf("api listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen and serve: %v", err)
	}
}
