package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"chicago_data_ingest/db"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

// CovidZip mirrors exactly Socrata’s JSON schema for /yhhz-zm2v.json.
// All “numeric” fields come in as JSON strings; we parse them below.
// The computed regions use the special JSON keys starting with ":".
type CovidZip struct {
	ZipCode                         string `json:"zip_code"`
	WeekNumber                      string `json:"week_number"`
	WeekStart                       string `json:"week_start"`
	WeekEnd                         string `json:"week_end"`
	CasesWeekly                     string `json:"cases_weekly"`
	CasesCumulative                 string `json:"cases_cumulative"`
	CaseRateWeekly                  string `json:"case_rate_weekly"`
	CaseRateCumulative              string `json:"case_rate_cumulative"`
	TestsWeekly                     string `json:"tests_weekly"`
	TestsCumulative                 string `json:"tests_cumulative"`
	TestRateWeekly                  string `json:"test_rate_weekly"`
	TestRateCumulative              string `json:"test_rate_cumulative"`
	PercentTestedPositiveWeekly     string `json:"percent_tested_positive_weekly"`
	PercentTestedPositiveCumulative string `json:"percent_tested_positive_cumulative"`
	DeathsWeekly                    string `json:"deaths_weekly"`
	DeathsCumulative                string `json:"deaths_cumulative"`
	DeathRateWeekly                 string `json:"death_rate_weekly"`
	DeathRateCumulative             string `json:"death_rate_cumulative"`
	Population                      string `json:"population"`
	RowID                           string `json:"row_id"`
	ZipCodeLocation                 struct {
		Type        string    `json:"type"`
		Coordinates []float64 `json:"coordinates"` // [lon, lat]
	} `json:"zip_code_location"`
	ComputedRegionRpca8um6 string `json:":@computed_region_rpca_8um6"`
	ComputedRegionVrxfVc4k string `json:":@computed_region_vrxf_vc4k"`
	ComputedRegion6mkvF3dw string `json:":@computed_region_6mkv_f3dw"`
	ComputedRegionBdys3d7i string `json:":@computed_region_bdys_3d7i"`
	ComputedRegion43wa7qmu string `json:":@computed_region_43wa_7qmu"`
}

// parseInt returns (value, true) if s ≠ "" and s ≠ "." and parses to int64.
// Otherwise it returns (0, false), which means “insert NULL.”
func parseInt(s string) (int64, bool) {
	if s == "" || s == "." {
		return 0, false
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return i, true
}

// parseFloat returns (value, true) if s ≠ "" and s ≠ "." and parses to float64.
// Otherwise it returns (0.0, false), which means “insert NULL.”
func parseFloat(s string) (float64, bool) {
	if s == "" || s == "." {
		return 0.0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0, false
	}
	return f, true
}

// LoadCovidZipConcurrent pages through “COVID-19 by ZIP (historical)” and COPY→postgres.
func LoadCovidZipConcurrent() {
	fmt.Println("▶ Concurrent COPY FROM using pgx.CopyFromRows for COVID-19 ZIP (historical)…")

	apiBase := "https://data.cityofchicago.org/resource/yhhz-zm2v.json"
	appToken := os.Getenv("APP_TOKEN")

	const (
		limit       = 2000  // 2 000 rows per Socrata page
		maxOffset   = 20000 // way larger than actual; Socrata will eventually return zero‐rows
		concurrency = 10    // 10 parallel workers
	)

	offsetCh := make(chan int, concurrency)
	var wg sync.WaitGroup

	// Spawn worker goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for offset := range offsetCh {
				fetchAndCopyBatchCovid(apiBase, appToken, limit, offset)
			}
		}()
	}

	// Enqueue offsets: 0, 2000, 4000, … until we see no more records
	for offset := 0; offset < maxOffset; offset += limit {
		offsetCh <- offset
	}
	close(offsetCh)
	wg.Wait()

	fmt.Println("✅ All COVID-19 ZIP pages have been ingested.")
}

