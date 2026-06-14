package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		places, err := app.FindCollectionByNameOrId("places")
		if err != nil {
			return err
		}

		// 1. Add the new geoPoint field
		places.Fields.Add(&core.GeoPointField{
			Name: "location",
		})
		if err := app.Save(places); err != nil {
			return err
		}

		// 2. Backfill location from existing latitude/longitude values
		records, err := app.FindAllRecords("places")
		if err != nil {
			return err
		}
		for _, r := range records {
			lat := r.GetFloat("latitude")
			lon := r.GetFloat("longitude")
			// Only backfill non-zero coordinates
			if lat != 0 || lon != 0 {
				r.Set("location", types.GeoPoint{Lat: lat, Lon: lon})
				if err := app.Save(r); err != nil {
					return err
				}
			}
		}

		// 3. Remove the old latitude and longitude fields
		places, err = app.FindCollectionByNameOrId("places")
		if err != nil {
			return err
		}
		places.Fields.RemoveByName("latitude")
		places.Fields.RemoveByName("longitude")
		return app.Save(places)
	}, func(app core.App) error {
		// Down: restore latitude/longitude and backfill from location, then remove location
		places, err := app.FindCollectionByNameOrId("places")
		if err != nil {
			return err
		}

		// 1. Re-add latitude and longitude fields
		places.Fields.Add(&core.NumberField{
			Name:     "latitude",
			Required: true,
		})
		places.Fields.Add(&core.NumberField{
			Name:     "longitude",
			Required: true,
		})
		if err := app.Save(places); err != nil {
			return err
		}

		// 2. Backfill lat/lon from location
		records, err := app.FindAllRecords("places")
		if err != nil {
			return err
		}
		for _, r := range records {
			loc := r.GetGeoPoint("location")
			r.Set("latitude", loc.Lat)
			r.Set("longitude", loc.Lon)
			if err := app.Save(r); err != nil {
				return err
			}
		}

		// 3. Remove location field
		places, err = app.FindCollectionByNameOrId("places")
		if err != nil {
			return err
		}
		places.Fields.RemoveByName("location")
		return app.Save(places)
	})
}
