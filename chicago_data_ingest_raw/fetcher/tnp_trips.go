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

// TNPTrip matches exactly the Socrata JSON schema for the TNP-Trips endpoint:
type TNPTrip struct {
	TripID               string  `json:"trip_id"`
	TripStartTimestamp   string  `json:"trip_start_timestamp"`
	TripEndTimestamp     string  `json:"trip_end_timestamp"`
	TripSeconds          int     `json:"trip_seconds,string"`
	TripMiles            float64 `json:"trip_miles,string"`
	PickupCommunityArea  string  `json:"pickup_community_area"`
	DropoffCommunityArea string  `json:"dropoff_community_area"`
	PickupCensusTract    string  `json:"pickup_census_tract"`
	DropoffCensusTract   string  `json:"dropoff_census_tract"`
	Fare                 float64 `json:"fare,string"`
	Tip                  float64 `json:"tip,string"`
	AdditionalCharges    float64 `json:"additional_charges,string"`
	TripTotal            float64 `json:"trip_total,string"`
	SharedTripAuthorized bool    `json:"shared_trip_authorized"`
	TripsPooled          int     `json:"trips_pooled,string"`
	PickupCentroidLat    float64 `json:"pickup_centroid_latitude,string"`
	PickupCentroidLong   float64 `json:"pickup_centroid_longitude,string"`
	PickupCentroidLoc    struct {
		Type        string    `json:"type"`
		Coordinates []float64 `json:"coordinates"`
	} `json:"pickup_centroid_location"`
	DropoffCentroidLat  float64 `json:"dropoff_centroid_latitude,string"`
	DropoffCentroidLong float64 `json:"dropoff_centroid_longitude,string"`
	DropoffCentroidLoc  struct {
		Type        string    `json:"type"`
		Coordinates []float64 `json:"coordinates"`
	} `json:"dropoff_centroid_location"`
}

// LoadTNPConcurrent launches multiple workers (Concurrency = 10) that:
//  1. Fetch 1 000-row pages of TNP trips for 2021–2022
//  2. Attempt a fast COPY FROM into tnp_trips
//  3. On a 23505 duplicate-key error, fall back to per-row
//     INSERT … ON CONFLICT DO NOTHING.
func LoadTNPConcurrent() {
	fmt.Println("▶ Concurrent COPY FROM using pgx.CopyFromRows for TNP Trips (2021–2022)…")

	apiBase := "https://data.cityofchicago.org/resource/m6dm-c72p.json"
	appToken := os.Getenv("APP_TOKEN")

	const (
		limit       = 2000    // pull 1 000 rows at a time
		maxOffset   = 8840000 // approximate total TNP rows for 2018–2022
		concurrency = 20      // number of parallel workers
	)

	offsetChan := make(chan int, concurrency)
	var wg sync.WaitGroup

	// Spawn ‘concurrency’ worker goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for offset := range offsetChan {
				fetchAndCopyBatchTNP(apiBase, appToken, limit, offset)
			}
		}()
	}

	// Enqueue offsets 0, 1000, 2000, … up to maxOffset
	for offset := 0; offset < maxOffset; offset += limit {
		offsetChan <- offset
	}
	close(offsetChan)

	wg.Wait()
	fmt.Println("✅ COPY Ingestion Finished.")
}

