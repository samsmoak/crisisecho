package controller

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	queryModel   "crisisecho/internal/apps/query/model"
	queryService "crisisecho/internal/apps/query/service"
)

// QueryController handles natural-language crisis query requests.
type QueryController struct {
	service queryService.QueryService
}

// NewQueryController constructs a QueryController.
func NewQueryController(svc queryService.QueryService) *QueryController {
	return &QueryController{service: svc}
}

// RegisterRoutes mounts the query endpoint on the given router group.
func (ctrl *QueryController) RegisterRoutes(router fiber.Router) {
	router.Get("/", ctrl.RunQuery)
}

// RunQuery handles GET /api/query?text=...&lat=...&lng=...&radius=...
func (ctrl *QueryController) RunQuery(c *fiber.Ctx) error {
	text := c.Query("text")
	if text == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "text query parameter is required",
		})
	}

	lat, err := strconv.ParseFloat(c.Query("lat", "0"), 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lat"})
	}
	lng, err := strconv.ParseFloat(c.Query("lng", "0"), 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lng"})
	}
	radius, _ := strconv.ParseFloat(c.Query("radius", "50"), 64)

	req := &queryModel.QueryRequest{
		Text:   text,
		Lat:    lat,
		Lng:    lng,
		Radius: radius,
	}

	result, err := ctrl.service.RunQuery(c.UserContext(), req)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
}
