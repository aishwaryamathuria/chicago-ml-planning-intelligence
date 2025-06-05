package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"chicago_data_ingest/db"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

// TaxiTrip matches exactly the Socrata JSON schema:
type TaxiTrip struct {
	TripID               string  `json:"trip_id"`
	TaxiID               string  `json:"taxi_id"`
	TripStartTimestamp   string  `json:"trip_start_timestamp"`
	TripEndTimestamp     string  `json:"trip_end_timestamp"`
	TripSeconds          int     `json:"trip_seconds,string"`
	TripMiles            float64 `json:"trip_miles,string"`
	PickupCommunityArea  string  `json:"pickup_community_area"`
	DropoffCommunityArea string  `json:"dropoff_community_area"`
	Fare                 float64 `json:"fare,string"`
	Tips                 float64 `json:"tips,string"`
	Tolls                float64 `json:"tolls,string"`
	Extras               float64 `json:"extras,string"`
	TripTotal            float64 `json:"trip_total,string"`
	PaymentType          string  `json:"payment_type"`
	Company              string  `json:"company"`
	PickupCentroidLat    float64 `json:"pickup_centroid_latitude,string"`
	PickupCentroidLong   float64 `json:"pickup_centroid_longitude,string"`
	DropoffCentroidLat   float64 `json:"dropoff_centroid_latitude,string"`
	DropoffCentroidLong  float64 `json:"dropoff_centroid_longitude,string"`
	PickupCensusTract    string  `json:"pickup_census_tract"`
	DropoffCensusTract   string  `json:"dropoff_census_tract"`
	ComputedRegion       string  `json":@computed_region_vrxf_vc4k"`
}

// LoadTaxiConcurrent launches multiple workers that fetch 1 000‐row pages
// from the Socrata endpoint and attempt to COPY them into taxi_trips.
// On a duplicate‐key error, it falls back to inserting rows one by one.
func LoadTaxiConcurrent() {
	fmt.Println("▶ Concurrent COPY FROM using pgx.CopyFromRows for Taxi Trips (2021–2022)…")

	apiBase := "https://data.cityofchicago.org/resource/wrvz-psew.json"
	appToken := os.Getenv("APP_TOKEN")

	const (
		limit       = 1000    // 1 000 rows per batch
		maxOffset   = 8840000 // approximate total rows
		concurrency = 10      // number of parallel workers
	)

	offsetChan := make(chan int, concurrency)
	var wg sync.WaitGroup

	// Spawn 'concurrency' workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for offset := range offsetChan {
				fetchAndCopyBatch(apiBase, appToken, limit, offset)
			}
		}()
	}

	// Enqueue all offsets: 0, 1000, 2000, … up to maxOffset
	for offset := 0; offset < maxOffset; offset += limit {
		offsetChan <- offset
	}
	close(offsetChan)
	wg.Wait()

	fmt.Println("✅ COPY Ingestion Finished.")
}

