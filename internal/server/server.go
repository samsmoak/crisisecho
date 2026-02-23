package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

// FiberServer is the application server. It wraps the Fiber app and holds
// references to all three MongoDB clients and the Redis client so that
// RegisterRoutes can pass them to repositories during dependency injection.
type FiberServer struct {
	App            *fiber.App
	MainClient     *mongo.Client // main database (posts, clusters, crises, alerts, …)
	LocationClient *mongo.Client // location database (geo_priors, place_index, location_cache)
	VectorClient   *mongo.Client // vector database (text_embeddings, image_embeddings)
	RedisClient    *redis.Client
}

// New constructs a FiberServer with the Fiber app configured for CrisisEcho.
func New(
	mainClient *mongo.Client,
	locationClient *mongo.Client,
	vectorClient *mongo.Client,
	redisClient *redis.Client,
) *FiberServer {
	app := fiber.New(fiber.Config{
		ServerHeader: "crisisecho",
		// Disable the default error handler stack trace in production.
		// Domain handlers return structured JSON errors directly.
		ErrorHandler: defaultErrorHandler,
	})

	return &FiberServer{
		App:            app,
		MainClient:     mainClient,
		LocationClient: locationClient,
		VectorClient:   vectorClient,
		RedisClient:    redisClient,
	}
}

// defaultErrorHandler returns a JSON error body for any unhandled Fiber error.
func defaultErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
	})
}
