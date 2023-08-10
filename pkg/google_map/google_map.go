package google_map

import (
	"log"

	"googlemaps.github.io/maps"
)

var Map *maps.Client

func init() {
	var initErr error
	Map, initErr = maps.NewClient(maps.WithAPIKey("AIzaSyAtcwUbA0jjJ6ARXl5_FqIqYcGbTI_XZEE"))
	if initErr != nil {
		log.Fatalf("fatal maps inicialization error: %s", initErr)
	}
}
