package google_map

import (
	"context"
	"log"

	"googlemaps.github.io/maps"
)

type Point struct{ X, Y float64 }

// traditional function
func GetRoute(from, to string) []maps.Route {
	request := &maps.DirectionsRequest{
		Origin:      from,
		Destination: to,
	}
	route, _, err := Map.Directions(context.Background(), request)
	if err != nil {
		log.Fatalf("fatal direction error: %s", err)
	}
	return route
}
