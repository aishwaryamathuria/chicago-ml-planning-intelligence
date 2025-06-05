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

// PublicHealth matches Socrata’s /iqnk-2tcu.json.  All “numeric” fields are plain strings here.
type PublicHealth struct {
	CommunityArea                         string `json:"community_area"`
	CommunityAreaName                     string `json:"community_area_name"`
	BirthRate                             string `json:"birth_rate"`
	GeneralFertilityRate                  string `json:"general_fertility_rate"`
	LowBirthWeight                        string `json:"low_birth_weight"`
	PrenatalCareBeginningInFirstTrimester string `json:"prenatal_care_beginning_in_first_trimester"`
	PretermBirths                         string `json:"preterm_births"`
	TeenBirthRate                         string `json:"teen_birth_rate"`
	AssaultHomicide                       string `json:"assault_homicide"`
	BreastCancerInFemales                 string `json:"breast_cancer_in_females"`
	CancerAllSites                        string `json:"cancer_all_sites"`
	ColorectalCancer                      string `json:"colorectal_cancer"`
	DiabetesRelated                       string `json:"diabetes_related"`
	FirearmRelated                        string `json:"firearm_related"`
	InfantMortalityRate                   string `json:"infant_mortality_rate"`
	LungCancer                            string `json:"lung_cancer"`
	ProstateCancerInMales                 string `json:"prostate_cancer_in_males"`
	StrokeCerebrovascularDisease          string `json:"stroke_cerebrovascular_disease"`
	ChildhoodBloodLeadLevelScreening      string `json:"childhood_blood_lead_level_screening"`
	ChildhoodLeadPoisoning                string `json:"childhood_lead_poisoning"`
	GonorrheaInFemales                    string `json:"gonorrhea_in_females"`
	GonorrheaInMales                      string `json:"gonorrhea_in_males"`
	Tuberculosis                          string `json:"tuberculosis"`
	BelowPovertyLevel                     string `json:"below_poverty_level"`
	CrowdedHousing                        string `json:"crowded_housing"`
	Dependency                            string `json:"dependency"`
	NoHighSchoolDiploma                   string `json:"no_high_school_diploma"`
	PerCapitaIncome                       string `json:"per_capita_income"`
	Unemployment                          string `json:"unemployment"`
}

// parseIntField returns (value, true) if s is non-empty, not ".", and parses as an integer.
// Otherwise it returns (0, false) so caller will insert NULL.
func parseIntField(s string) (int64, bool) {
	if s == "" || s == "." {
		return 0, false
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return i, true
}

// parseFloatField returns (value, true) if s is non-empty, not ".", and parses as a float.
// Otherwise it returns (0.0, false) so caller will insert NULL.
func parseFloatField(s string) (float64, bool) {
	if s == "" || s == "." {
		return 0.0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0, false
	}
	return f, true
}

// LoadPublicHealthConcurrent fetches pages of /iqnk-2tcu.json and COPYs into public_health_stats.
// On HTTP 429, it retries a few times; on a duplicate-key (unlikely), it falls back to per-row INSERT.
func LoadPublicHealthConcurrent() {
	fmt.Println("▶ Fetching Public Health Statistics → COPY INTO public_health_stats…")

	apiBase := "https://data.cityofchicago.org/resource/iqnk-2tcu.json"
	appToken := os.Getenv("APP_TOKEN")

	const (
		limit       = 1000 // Socrata’s page size
		maxOffset   = 2000 // There are ~77 records; two pages will suffice
		concurrency = 1    // Only a handful of rows—one worker is enough
	)

	offsetChan := make(chan int, concurrency)
	var wg sync.WaitGroup

	// Spawn 'concurrency' workers (just 1 is fine here)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for offset := range offsetChan {
				fetchAndCopyBatchPHS(apiBase, appToken, limit, offset)
			}
		}()
	}

	// Enqueue offsets: 0, 1000
	for offset := 0; offset < maxOffset; offset += limit {
		offsetChan <- offset
	}
	close(offsetChan)

	wg.Wait()
	fmt.Println("✅ public_health_stats ingestion complete.")
}

