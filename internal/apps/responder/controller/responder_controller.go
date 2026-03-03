package controller

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"crisisecho/internal/apps/responder/model"
	"crisisecho/internal/apps/responder/service"
	// "crisisecho/internal/middleware"
)

// ResponderController handles HTTP requests for the responder domain.
type ResponderController struct {
	svc service.ResponderService
}

// NewResponderController constructs a ResponderController with the given service.
func NewResponderController(svc service.ResponderService) *ResponderController {
	return &ResponderController{svc: svc}
}

// RegisterRoutes mounts the responder routes on the provided router group.
func (c *ResponderController) RegisterRoutes(router fiber.Router) {
	// router.Post("/register", middleware.FirebaseAuth(), c.Register)
	// router.Get("/me", middleware.FirebaseAuth(), c.GetMe)
	// router.Put("/me", middleware.FirebaseAuth(), c.UpdateMe)
	// router.Post("/respond/:alertId", middleware.FirebaseAuth(), c.Respond)
	router.Post("/register", c.Register)
	router.Get("/me", c.GetMe)
	router.Put("/me", c.UpdateMe)
	router.Get("/near", c.FindNear)
	router.Post("/respond/:alertId", c.Respond)
	router.Put("/response/:id/status", c.UpdateResponseStatus)
	router.Put("/response/:id/rate", c.RateResponse)
}

// POST /api/responders/register
func (c *ResponderController) Register(ctx *fiber.Ctx) error {
	var responder model.Responder
	if err := ctx.BodyParser(&responder); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	// responder.UserID = ctx.Locals("firebase_uid").(string)
	if err := c.svc.RegisterResponder(ctx.UserContext(), &responder); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(responder)
}

// GET /api/responders/me
func (c *ResponderController) GetMe(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("firebase_uid").(string)
	if userID == "" {
		userID = ctx.Query("user_id")
	}
	responder, err := c.svc.GetResponderByUser(ctx.UserContext(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(responder)
}

// PUT /api/responders/me
func (c *ResponderController) UpdateMe(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("firebase_uid").(string)
	if userID == "" {
		userID = ctx.Query("user_id")
	}
	// Look up the responder to get the MongoDB ID for the update
	existing, err := c.svc.GetResponderByUser(ctx.UserContext(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	var responder model.Responder
	if err := ctx.BodyParser(&responder); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if err := c.svc.UpdateResponder(ctx.UserContext(), existing.ID.Hex(), &responder); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(responder)
}

// GET /api/responders/near?lat=&lng=&radius=
func (c *ResponderController) FindNear(ctx *fiber.Ctx) error {
	lat, err := strconv.ParseFloat(ctx.Query("lat"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lat"})
	}
	lng, err := strconv.ParseFloat(ctx.Query("lng"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lng"})
	}
	radius, _ := strconv.ParseFloat(ctx.Query("radius", "10"), 64)

	responders, err := c.svc.FindNearby(ctx.UserContext(), lat, lng, radius)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if responders == nil {
		responders = []*model.Responder{}
	}
	return ctx.JSON(responders)
}

// POST /api/responders/respond/:alertId
func (c *ResponderController) Respond(ctx *fiber.Ctx) error {
	alertID := ctx.Params("alertId")
	alertOID, err := primitive.ObjectIDFromHex(alertID)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid alertId"})
	}

	userID, _ := ctx.Locals("firebase_uid").(string)
	// Look up responder by user
	responder, err := c.svc.GetResponderByUser(ctx.UserContext(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "responder profile not found"})
	}

	var body struct {
		AlertKind string `json:"alert_kind"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.AlertKind == "" {
		body.AlertKind = "crisis"
	}

	response := &model.Response{
		ResponderID: responder.ID,
		AlertID:     alertOID,
		AlertKind:   body.AlertKind,
	}
	if err := c.svc.CreateResponse(ctx.UserContext(), response); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(response)
}

// PUT /api/responders/response/:id/status
func (c *ResponderController) UpdateResponseStatus(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var body struct {
		Status string `json:"status"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	result, err := c.svc.UpdateResponseStatus(ctx.UserContext(), id, body.Status)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(result)
}

// PUT /api/responders/response/:id/rate
func (c *ResponderController) RateResponse(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var body struct {
		Rating int `json:"rating"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	result, err := c.svc.RateResponse(ctx.UserContext(), id, body.Rating)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(result)
}
