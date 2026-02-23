package geo

// GeoJSONPoint represents a GeoJSON Point for MongoDB 2dsphere indexing.
// Coordinates are stored as [longitude, latitude] per the GeoJSON spec (RFC 7946).
type GeoJSONPoint struct {
	Type        string     `bson:"type"        json:"type"`
	Coordinates [2]float64 `bson:"coordinates" json:"coordinates"`
}

// GeoJSONPolygon represents a GeoJSON Polygon for area boundaries (e.g. cluster affected area).
// Coordinates is a slice of rings; each ring is a slice of [lng, lat] pairs.
// The first ring is the exterior; any subsequent rings are holes.
type GeoJSONPolygon struct {
	Type        string          `bson:"type"        json:"type"`
	Coordinates [][][2]float64  `bson:"coordinates" json:"coordinates"`
}

// NewPoint constructs a GeoJSONPoint from lat/lng, converting to [lng, lat] storage order.
func NewPoint(lat, lng float64) GeoJSONPoint {
	return GeoJSONPoint{
		Type:        "Point",
		Coordinates: [2]float64{lng, lat},
	}
}

// Lat returns the latitude component of the point (Coordinates[1]).
func (p GeoJSONPoint) Lat() float64 { return p.Coordinates[1] }

// Lng returns the longitude component of the point (Coordinates[0]).
func (p GeoJSONPoint) Lng() float64 { return p.Coordinates[0] }

// NewPolygon constructs a GeoJSONPolygon from a slice of [lng, lat] ring coordinates.
// rings[0] is the exterior ring; subsequent rings are holes.
func NewPolygon(rings [][][2]float64) GeoJSONPolygon {
	return GeoJSONPolygon{
		Type:        "Polygon",
		Coordinates: rings,
	}
}
