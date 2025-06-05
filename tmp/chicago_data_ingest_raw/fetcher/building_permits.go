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

// BuildingPermit matches exactly the Socrata JSON schema you provided:
type BuildingPermit struct {
	ID                   string  `json:"id"`
	Permit               string  `json:"permit_"`
	PermitStatus         string  `json:"permit_status"`
	PermitMilestone      string  `json:"permit_milestone"`
	PermitType           string  `json:"permit_type"`
	ReviewType           string  `json:"review_type"`
	ApplicationStartDate string  `json:"application_start_date"`
	IssueDate            string  `json:"issue_date"`
	ProcessingTime       int     `json:"processing_time,string"`
	StreetNumber         string  `json:"street_number"`
	StreetDirection      string  `json:"street_direction"`
	StreetName           string  `json:"street_name"`
	WorkType             string  `json:"work_type"`
	WorkDescription      string  `json:"work_description"`
	BuildingFeePaid      float64 `json:"building_fee_paid,string"`
	ZoningFeePaid        float64 `json:"zoning_fee_paid,string"`
	OtherFeePaid         float64 `json:"other_fee_paid,string"`
	SubtotalPaid         float64 `json:"subtotal_paid,string"`
	BuildingFeeUnpaid    float64 `json:"building_fee_unpaid,string"`
	ZoningFeeUnpaid      float64 `json:"zoning_fee_unpaid,string"`
	OtherFeeUnpaid       float64 `json:"other_fee_unpaid,string"`
	SubtotalUnpaid       float64 `json:"subtotal_unpaid,string"`
	BuildingFeeWaived    float64 `json:"building_fee_waived,string"`
	BuildingFeeSubtotal  float64 `json:"building_fee_subtotal,string"`
	ZoningFeeSubtotal    float64 `json:"zoning_fee_subtotal,string"`
	OtherFeeSubtotal     float64 `json:"other_fee_subtotal,string"`
	ZoningFeeWaived      float64 `json:"zoning_fee_waived,string"`
	OtherFeeWaived       float64 `json:"other_fee_waived,string"`
	SubtotalWaived       float64 `json:"subtotal_waived,string"`
	TotalFee             float64 `json:"total_fee,string"`
	Contact1Type         string  `json:"contact_1_type"`
	Contact1Name         string  `json:"contact_1_name"`
	Contact1City         string  `json:"contact_1_city"`
	Contact1State        string  `json:"contact_1_state"`
	Contact1Zipcode      string  `json:"contact_1_zipcode"`
	Contact2Type         string  `json:"contact_2_type"`
	Contact2Name         string  `json:"contact_2_name"`
	Contact2City         string  `json:"contact_2_city"`
	Contact2State        string  `json:"contact_2_state"`
	Contact2Zipcode      string  `json:"contact_2_zipcode"`
	Contact3Type         string  `json:"contact_3_type"`
	Contact3Name         string  `json:"contact_3_name"`
	Contact3City         string  `json:"contact_3_city"`
	Contact3State        string  `json:"contact_3_state"`
	Contact3Zipcode      string  `json:"contact_3_zipcode"`
	ReportedCost         float64 `json:"reported_cost,string"`
	CommunityArea        string  `json:"community_area"`
	CensusTract          string  `json:"census_tract"`
	Ward                 string  `json:"ward"`
	XCoordinate          float64 `json:"xcoordinate,string"`
	YCoordinate          float64 `json:"ycoordinate,string"`
}

// LoadBuildingPermitsConcurrent starts N workers that fetch 2 000‐row pages
// from the Socrata “Building Permits” endpoint and COPY them into Postgres.
// On duplicate‐key (23505), it falls back to row‐by‐row INSERT … ON CONFLICT DO NOTHING.
func LoadBuildingPermitsConcurrent() {
	fmt.Println("▶ Concurrent COPY FROM using pgx.CopyFromRows for Building Permits…")

	// Socrata endpoint for “Building Permits” (no year filter)
	apiBase := "https://data.cityofchicago.org/resource/ydr8-5enu.json"
	appToken := os.Getenv("APP_TOKEN")

	const (
		limit       = 2000    // 2 000 rows per page
		maxOffset   = 3000000 // approximate total count—tweak as needed
		concurrency = 20      // number of parallel workers
	)

	offsetChan := make(chan int, concurrency)
	var wg sync.WaitGroup

	// Spawn ‘concurrency’ workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for offset := range offsetChan {
				fetchAndCopyBatchBP(apiBase, appToken, limit, offset)
			}
		}()
	}

	// Enqueue offsets: 0, 2000, 4000, … up to maxOffset
	for offset := 0; offset < maxOffset; offset += limit {
		offsetChan <- offset
	}
	close(offsetChan)
	wg.Wait()

	fmt.Println("✅ COPY Ingestion Finished.")
}

