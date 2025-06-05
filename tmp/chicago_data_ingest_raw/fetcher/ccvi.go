package fetcher

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"chicago_data_ingest/db"
)

type CCVIRecord struct {
	CommunityAreaOrZip              string  `json:"community_area_or_zip"`
	CommunityAreaName               string  `json:"community_area_name"`
	CCVIScore                       float64 `json:"ccvi_score,string"`
	CCVICategory                    string  `json:"ccvi_category"`
	Rank                            int     `json:"rank,string"`
	Population                      int     `json:"population,string"`
	PercentUnemployed               float64 `json:"percent_unemployed,string"`
	PercentBelowPoverty             float64 `json:"percent_below_poverty,string"`
	PercentWithoutHighSchoolDiploma float64 `json:"percent_without_high_school_diploma,string"`
	PercentWithDisability           float64 `json:"percent_with_disability,string"`
	PercentWithoutHealthInsurance   float64 `json:"percent_without_health_insurance,string"`
	PercentHouseholdsWithCrowding   float64 `json:"percent_households_with_crowding,string"`
	PercentHouseholdsWithNoInternet float64 `json:"percent_households_with_no_internet,string"`
	PercentHouseholdsWithNoVehicle  float64 `json:"percent_households_with_no_vehicle,string"`
	PercentNonWhite                 float64 `json:"percent_non_white,string"`
	PercentLimitedEnglish           float64 `json:"percent_limited_english,string"`
	PercentEssentialWorkers         float64 `json:"percent_essential_workers,string"`
	CovidCasesPer100k               float64 `json:"covid_cases_per_100k,string"`
	CovidDeathsPer100k              float64 `json:"covid_deaths_per_100k,string"`
	CovidTestsPer100k               float64 `json:"covid_tests_per_100k,string"`
	CovidPositivityRate             float64 `json:"covid_positivity_rate,string"`
}

func LoadCCVI() {
	fmt.Println("▶ Fetching CCVI data...")
	resp, err := http.Get("https://data.cityofchicago.org/resource/xhc6-88s9.json")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var records []CCVIRecord
	if err := json.Unmarshal(body, &records); err != nil {
		log.Fatal(err)
	}

	conn := db.GetDB()
	for _, r := range records {
		_, err := conn.Exec(`
			INSERT INTO ccvi (
				community_area_or_zip, community_area_name, ccvi_score, ccvi_category, rank, population,
				percent_unemployed, percent_below_poverty, percent_without_high_school_diploma,
				percent_with_disability, percent_without_health_insurance, percent_households_with_crowding,
				percent_households_with_no_internet, percent_households_with_no_vehicle, percent_non_white,
				percent_limited_english, percent_essential_workers, covid_cases_per_100k, covid_deaths_per_100k,
				covid_tests_per_100k, covid_positivity_rate
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
		`, r.CommunityAreaOrZip, r.CommunityAreaName, r.CCVIScore, r.CCVICategory, r.Rank, r.Population,
			r.PercentUnemployed, r.PercentBelowPoverty, r.PercentWithoutHighSchoolDiploma,
			r.PercentWithDisability, r.PercentWithoutHealthInsurance, r.PercentHouseholdsWithCrowding,
			r.PercentHouseholdsWithNoInternet, r.PercentHouseholdsWithNoVehicle, r.PercentNonWhite,
			r.PercentLimitedEnglish, r.PercentEssentialWorkers, r.CovidCasesPer100k, r.CovidDeathsPer100k,
			r.CovidTestsPer100k, r.CovidPositivityRate)
		if err != nil {
			log.Println("❌ Error inserting row:", err)
		}
	}
	fmt.Println("✅ CCVI data loaded successfully.")
}