func fetchAndCopyBatchPHS(
	apiBase, appToken string,
	limit, offset int,
) {
	// 1) Build Socrata URL
	url := fmt.Sprintf("%s?$limit=%d&$offset=%d", apiBase, limit, offset)

	// 2) HTTP GET with up to 3 retries on 429
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
		break
	}
	if resp == nil || resp.StatusCode != http.StatusOK {
		log.Printf("❌ Giving up on offset %d after retries", offset)
		return
	}
	defer resp.Body.Close()

	// 3) Decode JSON into []PublicHealth
	var records []PublicHealth
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&records); err != nil {
		log.Printf("❌ JSON decode error at offset %d: %v", offset, err)
		return
	}
	if len(records) == 0 {
		log.Printf("✅ No records at offset %d", offset)
		return
	}

	// 4) Open a fresh pgx.Conn
	ctx := context.Background()
	conn := db.GetPgxConn() // returns *pgx.Conn
	defer conn.Close(ctx)

	// 5) Build [][]interface{} for CopyFromRows
	rows := make([][]interface{}, 0, len(records))
	for _, r := range records {
		// community_area is required
		caInt, caOK := parseIntField(r.CommunityArea)
		if !caOK {
			// skip if community_area is invalid
			continue
		}

		// Parse every numeric field; if invalid or ".", we get (0, false) and insert NULL
		br, brOK := parseFloatField(r.BirthRate)
		gfr, gfrOK := parseFloatField(r.GeneralFertilityRate)
		lbw, lbwOK := parseFloatField(r.LowBirthWeight)
		pc1, pc1OK := parseFloatField(r.PrenatalCareBeginningInFirstTrimester)
		pt, ptOK := parseFloatField(r.PretermBirths)
		tb, tbOK := parseFloatField(r.TeenBirthRate)
		ah, ahOK := parseFloatField(r.AssaultHomicide)
		bcf, bcfOK := parseFloatField(r.BreastCancerInFemales)
		cas, casOK := parseFloatField(r.CancerAllSites)
		crc, crcOK := parseFloatField(r.ColorectalCancer)
		dr, drOK := parseFloatField(r.DiabetesRelated)
		fr, frOK := parseFloatField(r.FirearmRelated)
		imr, imrOK := parseFloatField(r.InfantMortalityRate)
		lc, lcOK := parseFloatField(r.LungCancer)
		pcm, pcmOK := parseFloatField(r.ProstateCancerInMales)
		scd, scdOK := parseFloatField(r.StrokeCerebrovascularDisease)
		cblls, cbllsOK := parseFloatField(r.ChildhoodBloodLeadLevelScreening)
		clp, clpOK := parseFloatField(r.ChildhoodLeadPoisoning)
		gf, gfOK := parseFloatField(r.GonorrheaInFemales)
		gm, gmOK := parseFloatField(r.GonorrheaInMales)
		tb2, tb2OK := parseFloatField(r.Tuberculosis)
		bpl, bplOK := parseFloatField(r.BelowPovertyLevel)
		ch, chOK := parseFloatField(r.CrowdedHousing)
		dp, dpOK := parseFloatField(r.Dependency)
		nhs, nhsOK := parseFloatField(r.NoHighSchoolDiploma)
		// per_capita_income is an integer
		pci, pciOK := parseIntField(r.PerCapitaIncome)
		// unemployment is a float
		unemp, unempOK := parseFloatField(r.Unemployment)

		// Build a single row: must match CREATE TABLE order exactly
		row := make([]interface{}, 29)
		row[0] = caInt               // community_area
		row[1] = r.CommunityAreaName // community_area_name
		if brOK {
			row[2] = br
		} else {
			row[2] = nil
		}
		if gfrOK {
			row[3] = gfr
		} else {
			row[3] = nil
		}
		if lbwOK {
			row[4] = lbw
		} else {
			row[4] = nil
		}
		if pc1OK {
			row[5] = pc1
		} else {
			row[5] = nil
		}
		if ptOK {
			row[6] = pt
		} else {
			row[6] = nil
		}
		if tbOK {
			row[7] = tb
		} else {
			row[7] = nil
		}
		if ahOK {
			row[8] = ah
		} else {
			row[8] = nil
		}
		if bcfOK {
			row[9] = bcf
		} else {
			row[9] = nil
		}
		if casOK {
			row[10] = cas
		} else {
			row[10] = nil
		}
		if crcOK {
			row[11] = crc
		} else {
			row[11] = nil
		}
		if drOK {
			row[12] = dr
		} else {
			row[12] = nil
		}
		if frOK {
			row[13] = fr
		} else {
			row[13] = nil
		}
		if imrOK {
			row[14] = imr
		} else {
			row[14] = nil
		}
		if lcOK {
			row[15] = lc
		} else {
			row[15] = nil
		}
		if pcmOK {
			row[16] = pcm
		} else {
			row[16] = nil
		}
		if scdOK {
			row[17] = scd
		} else {
			row[17] = nil
		}
		if cbllsOK {
			row[18] = cblls
		} else {
			row[18] = nil
		}
		if clpOK {
			row[19] = clp
		} else {
			row[19] = nil
		}
		if gfOK {
			row[20] = gf
		} else {
			row[20] = nil
		}
		if gmOK {
			row[21] = gm
		} else {
			row[21] = nil
		}
		if tb2OK {
			row[22] = tb2
		} else {
			row[22] = nil
		}
		if bplOK {
			row[23] = bpl
		} else {
			row[23] = nil
		}
		if chOK {
			row[24] = ch
		} else {
			row[24] = nil
		}
		if dpOK {
			row[25] = dp
		} else {
			row[25] = nil
		}
		if nhsOK {
			row[26] = nhs
		} else {
			row[26] = nil
		}
		if pciOK {
			row[27] = pci
		} else {
			row[27] = nil
		}
		if unempOK {
			row[28] = unemp
		} else {
			row[28] = nil
		}

		rows = append(rows, row)
	}

	// 6) Column list must match CREATE TABLE exactly (no geometry columns here).
	columns := []string{
		"community_area",
		"community_area_name",
		"birth_rate",
		"general_fertility_rate",
		"low_birth_weight",
		"prenatal_care_beginning_in_first_trimester",
		"preterm_births",
		"teen_birth_rate",
		"assault_homicide",
		"breast_cancer_in_females",
		"cancer_all_sites",
		"colorectal_cancer",
		"diabetes_related",
		"firearm_related",
		"infant_mortality_rate",
		"lung_cancer",
		"prostate_cancer_in_males",
		"stroke_cerebrovascular_disease",
		"childhood_blood_lead_level_screening",
		"childhood_lead_poisoning",
		"gonorrhea_in_females",
		"gonorrhea_in_males",
		"tuberculosis",
		"below_poverty_level",
		"crowded_housing",
		"dependency",
		"no_high_school_diploma",
		"per_capita_income",
		"unemployment",
	}

	// 7) Do the COPY FROM … public_health_stats
	copiedRows, err := conn.CopyFrom(
		ctx,
		pgx.Identifier{"public_health_stats"},
		columns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		// On duplicate-key (23505), fall back to per-row INSERT
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			log.Printf("⚠️ Duplicate-key at offset %d, retrying rows individually…", offset)
			insertRowsOneByOnePHS(ctx, conn, rows)
			return
		}
		log.Printf("❌ COPY failed at offset %d: %v", offset, err)
		return
	}

	fmt.Printf("✅ Bulk-COPIED %d rows into public_health_stats at offset %d\n", copiedRows, offset)
}

