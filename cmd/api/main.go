package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	ingestSvc "crisisecho/internal/apps/ingest/service"
	postRepo  "crisisecho/internal/apps/post/repository"
	postSvc   "crisisecho/internal/apps/post/service"
	"crisisecho/internal/database"
	"crisisecho/internal/server"
)

func main() {
	// Panic recovery — catches any unhandled panic in the main goroutine and
	// logs it before the process exits.
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("crisisecho: panic: %v", r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// ── Connect all three MongoDB clients ────────────────────────────────────
	mainClient, err := database.ConnectMain(ctx)
	if err != nil {
		log.Fatalf("crisisecho: failed to connect main DB: %v", err)
	}

	locationClient, err := database.ConnectLocation(ctx)
	if err != nil {
		log.Fatalf("crisisecho: failed to connect location DB: %v", err)
	}

	vectorClient, err := database.ConnectVector(ctx)
	if err != nil {
		log.Fatalf("crisisecho: failed to connect vector DB: %v", err)
	}

	// ── Connect Redis ─────────────────────────────────────────────────────────
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("crisisecho: REDIS_URL is not set")
	}
	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("crisisecho: invalid REDIS_URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("crisisecho: failed to connect Redis: %v", err)
	}

	// ── Build server + register routes ───────────────────────────────────────
	srv := server.New(mainClient, locationClient, vectorClient, redisClient)
	server.RegisterRoutes(srv)

	// ── Application context (cancelled on shutdown signal) ───────────────────
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	// ── Kafka ingest goroutine (guarded by KAFKA_BROKERS) ────────────────────
	if brokersEnv := os.Getenv("KAFKA_BROKERS"); brokersEnv != "" {
		brokers := strings.Split(brokersEnv, ",")
		groupID := os.Getenv("KAFKA_GROUP_ID")
		if groupID == "" {
			groupID = "crisisecho-ingest"
		}

		mainDB     := mainClient.Database(os.Getenv("MONGO_DB_DATABASE"))
		unifiedRepo := postRepo.NewUnifiedPostRepository(mainDB)
		postService  := postSvc.NewPostService(mainDB, unifiedRepo)
		ingestService := ingestSvc.NewIngestService(brokers, groupID, postService)

		go func() {
			if err := ingestService.ConsumeAndRoute(appCtx); err != nil {
				log.Printf("crisisecho: ingest consumer stopped: %v", err)
			}
		}()
		fmt.Printf("crisisecho: Kafka consumer started (brokers=%s group=%s)\n", brokersEnv, groupID)
	}

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		fmt.Printf("crisisecho: listening on :%s\n", port)
		if err := srv.App.Listen(":" + port); err != nil {
			log.Printf("crisisecho: server error: %v", err)
		}
	}()

	<-quit
	fmt.Println("crisisecho: shutting down…")

	// Cancel application context to stop the Kafka goroutine.
	appCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown Fiber first so no new requests arrive.
	if err := srv.App.Shutdown(); err != nil {
		log.Printf("crisisecho: fiber shutdown error: %v", err)
	}

	// Disconnect all three MongoDB clients.
	if err := mainClient.Disconnect(shutdownCtx); err != nil {
		log.Printf("crisisecho: main DB disconnect error: %v", err)
	}
	if err := locationClient.Disconnect(shutdownCtx); err != nil {
		log.Printf("crisisecho: location DB disconnect error: %v", err)
	}
	if err := vectorClient.Disconnect(shutdownCtx); err != nil {
		log.Printf("crisisecho: vector DB disconnect error: %v", err)
	}

	// Close Redis.
	if err := redisClient.Close(); err != nil {
		log.Printf("crisisecho: redis close error: %v", err)
	}

	fmt.Println("crisisecho: shutdown complete")
}