// fetchAndCopyBatch does:
//   - build a URL that filters trip_start_timestamp to 2021–2022,
//     orders by trip_start_timestamp ASC,
//     applies $limit and $offset
//   - issue HTTP GET, parse JSON into []TNPTrip
//   - try a bulk COPY INTO tnp_trips
//   - on 23505 (duplicate-key), fall back to row-by-row INSERT … ON CONFLICT DO NOTHING.
func fetchAndCopyBatchTNP(
	apiBase, appToken string,
	limit, offset int,
) {
	// Build the Socrata URL for this batch (only 2021–2022)
	url := fmt.Sprintf(
		"%s?$where=trip_start_timestamp%%20between%%20'2021-01-01T00:00:00'"+
			"%%20and%%20'2022-12-31T23:59:59'"+
			"&$order=trip_start_timestamp%%20ASC"+
			"&$limit=%d&$offset=%d",
		apiBase, limit, offset,
	)
	fmt.Println("🔗", url)

	// Retry logic for HTTP 429 / transient errors
	var resp *http.Response
	var err error
	maxRetries := 5
	backoff := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, _ := http.NewRequest("GET", url, nil)
		if appToken != "" {
			req.Header.Set("X-App-Token", appToken)
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("⚠️ HTTP error at offset %d, attempt %d: %v", offset, attempt, err)
		} else if resp.StatusCode == http.StatusOK {
			// success—break out of retry loop
			break
		} else if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			// 429 or 5xx → retryable
			log.Printf("⚠️ HTTP %d at offset %d, attempt %d—retrying in %s…",
				resp.StatusCode, offset, attempt, backoff)
		} else {
			// any other status (e.g. 400) is not retryable
			log.Printf("❌ HTTP status %d at offset %d—giving up", resp.StatusCode, offset)
			resp.Body.Close()
			return
		}

		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(backoff)
		backoff *= 2
	}

	if err != nil {
		log.Printf("❌ HTTP error at offset %d after retries: %v", offset, err)
		return
	}
	if resp == nil || resp.StatusCode != http.StatusOK {
		log.Printf("❌ Giving up on offset %d, last status: %v", offset,
			func() int {
				if resp != nil {
					return resp.StatusCode
				}
				return 0
			}(),
		)
		return
	}
	defer resp.Body.Close()

	// …rest of the function remains unchanged…
	// 3) Decode JSON into []TNPTrip
	var trips []TNPTrip
	if err := json.NewDecoder(resp.Body).Decode(&trips); err != nil {
		log.Printf("❌ JSON decode error at offset %d: %v", offset, err)
		return
	}
	if len(trips) == 0 {
		log.Printf("✅ No records at offset %d", offset)
		return
	}

	// 4) Open a fresh pgx.Conn for this batch
	ctx := context.Background()
	conn := db.GetPgxConn() // returns *pgx.Conn
	defer conn.Close(ctx)

	// 5) Build [][]interface{} for CopyFromRows
	const layout = "2006-01-02T15:04:05.000"
	rows := make([][]interface{}, 0, len(trips))
	for _, r := range trips {
		startTime, _ := time.Parse(layout, r.TripStartTimestamp)
		endTime, _ := time.Parse(layout, r.TripEndTimestamp)
		row := []interface{}{
			r.TripID,
			startTime,
			endTime,
			r.TripSeconds,
			r.TripMiles,
			r.PickupCommunityArea,
			r.DropoffCommunityArea,
			r.PickupCensusTract,
			r.DropoffCensusTract,
			r.Fare,
			r.Tip,
			r.AdditionalCharges,
			r.TripTotal,
			r.SharedTripAuthorized,
			r.TripsPooled,
			r.PickupCentroidLat,
			r.PickupCentroidLong,
			r.DropoffCentroidLat,
			r.DropoffCentroidLong,
		}
		rows = append(rows, row)
	}

	columns := []string{
		"trip_id",
		"trip_start_timestamp",
		"trip_end_timestamp",
		"trip_seconds",
		"trip_miles",
		"pickup_community_area",
		"dropoff_community_area",
		"pickup_census_tract",
		"dropoff_census_tract",
		"fare",
		"tip",
		"additional_charges",
		"trip_total",
		"shared_trip_authorized",
		"trips_pooled",
		"pickup_centroid_latitude",
		"pickup_centroid_longitude",
		"dropoff_centroid_latitude",
		"dropoff_centroid_longitude",
	}

	// 7) Attempt COPY
	rowCount, err := conn.CopyFrom(
		ctx,
		pgx.Identifier{"tnp_trips"},
		columns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			log.Printf("⚠️ Duplicate-key at offset %d, retrying rows individually…", offset)
			insertRowsOneByOneTNP(ctx, conn, rows)
			return
		}
		log.Printf("❌ COPY failed at offset %d: %v", offset, err)
		return
	}

	fmt.Printf("✅ Bulk-COPIED %d rows into tnp_trips at offset %d\n", rowCount, offset)
}

// insertRowsOneByOne does INSERT … ON CONFLICT DO NOTHING per row.
// This is only called if the bulk COPY encountered a duplicate-key (23505).
func insertRowsOneByOneTNP(ctx context.Context, conn *pgx.Conn, rows [][]interface{}) {
	const sqlText = `
	  INSERT INTO tnp_trips (
	    trip_id,
	    trip_start_timestamp,
	    trip_end_timestamp,
	    trip_seconds,
	    trip_miles,
	    pickup_community_area,
	    dropoff_community_area,
	    pickup_census_tract,
	    dropoff_census_tract,
	    fare,
	    tip,
	    additional_charges,
	    trip_total,
	    shared_trip_authorized,
	    trips_pooled,
	    pickup_centroid_latitude,
	    pickup_centroid_longitude,
	    dropoff_centroid_latitude,
	    dropoff_centroid_longitude
	  ) VALUES (
	    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19
	  )
	  ON CONFLICT (trip_id) DO NOTHING;
	`

	for idx, r := range rows {
		if _, err := conn.Exec(ctx, sqlText, r...); err != nil {
			log.Printf("❌ row-by-row INSERT failed at sub-index %d: %v", idx, err)
		}
	}
}