// insertRowsOneByOnePHS performs INSERT … ON CONFLICT DO NOTHING row by row.
// Only invoked if the bulk COPY encountered a 23505 duplicate-key error.
func insertRowsOneByOnePHS(ctx context.Context, conn *pgx.Conn, rows [][]interface{}) {
	const sqlText = `
	  INSERT INTO public_health_stats (
	    community_area,
	    community_area_name,
	    birth_rate,
	    general_fertility_rate,
	    low_birth_weight,
	    prenatal_care_beginning_in_first_trimester,
	    preterm_births,
	    teen_birth_rate,
	    assault_homicide,
	    breast_cancer_in_females,
	    cancer_all_sites,
	    colorectal_cancer,
	    diabetes_related,
	    firearm_related,
	    infant_mortality_rate,
	    lung_cancer,
	    prostate_cancer_in_males,
	    stroke_cerebrovascular_disease,
	    childhood_blood_lead_level_screening,
	    childhood_lead_poisoning,
	    gonorrhea_in_females,
	    gonorrhea_in_males,
	    tuberculosis,
	    below_poverty_level,
	    crowded_housing,
	    dependency,
	    no_high_school_diploma,
	    per_capita_income,
	    unemployment
	  ) VALUES (
	    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29
	  )
	  ON CONFLICT (community_area) DO NOTHING;
	`

	for idx, r := range rows {
		if _, err := conn.Exec(ctx, sqlText, r...); err != nil {
			log.Printf("❌ row-by-row INSERT failed at sub-index %d: %v", idx, err)
		}
	}
}