// fetchAndCopyBatchCovid fetches one page (limit, offset) from Socrata and attempts a bulk COPY.
// On 23505 “duplicate-key,” it falls back to per-row INSERT … ON CONFLICT DO NOTHING.
func fetchAndCopyBatchCovid(apiBase, appToken string, limit, offset int) {
	// 1) Build the Socrata URL (no date filter because “historical” covers everything).
	url := fmt.Sprintf(
		"%s?$order=week_start%%20ASC&$limit=%d&$offset=%d",
		apiBase, limit, offset,
	)
	fmt.Println("🔗", url)

	// 2) HTTP GET with up to 3 retries if we get 429
	var resp *http.Response
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		req, _ := http.NewRequest("GET", url, nil)
		if appToken != "" {
			req.Header.Set("X-App-Token", appToken)
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("❌ HTTP error at offset %d: %v", offset, err)
			return
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			backoff := time.Duration(attempt) * time.Second
			log.Printf("⚠️ HTTP 429 at offset %d, retrying in %v…", offset, backoff)
			time.Sleep(backoff)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			log.Printf("❌ HTTP status %d at offset %d", resp.StatusCode, offset)
			resp.Body.Close()
			return
		}
		// 200 OK → proceed
		break
	}
	if resp == nil || resp.StatusCode != http.StatusOK {
		log.Printf("❌ Giving up on offset %d after retries", offset)
		return
	}
	defer resp.Body.Close()

	// 3) Decode JSON into []CovidZip
	var records []CovidZip
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&records); err != nil {
		log.Printf("❌ JSON decode error at offset %d: %v", offset, err)
		return
	}
	if len(records) == 0 {
		log.Printf("✅ No more records at offset %d; stopping.", offset)
		return
	}

	// 4) Open a fresh pgx.Conn for this batch
	ctx := context.Background()
	conn := db.GetPgxConn()
	defer conn.Close(ctx)

	// 5) Build [][]interface{} for CopyFromRows
	rows := make([][]interface{}, 0, len(records))
	const layout = "2006-01-02T15:04:05.000" // Socrata timestamp format

	for _, rec := range records {
		// Required fields:
		zipCode := rec.ZipCode
		if zipCode == "" {
			continue // skip if somehow missing
		}

		//  week_number → int
		wn, wnOK := parseInt(rec.WeekNumber)
		if !wnOK {
			continue
		}

		//  week_start → time.Time
		ws, _ := time.Parse(layout, rec.WeekStart)
		we, _ := time.Parse(layout, rec.WeekEnd)

		//  All numeric fields → parseInt / parseFloat
		cw, cwOK := parseInt(rec.CasesWeekly)
		cc, ccOK := parseInt(rec.CasesCumulative)
		crw, crwOK := parseFloat(rec.CaseRateWeekly)
		crc, crcOK := parseFloat(rec.CaseRateCumulative)
		tw, twOK := parseInt(rec.TestsWeekly)
		tc, tcOK := parseInt(rec.TestsCumulative)
		trw, trwOK := parseFloat(rec.TestRateWeekly)
		trc, trcOK := parseFloat(rec.TestRateCumulative)
		ptpw, ptpwOK := parseFloat(rec.PercentTestedPositiveWeekly)
		ptpc, ptpcOK := parseFloat(rec.PercentTestedPositiveCumulative)
		dw, dwOK := parseInt(rec.DeathsWeekly)
		dc, dcOK := parseInt(rec.DeathsCumulative)
		drw, drwOK := parseFloat(rec.DeathRateWeekly)
		drc, drcOK := parseFloat(rec.DeathRateCumulative)
		pop, popOK := parseInt(rec.Population)

		//  row_id → string, always present
		rowID := rec.RowID

		//  zip_code_location → coords[0]=lon, coords[1]=lat
		var zipLon, zipLat float64
		if len(rec.ZipCodeLocation.Coordinates) == 2 {
			zipLon = rec.ZipCodeLocation.Coordinates[0]
			zipLat = rec.ZipCodeLocation.Coordinates[1]
		}

		//  computed regions → strings or empty
		crRpca, crRpcaOK := rec.ComputedRegionRpca8um6, (rec.ComputedRegionRpca8um6 != "")
		crVrxf, crVrxfOK := rec.ComputedRegionVrxfVc4k, (rec.ComputedRegionVrxfVc4k != "")
		cr6mkv, cr6mkvOK := rec.ComputedRegion6mkvF3dw, (rec.ComputedRegion6mkvF3dw != "")
		crBdys, crBdysOK := rec.ComputedRegionBdys3d7i, (rec.ComputedRegionBdys3d7i != "")
		cr43wa, cr43waOK := rec.ComputedRegion43wa7qmu, (rec.ComputedRegion43wa7qmu != "")

		// Build exactly 27 columns in the same order as our CREATE TABLE:
		// 1 zip_code, 2 week_number, 3 week_start, 4 week_end,
		// 5 cases_weekly, 6 cases_cumulative, 7 case_rate_weekly, 8 case_rate_cumulative,
		// 9 tests_weekly, 10 tests_cumulative, 11 test_rate_weekly, 12 test_rate_cumulative,
		// 13 percent_tested_positive_weekly, 14 percent_tested_positive_cumulative,
		// 15 deaths_weekly, 16 deaths_cumulative, 17 death_rate_weekly, 18 death_rate_cumulative,
		// 19 population, 20 row_id,
		// 21 zip_lat, 22 zip_long,
		// 23 computed_region_rpca_8um6, 24 computed_region_vrxf_vc4k, 25 computed_region_6mkv_f3dw,
		// 26 computed_region_bdys_3d7i, 27 computed_region_43wa_7qmu.
		row := make([]interface{}, 27)
		row[0] = zipCode
		row[1] = wn
		row[2] = ws
		row[3] = we

		if cwOK {
			row[4] = cw
		} else {
			row[4] = nil
		}
		if ccOK {
			row[5] = cc
		} else {
			row[5] = nil
		}
		if crwOK {
			row[6] = crw
		} else {
			row[6] = nil
		}
		if crcOK {
			row[7] = crc
		} else {
			row[7] = nil
		}

		if twOK {
			row[8] = tw
		} else {
			row[8] = nil
		}
		if tcOK {
			row[9] = tc
		} else {
			row[9] = nil
		}
		if trwOK {
			row[10] = trw
		} else {
			row[10] = nil
		}
		if trcOK {
			row[11] = trc
		} else {
			row[11] = nil
		}

		if ptpwOK {
			row[12] = ptpw
		} else {
			row[12] = nil
		}
		if ptpcOK {
			row[13] = ptpc
		} else {
			row[13] = nil
		}

		if dwOK {
			row[14] = dw
		} else {
			row[14] = nil
		}
		if dcOK {
			row[15] = dc
		} else {
			row[15] = nil
		}
		if drwOK {
			row[16] = drw
		} else {
			row[16] = nil
		}
		if drcOK {
			row[17] = drc
		} else {
			row[17] = nil
		}

		if popOK {
			row[18] = pop
		} else {
			row[18] = nil
		}
		row[19] = rowID

		// NOTE: order in CREATE TABLE was zip_lat THEN zip_long
		row[20] = zipLat
		row[21] = zipLon

		if crRpcaOK {
			row[22] = crRpca
		} else {
			row[22] = nil
		}
		if crVrxfOK {
			row[23] = crVrxf
		} else {
			row[23] = nil
		}
		if cr6mkvOK {
			row[24] = cr6mkv
		} else {
			row[24] = nil
		}
		if crBdysOK {
			row[25] = crBdys
		} else {
			row[25] = nil
		}
		if cr43waOK {
			row[26] = cr43wa
		} else {
			row[26] = nil
		}

		rows = append(rows, row)
	}

	// 6) Columns must exactly match the order in CREATE TABLE above:
	columns := []string{
		"zip_code",
		"week_number",
		"week_start",
		"week_end",
		"cases_weekly",
		"cases_cumulative",
		"case_rate_weekly",
		"case_rate_cumulative",
		"tests_weekly",
		"tests_cumulative",
		"test_rate_weekly",
		"test_rate_cumulative",
		"percent_tested_positive_weekly",
		"percent_tested_positive_cumulative",
		"deaths_weekly",
		"deaths_cumulative",
		"death_rate_weekly",
		"death_rate_cumulative",
		"population",
		"row_id",
		"zip_lat",
		"zip_long",
		"computed_region_rpca_8um6",
		"computed_region_vrxf_vc4k",
		"computed_region_6mkv_f3dw",
		"computed_region_bdys_3d7i",
		"computed_region_43wa_7qmu",
	}

	// 7) Attempt fast COPY FROM STDIN → covid_zip_historical
	nCopied, err := conn.CopyFrom(
		ctx,
		pgx.Identifier{"covid_zip_historical"},
		columns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		// On duplicate-key (23505), fallback to per-row INSERT … ON CONFLICT
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			log.Printf("⚠️ Duplicate-key at offset %d, retrying rows individually…", offset)
			insertRowsOneByOneCovid(ctx, conn, rows)
			return
		}
		log.Printf("❌ COPY failed at offset %d: %v", offset, err)
		return
	}

	fmt.Printf("✅ Bulk-COPIED %d rows into covid_zip_historical at offset %d\n", nCopied, offset)
}

