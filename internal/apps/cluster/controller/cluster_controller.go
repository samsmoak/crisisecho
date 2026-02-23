package controller

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"crisisecho/internal/apps/cluster/model"
	"crisisecho/internal/apps/cluster/service"
)

// ClusterController handles HTTP requests for the cluster domain.
type ClusterController struct {
	svc service.ClusterService
}

// NewClusterController constructs a ClusterController with the given service.
func NewClusterController(svc service.ClusterService) *ClusterController {
	return &ClusterController{svc: svc}
}

// RegisterRoutes mounts the cluster routes on the provided router group.
// /hotspots is registered before /:id to prevent "hotspots" being captured as an ID param.
func (c *ClusterController) RegisterRoutes(router fiber.Router) {
	router.Get("/hotspots", c.GetHotspots)
	router.Get("/:id", c.GetClusterDetail)
}

// GET /api/clusters/hotspots?lat=&lng=&radius=
// Returns active clusters near the given coordinates.
func (c *ClusterController) GetHotspots(ctx *fiber.Ctx) error {
	lat, err := strconv.ParseFloat(ctx.Query("lat"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lat"})
	}
	lng, err := strconv.ParseFloat(ctx.Query("lng"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lng"})
	}
	radius, err := strconv.ParseFloat(ctx.Query("radius", "50"), 64)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid radius"})
	}

	clusters, err := c.svc.GetHotspots(ctx.UserContext(), lat, lng, radius)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if clusters == nil {
		clusters = []*model.Cluster{}
	}
	return ctx.JSON(clusters)
}

// GET /api/clusters/:id
// Returns the full detail of a single cluster by its ObjectID.
func (c *ClusterController) GetClusterDetail(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	cluster, err := c.svc.GetClusterDetail(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(cluster)
}
