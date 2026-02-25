package server

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	alertCtrl "crisisecho/internal/apps/alert/controller"
	alertRepo "crisisecho/internal/apps/alert/repository"
	alertSvc "crisisecho/internal/apps/alert/service"
	clusterCtrl "crisisecho/internal/apps/cluster/controller"
	clusterRepo "crisisecho/internal/apps/cluster/repository"
	clusterSvc "crisisecho/internal/apps/cluster/service"
	crisisCtrl "crisisecho/internal/apps/crisis/controller"
	crisisRepo "crisisecho/internal/apps/crisis/repository"
	crisisSvc "crisisecho/internal/apps/crisis/service"
	notifyCtrl "crisisecho/internal/apps/notify/controller"
	notifyRepo "crisisecho/internal/apps/notify/repository"
	notifySvc "crisisecho/internal/apps/notify/service"
	postCtrl "crisisecho/internal/apps/post/controller"
	postRepo "crisisecho/internal/apps/post/repository"
	postSvc "crisisecho/internal/apps/post/service"
	queryCtrl "crisisecho/internal/apps/query/controller"
	querySvc "crisisecho/internal/apps/query/service"
	ragSvc     "crisisecho/internal/apps/rag/service"
	uploadCtrl "crisisecho/internal/apps/upload/controller"
	uploadRepo "crisisecho/internal/apps/upload/repository"
	uploadSvc  "crisisecho/internal/apps/upload/service"
	unifiedPostCtrl "crisisecho/internal/apps/unifiedpost/controller"
	unifiedPostRepo "crisisecho/internal/apps/unifiedpost/repository"
	unifiedPostSvc "crisisecho/internal/apps/unifiedpost/service"
	"crisisecho/internal/database"
	locationRepo "crisisecho/internal/database/abstractrepository/location"
	vectorRepo "crisisecho/internal/database/abstractrepository/vectordb"
)

// RegisterRoutes wires all route groups and their dependencies into the Fiber app.
// This is the single point of manual dependency injection for the entire application.
func RegisterRoutes(srv *FiberServer) {
	// ── Global CORS ───────────────────────────────────────────────────────────
	srv.App.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// ── Health endpoints ──────────────────────────────────────────────────────
	srv.App.Get("/", healthHandler(srv))
	srv.App.Get("/health", healthHandler(srv))

	// ── Database handles ──────────────────────────────────────────────────────
	mainDB := srv.MainClient.Database(os.Getenv("MONGO_DB_DATABASE"))

	api := srv.App.Group("/api")

	// === Post ===
	sourcePostRepo := postRepo.NewSourcePostRepository(mainDB)
	postService := postSvc.NewPostService(mainDB, sourcePostRepo)
	postController := postCtrl.NewPostController(postService)
	postController.RegisterRoutes(api.Group("/source-posts"))

	// === Unified Post ===
	unifiedPostRepository := unifiedPostRepo.NewUnifiedPostRepository(mainDB)
	unifiedPostService := unifiedPostSvc.NewUnifiedPostService(unifiedPostRepository)
	unifiedPostController := unifiedPostCtrl.NewUnifiedPostController(unifiedPostService)
	unifiedPostController.RegisterRoutes(api.Group("/unified-posts"))

	// === Cluster ===
	clusterRepository := clusterRepo.NewClusterRepository(mainDB)
	clusterService := clusterSvc.NewClusterService(clusterRepository)
	clusterController := clusterCtrl.NewClusterController(clusterService)
	clusterController.RegisterRoutes(api.Group("/clusters"))

	// === Crisis ===
	crisisRepository := crisisRepo.NewCrisisRepository(mainDB)
	crisisService := crisisSvc.NewCrisisService(crisisRepository)
	crisisController := crisisCtrl.NewCrisisController(crisisService)
	crisisController.RegisterRoutes(api.Group("/crises"))

	// === Alert ===
	alertRepository := alertRepo.NewAlertRepository(mainDB)
	alertService := alertSvc.NewAlertService(alertRepository, srv.RedisClient)
	alertController := alertCtrl.NewAlertController(alertService)
	alertController.RegisterRoutes(api.Group("/alerts"))

	// === Notify ===
	subscriptionRepo := notifyRepo.NewSubscriptionRepository(mainDB)
	notifyService := notifySvc.NewNotifyService(subscriptionRepo)
	notifyController := notifyCtrl.NewNotifyController(notifyService)
	notifyController.RegisterRoutes(api.Group("/subscribe"))

	// === Upload (S3 via AWS SDK) ===
	uploadRepository := uploadRepo.NewUploadRepository()
	uploadService    := uploadSvc.NewUploadService(uploadRepository)
	uploadController := uploadCtrl.NewUploadController(uploadService)
	uploadController.RegisterRoutes(api.Group("/upload"))

	// === Vector + Location (shared infrastructure) ===
	vr := vectorRepo.NewVectorRepository(
		database.GetTextEmbeddingsCollection(srv.VectorClient),
		database.GetImageEmbeddingsCollection(srv.VectorClient),
	)
	_ = locationRepo.NewLocationRepository(
		database.GetGeoPriorsCollection(srv.LocationClient),
		database.GetPlaceIndexCollection(srv.LocationClient),
		database.GetLocationCacheCollection(srv.LocationClient),
	)

	// === RAG + Query (conditional on ANTHROPIC_API_KEY) ===
	if os.Getenv("GOOGLE_API_KEY") != "" {
		ragService := ragSvc.NewRAGService(vr)
		queryService := querySvc.NewQueryService()
		queryController := queryCtrl.NewQueryController(queryService)
		queryController.RegisterRoutes(api.Group("/query"))

		// Pipeline scheduler — pings the Python sidecar every 60 s.
		// The Python APScheduler is authoritative; this goroutine keeps the sidecar warm.
		go func() {
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
				if err := ragService.TriggerPipeline(ctx, 0, 0); err != nil {
					log.Printf("routes: pipeline trigger error (non-critical): %v", err)
				}
				cancel()
			}
		}()

		log.Println("routes: /api/query registered; pipeline scheduler goroutine started")
	} else {
		log.Println("routes: GOOGLE_API_KEY not set — /api/query and pipeline disabled")
	}

	// ── WebSocket routes ──────────────────────────────────────────────────────
	RegisterWSRoutes(srv)

	log.Println("routes: all routes registered successfully")
}

// healthHandler returns a Fiber handler that pings all three MongoDB databases
// and Redis, then responds with a JSON status document.
func healthHandler(srv *FiberServer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()

		mainErr := srv.MainClient.Ping(ctx, nil)
		locationErr := srv.LocationClient.Ping(ctx, nil)
		vectorErr := srv.VectorClient.Ping(ctx, nil)
		redisErr := srv.RedisClient.Ping(ctx).Err()

		status := fiber.Map{
			"status":      "ok",
			"main_db":     dbStatus(mainErr),
			"location_db": dbStatus(locationErr),
			"vector_db":   dbStatus(vectorErr),
			"redis":       dbStatus(redisErr),
		}

		code := fiber.StatusOK
		if mainErr != nil || locationErr != nil || vectorErr != nil || redisErr != nil {
			status["status"] = "degraded"
			code = fiber.StatusServiceUnavailable
		}

		return c.Status(code).JSON(status)
	}
}

func dbStatus(err error) string {
	if err != nil {
		return "down"
	}
	return "up"
}