// insertRowsOneByOneCovid does INSERT … ON CONFLICT DO NOTHING per row.
// Only invoked when the bulk COPY complains about a 23505 duplicate-key.
func insertRowsOneByOneCovid(ctx context.Context, conn *pgx.Conn, rows [][]interface{}) {
	const sqlText = `
	  INSERT INTO covid_zip_historical (
	    zip_code,
	    week_number,
	    week_start,
	    week_end,
	    cases_weekly,
	    cases_cumulative,
	    case_rate_weekly,
	    case_rate_cumulative,
	    tests_weekly,
	    tests_cumulative,
	    test_rate_weekly,
	    test_rate_cumulative,
	    percent_tested_positive_weekly,
	    percent_tested_positive_cumulative,
	    deaths_weekly,
	    deaths_cumulative,
	    death_rate_weekly,
	    death_rate_cumulative,
	    population,
	    row_id,
	    zip_lat,
	    zip_long,
	    computed_region_rpca_8um6,
	    computed_region_vrxf_vc4k,
	    computed_region_6mkv_f3dw,
	    computed_region_bdys_3d7i,
	    computed_region_43wa_7qmu
	  ) VALUES (
	    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27
	  )
	  ON CONFLICT (row_id) DO NOTHING;
	`
	for idx, r := range rows {
		if _, err := conn.Exec(ctx, sqlText, r...); err != nil {
			log.Printf("❌ row-by-row INSERT failed at sub-index %d: %v", idx, err)
		}
	}
}
