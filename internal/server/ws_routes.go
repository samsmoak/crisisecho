package server

import (
	"context"
	"log"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// RegisterWSRoutes mounts WebSocket endpoints on the Fiber app.
func RegisterWSRoutes(srv *FiberServer) {
	// Capture the Redis client in a local variable so the closure below can access it
	// without holding a reference to the entire FiberServer.
	redisClient := srv.RedisClient

	// Upgrade middleware: only requests carrying the Upgrade: websocket header proceed.
	srv.App.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// GET /ws/alerts — streams live alert events to subscribed clients.
	// Subscribes to the Redis "alerts:live" Pub/Sub channel and forwards every
	// published message as a WebSocket text frame.
	srv.App.Get("/ws/alerts", websocket.New(func(c *websocket.Conn) {
		pubsub := redisClient.Subscribe(context.Background(), "alerts:live")
		defer pubsub.Close()

		// Goroutine: read from the WebSocket to detect client disconnects.
		// Closes the done channel when the client closes the connection or sends an error.
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				if _, _, err := c.ReadMessage(); err != nil {
					return // client disconnected or error
				}
			}
		}()

		ch := pubsub.Channel()
		for {
			select {
			case <-done:
				return // client disconnected

			case msg, ok := <-ch:
				if !ok {
					return // Redis subscription closed
				}
				if err := c.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
					log.Printf("ws_routes: write error: %v", err)
					return
				}
			}
		}
	}))
}

