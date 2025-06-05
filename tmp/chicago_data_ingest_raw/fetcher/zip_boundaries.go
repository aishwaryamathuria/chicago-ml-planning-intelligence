package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"chicago_data_ingest/db"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

// ZipBoundary matches Socrata’s JSON schema for “Boundaries – ZIP Codes”.
//
//   - the_geom is nested GeoJSON (MultiPolygon).  We keep it as json.RawMessage
//     and then turn it into a plain string when building COPY rows.
//   - objectid, zip, shape_area, shape_len map exactly.
type ZipBoundary struct {
	TheGeom   json.RawMessage `json:"the_geom"` // nested GeoJSON MultiPolygon
	ObjectID  string          `json:"objectid"` // unique ID as string
	Zip       string          `json:"zip"`      // 5-digit ZIP code
	ShapeArea float64         `json:"shape_area,string"`
	ShapeLen  float64         `json:"shape_len,string"`
}

// LoadZipBoundariesConcurrent pages through Socrata’s “ZIP Boundaries” feed
// and does a concurrent COPY INTO zip_boundaries.  On a 23505 duplicate-key,
// it falls back to row-by-row INSERT … ON CONFLICT DO NOTHING.
func LoadZipBoundariesConcurrent() {
	fmt.Println("▶ Concurrent COPY FROM using pgx.CopyFromRows for ZIP Boundaries…")

	apiBase := "https://data.cityofchicago.org/resource/unjd-c2ca.json"
	appToken := os.Getenv("APP_TOKEN")

	const (
		limit       = 2000            // fetch 2 000 rows per batch
		maxOffset   = 60000           // upper bound; Socrata will return [] once exhausted
		concurrency = 10              // number of parallel workers
		retryDelay  = 2 * time.Second // wait if 429 or 5xx
		maxRetries  = 3
	)

	offsetChan := make(chan int, concurrency)
	var wg sync.WaitGroup

	// Spawn “concurrency” worker goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for offset := range offsetChan {
				fetchAndCopyBatchZip(apiBase, appToken, limit, offset, retryDelay, maxRetries)
			}
		}()
	}

	// Enqueue offsets: 0, limit, 2*limit, … up to maxOffset
	for offset := 0; offset < maxOffset; offset += limit {
		offsetChan <- offset
	}
	close(offsetChan)

	wg.Wait()
	fmt.Println("✅ ZIP Boundaries ingestion finished.")
}

func fetchAndCopyBatchZip(
	apiBase, appToken string,
	limit, offset int,
	retryDelay time.Duration,
	maxRetries int,
) {
	// 1) Build the Socrata URL for this batch
	url := fmt.Sprintf("%s?$limit=%d&$offset=%d", apiBase, limit, offset)

	var resp *http.Response
	var err error

	// 2) Issue HTTP GET with retry on 429 or 5xx
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, _ := http.NewRequest("GET", url, nil)
		if appToken != "" {
			req.Header.Set("X-App-Token", appToken)
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("❌ HTTP error at offset %d: %v", offset, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			log.Printf("⚠️ HTTP %d at offset %d, retrying in %v…", resp.StatusCode, offset, retryDelay)
			time.Sleep(retryDelay)
			continue
		}
		break
	}

	if resp == nil {
		log.Printf("❌ No HTTP response at offset %d", offset)
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ HTTP status %d at offset %d—skipping batch", resp.StatusCode, offset)
		return
	}

	// 3) Decode JSON into []ZipBoundary
	var rowsJSON []ZipBoundary
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&rowsJSON); err != nil {
		if err == io.EOF {
			log.Printf("✅ No records at offset %d", offset)
			return
		}
		log.Printf("❌ JSON decode error at offset %d: %v", offset, err)
		return
	}
	if len(rowsJSON) == 0 {
		log.Printf("✅ No records at offset %d", offset)
		return
	}

	// 4) Open a fresh pgx.Conn for this batch
	ctx := context.Background()
	conn := db.GetPgxConn() // adjust to your own helper that returns *pgx.Conn
	defer conn.Close(ctx)

	// 5) Build [][]interface{} for CopyFromRows.  **Crucial**: we do `string(r.TheGeom)`
	// so that COPY sends plain GeoJSON text into the JSON column.
	rows := make([][]interface{}, 0, len(rowsJSON))
	for _, r := range rowsJSON {
		row := []interface{}{
			// 1) the_geom as TEXT (GeoJSON) → will land in a JSON column
			string(r.TheGeom),
			// 2) objectid
			r.ObjectID,
			// 3) zip
			r.Zip,
			// 4) shape_area
			r.ShapeArea,
			// 5) shape_len
			r.ShapeLen,
		}
		rows = append(rows, row)
	}

	// 6) The COPY‐column list must match exactly zip_boundaries(...) in DDL
	columns := []string{
		"the_geom",   // JSON
		"objectid",   // TEXT
		"zip",        // TEXT
		"shape_area", // DOUBLE PRECISION
		"shape_len",  // DOUBLE PRECISION
	}

	// 7) Attempt COPY FROM STDIN → zip_boundaries
	rowCount, err := conn.CopyFrom(
		ctx,
		pgx.Identifier{"zip_boundaries"},
		columns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		// If duplicate‐key (23505), fall back to row-by-row
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			log.Printf("⚠️ Duplicate-key at offset %d, retrying rows individually…", offset)
			insertRowsOneByOneZip(ctx, conn, rows)
			return
		}
		log.Printf("❌ COPY failed at offset %d: %v", offset, err)
		return
	}

	fmt.Printf("✅ Bulk-COPIED %d rows into zip_boundaries at offset %d\n", rowCount, offset)
}

// insertRowsOneByOneZip does INSERT … ON CONFLICT DO NOTHING per row.
func insertRowsOneByOneZip(ctx context.Context, conn *pgx.Conn, rows [][]interface{}) {
	const sqlText = `
	  INSERT INTO zip_boundaries (
	    the_geom,   -- JSON column
	    objectid,
	    zip,
	    shape_area,
	    shape_len
	  ) VALUES (
	    $1::json,
	    $2,
	    $3,
	    $4,
	    $5
	  )
	  ON CONFLICT (objectid) DO NOTHING;
	`

	for idx, r := range rows {
		if _, err := conn.Exec(ctx, sqlText, r...); err != nil {
			log.Printf("❌ row-by-row INSERT failed at sub-index %d: %v", idx, err)
		}
	}
}