// fetchAndCopyBatch requests one batch of 'limit' rows at 'offset', decodes JSON,
// and tries to bulk-COPY them into Postgres. On a duplicate-key (23505) error,
// it retries row by row with INSERT … ON CONFLICT DO NOTHING.
func fetchAndCopyBatch(apiBase, appToken string, limit, offset int) {
	// Build the Socrata URL for this batch
	url := fmt.Sprintf(
		"%s?$where=trip_start_timestamp%%20between%%20'2021-01-01T00:00:00'"+
			"%%20and%%20'2022-12-31T23:59:59'"+
			"&$order=trip_start_timestamp%%20ASC"+
			"&$limit=%d&$offset=%d",
		apiBase, limit, offset,
	)
	fmt.Println("🔗", url)

	// Issue the HTTP request
	req, _ := http.NewRequest("GET", url, nil)
	if appToken != "" {
		req.Header.Set("X-App-Token", appToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("❌ HTTP error at offset %d: %v", offset, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ HTTP status %d at offset %d", resp.StatusCode, offset)
		return
	}

	// Decode JSON into []TaxiTrip
	var trips []TaxiTrip
	if err := json.NewDecoder(resp.Body).Decode(&trips); err != nil {
		log.Printf("❌ JSON decode error at offset %d: %v", offset, err)
		return
	}
	if len(trips) == 0 {
		log.Printf("✅ No records at offset %d", offset)
		return
	}

	// Open a fresh pgx.Conn for this batch
	ctx := context.Background()
	conn := db.GetPgxConn() // returns *pgx.Conn
	defer conn.Close(ctx)

	// Build a [][]interface{} for CopyFromRows
	const layout = "2006-01-02T15:04:05.000"
	rows := make([][]interface{}, 0, len(trips))

	for _, r := range trips {
		startTime, _ := time.Parse(layout, r.TripStartTimestamp)
		endTime, _ := time.Parse(layout, r.TripEndTimestamp)

		row := []interface{}{
			r.TripID,
			r.TaxiID,
			startTime,
			endTime,
			r.TripSeconds,
			r.TripMiles,
			r.PickupCommunityArea,
			r.DropoffCommunityArea,
			r.Fare,
			r.Tips,
			r.Tolls,
			r.Extras,
			r.TripTotal,
			r.PaymentType,
			r.Company,
			r.PickupCentroidLat,
			r.PickupCentroidLong,
			r.DropoffCentroidLat,
			r.DropoffCentroidLong,
			r.PickupCensusTract,
			r.DropoffCensusTract,
			r.ComputedRegion,
		}
		rows = append(rows, row)
	}

	columns := []string{
		"trip_id",
		"taxi_id",
		"trip_start_timestamp",
		"trip_end_timestamp",
		"trip_seconds",
		"trip_miles",
		"pickup_community_area",
		"dropoff_community_area",
		"fare",
		"tips",
		"tolls",
		"extras",
		"trip_total",
		"payment_type",
		"company",
		"pickup_centroid_latitude",
		"pickup_centroid_longitude",
		"dropoff_centroid_latitude",
		"dropoff_centroid_longitude",
		"pickup_census_tract",
		"dropoff_census_tract",
		"computed_region_vrxf_vc4k",
	}

	// Attempt a fast COPY FROM STDIN → taxi_trips
	rowCount, err := conn.CopyFrom(
		ctx,
		pgx.Identifier{"taxi_trips"},
		columns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		// If it’s a duplicate-key error (23505), fall back to inserting rows one by one.
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			log.Printf("⚠️ Duplicate-key at offset %d, retrying rows individually …", offset)
			insertRowsOneByOne(ctx, conn, rows)
			return
		}
		// Any other error: log and return
		log.Printf("❌ COPY failed at offset %d: %v", offset, err)
		return
	}

	fmt.Printf("✅ Bulk-COPIED %d rows at offset %d\n", rowCount, offset)
}

// insertRowsOneByOne loops over each row and does:
//
//	INSERT … ON CONFLICT (trip_id) DO NOTHING
func insertRowsOneByOne(ctx context.Context, conn *pgx.Conn, rows [][]interface{}) {
	const sqlText = `
	  INSERT INTO taxi_trips (
	    trip_id, taxi_id, trip_start_timestamp, trip_end_timestamp,
	    trip_seconds, trip_miles, pickup_community_area, dropoff_community_area,
	    fare, tips, tolls, extras, trip_total, payment_type, company,
	    pickup_centroid_latitude, pickup_centroid_longitude,
	    dropoff_centroid_latitude, dropoff_centroid_longitude,
	    pickup_census_tract, dropoff_census_tract, computed_region_vrxf_vc4k
	  ) VALUES (
	    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22
	  )
	  ON CONFLICT (trip_id) DO NOTHING;
	`

	for idx, r := range rows {
		_, err := conn.Exec(ctx, sqlText, r...)
		if err != nil {
			log.Printf("❌ row-by-row insert failed at sub-index %d: %v", idx, err)
		}
	}
}