// fetchAndCopyBatchBP fetches one 2 000‐row batch from Socrata and tries to COPY it.
// If COPY fails with 23505 (duplicate‐key), it falls back to row‐by‐row INSERT.
func fetchAndCopyBatchBP(apiBase, appToken string, limit, offset int) {
	// 1) Build the Socrata URL for this batch (no time filtering):
	url := fmt.Sprintf(
		"%s?$limit=%d&$offset=%d",
		apiBase, limit, offset,
	)
	fmt.Println("🔗", url)

	// 2) HTTP GET with retry on 429 or 5xx
	var resp *http.Response
	var err error
	const maxRetries = 5
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
			break
		} else if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			log.Printf(
				"⚠️ HTTP %d at offset %d, attempt %d—retrying in %s…",
				resp.StatusCode, offset, attempt, backoff,
			)
		} else {
			log.Printf("❌ HTTP status %d at offset %d—no retry", resp.StatusCode, offset)
			if resp != nil {
				resp.Body.Close()
			}
			return
		}

		// close body, sleep, then try again
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
		lastStatus := 0
		if resp != nil {
			lastStatus = resp.StatusCode
		}
		log.Printf("❌ Giving up on offset %d, last status: %d", offset, lastStatus)
		return
	}
	defer resp.Body.Close()

	// 3) Decode JSON into []BuildingPermit
	var permits []BuildingPermit
	if err := json.NewDecoder(resp.Body).Decode(&permits); err != nil {
		log.Printf("❌ JSON decode error at offset %d: %v", offset, err)
		return
	}
	if len(permits) == 0 {
		log.Printf("✅ No records at offset %d", offset)
		return
	}

	// 4) Open a fresh pgx.Conn for this batch
	ctx := context.Background()
	conn := db.GetPgxConn() // returns *pgx.Conn
	defer conn.Close(ctx)

	// 5) Build a [][]interface{} for pgx.CopyFromRows
	const layout = "2006-01-02T15:04:05.000"
	rows := make([][]interface{}, 0, len(permits))

	for _, p := range permits {
		appStart, _ := time.Parse(layout, p.ApplicationStartDate)
		issueDate, _ := time.Parse(layout, p.IssueDate)

		row := []interface{}{
			p.ID,                  //  1) id
			p.Permit,              //  2) permit_
			p.PermitStatus,        //  3) permit_status
			p.PermitMilestone,     //  4) permit_milestone
			p.PermitType,          //  5) permit_type
			p.ReviewType,          //  6) review_type
			appStart,              //  7) application_start_date
			issueDate,             //  8) issue_date
			p.ProcessingTime,      //  9) processing_time
			p.StreetNumber,        // 10) street_number
			p.StreetDirection,     // 11) street_direction
			p.StreetName,          // 12) street_name
			p.WorkType,            // 13) work_type
			p.WorkDescription,     // 14) work_description
			p.BuildingFeePaid,     // 15) building_fee_paid
			p.ZoningFeePaid,       // 16) zoning_fee_paid
			p.OtherFeePaid,        // 17) other_fee_paid
			p.SubtotalPaid,        // 18) subtotal_paid
			p.BuildingFeeUnpaid,   // 19) building_fee_unpaid
			p.ZoningFeeUnpaid,     // 20) zoning_fee_unpaid
			p.OtherFeeUnpaid,      // 21) other_fee_unpaid
			p.SubtotalUnpaid,      // 22) subtotal_unpaid
			p.BuildingFeeWaived,   // 23) building_fee_waived
			p.BuildingFeeSubtotal, // 24) building_fee_subtotal
			p.ZoningFeeSubtotal,   // 25) zoning_fee_subtotal
			p.OtherFeeSubtotal,    // 26) other_fee_subtotal
			p.ZoningFeeWaived,     // 27) zoning_fee_waived
			p.OtherFeeWaived,      // 28) other_fee_waived
			p.SubtotalWaived,      // 29) subtotal_waived
			p.TotalFee,            // 30) total_fee
			p.Contact1Type,        // 31) contact_1_type
			p.Contact1Name,        // 32) contact_1_name
			p.Contact1City,        // 33) contact_1_city
			p.Contact1State,       // 34) contact_1_state
			p.Contact1Zipcode,     // 35) contact_1_zipcode
			p.Contact2Type,        // 36) contact_2_type
			p.Contact2Name,        // 37) contact_2_name
			p.Contact2City,        // 38) contact_2_city
			p.Contact2State,       // 39) contact_2_state
			p.Contact2Zipcode,     // 40) contact_2_zipcode
			p.Contact3Type,        // 41) contact_3_type
			p.Contact3Name,        // 42) contact_3_name
			p.Contact3City,        // 43) contact_3_city
			p.Contact3State,       // 44) contact_3_state
			p.Contact3Zipcode,     // 45) contact_3_zipcode
			p.ReportedCost,        // 46) reported_cost
			p.CommunityArea,       // 47) community_area
			p.CensusTract,         // 48) census_tract
			p.Ward,                // 49) ward
			p.XCoordinate,         // 50) xcoordinate
			p.YCoordinate,         // 51) ycoordinate
		}
		rows = append(rows, row)
	}

	// 6) Column list (must match the INSERTable columns in building_permits exactly).
	columns := []string{
		"id",
		"permit_",
		"permit_status",
		"permit_milestone",
		"permit_type",
		"review_type",
		"application_start_date",
		"issue_date",
		"processing_time",
		"street_number",
		"street_direction",
		"street_name",
		"work_type",
		"work_description",
		"building_fee_paid",
		"zoning_fee_paid",
		"other_fee_paid",
		"subtotal_paid",
		"building_fee_unpaid",
		"zoning_fee_unpaid",
		"other_fee_unpaid",
		"subtotal_unpaid",
		"building_fee_waived",
		"building_fee_subtotal",
		"zoning_fee_subtotal",
		"other_fee_subtotal",
		"zoning_fee_waived",
		"other_fee_waived",
		"subtotal_waived",
		"total_fee",
		"contact_1_type",
		"contact_1_name",
		"contact_1_city",
		"contact_1_state",
		"contact_1_zipcode",
		"contact_2_type",
		"contact_2_name",
		"contact_2_city",
		"contact_2_state",
		"contact_2_zipcode",
		"contact_3_type",
		"contact_3_name",
		"contact_3_city",
		"contact_3_state",
		"contact_3_zipcode",
		"reported_cost",
		"community_area",
		"census_tract",
		"ward",
		"xcoordinate",
		"ycoordinate",
	}

	// 7) Attempt a fast COPY INTO building_permits
	rowCount, err := conn.CopyFrom(
		ctx,
		pgx.Identifier{"building_permits"},
		columns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		// If a duplicate‐key error occurs, fall back to individual row inserts
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			log.Printf("⚠️ Duplicate-key at offset %d, retrying rows individually…", offset)
			insertRowsOneByOneBP(ctx, conn, rows)
			return
		}
		log.Printf("❌ COPY failed at offset %d: %v", offset, err)
		return
	}

	fmt.Printf("✅ Bulk-COPIED %d rows into building_permits at offset %d\n", rowCount, offset)
}

// insertRowsOneByOneBP does an INSERT … ON CONFLICT DO NOTHING for each row.
func insertRowsOneByOneBP(ctx context.Context, conn *pgx.Conn, rows [][]interface{}) {
	const sqlText = `
	  INSERT INTO building_permits (
	    id,
	    permit_,
	    permit_status,
	    permit_milestone,
	    permit_type,
	    review_type,
	    application_start_date,
	    issue_date,
	    processing_time,
	    street_number,
	    street_direction,
	    street_name,
	    work_type,
	    work_description,
	    building_fee_paid,
	    zoning_fee_paid,
	    other_fee_paid,
	    subtotal_paid,
	    building_fee_unpaid,
	    zoning_fee_unpaid,
	    other_fee_unpaid,
	    subtotal_unpaid,
	    building_fee_waived,
	    building_fee_subtotal,
	    zoning_fee_subtotal,
	    other_fee_subtotal,
	    zoning_fee_waived,
	    other_fee_waived,
	    subtotal_waived,
	    total_fee,
	    contact_1_type,
	    contact_1_name,
	    contact_1_city,
	    contact_1_state,
	    contact_1_zipcode,
	    contact_2_type,
	    contact_2_name,
	    contact_2_city,
	    contact_2_state,
	    contact_2_zipcode,
	    contact_3_type,
	    contact_3_name,
	    contact_3_city,
	    contact_3_state,
	    contact_3_zipcode,
	    reported_cost,
	    community_area,
	    census_tract,
	    ward,
	    xcoordinate,
	    ycoordinate
	  ) VALUES (
	    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,
	    $13,$14,$15,$16,$17,$18,$19,$20,$21,$22,
	    $23,$24,$25,$26,$27,$28,$29,$30,$31,$32,
	    $33,$34,$35,$36,$37,$38,$39,$40,$41,$42,
	    $43,$44,$45,$46,$47,$48,$49,$50,$51
	  )
	  ON CONFLICT (id) DO NOTHING;
	`

	for idx, r := range rows {
		if _, err := conn.Exec(ctx, sqlText, r...); err != nil {
			log.Printf("❌ row-by-row INSERT failed at sub-index %d: %v", idx, err)
		}
	}
}
