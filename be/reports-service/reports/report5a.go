package reports

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

type Report5AResult struct {
	Neighborhood      string `json:"neighborhood"`
	CommunityAreaName string `json:"community_area_name"`
	UnemploymentRate  string `json:"max_unemployment_rate"`
	PovertyLevel      string `json:"max_below_poverty_level"`
}

func HandleReport5A(w http.ResponseWriter, r *http.Request) {
	handleError := func(err error, message string, statusCode int) {
		log.Printf("%s: %v", message, err)
		http.Error(w, message, statusCode)
	}

	db, err := sql.Open("postgres", os.Getenv("DB_CONN"))
	if err != nil {
		handleError(fmt.Errorf("failed to open database connection: %w", err), "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	query := `WITH neighborhood_stats AS (
				SELECT 
					bp.neighborhood as neighborhood,
					ue.community_area_name as community_area_name,
					MAX(ue.unemployment_rate) AS max_unemployment_rate,
					MAX(ue.below_poverty_level) AS max_below_poverty_level
				FROM 
					public.stg_building_permits bp
				JOIN 
					public.stg_unemployment ue
				ON 
					bp.community_area = ue.community_area_number
				GROUP BY 
					bp.neighborhood, ue.community_area_name
			),
			ranked_neighborhoods AS (
				SELECT
					neighborhood,
					community_area_name,
					max_unemployment_rate,
					max_below_poverty_level,
					ROW_NUMBER() OVER (
						ORDER BY max_unemployment_rate DESC, max_below_poverty_level DESC
					) AS rn
				FROM neighborhood_stats
			)
			SELECT 
				neighborhood,
				community_area_name,
				max_unemployment_rate,
				max_below_poverty_level
			FROM 
				ranked_neighborhoods
			WHERE 
				rn <= 5;
			`

	rows, err := db.Query(query)
	if err != nil {
		handleError(fmt.Errorf("failed to execute query: %w", err), "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []Report5AResult
	for rows.Next() {
		var r Report5AResult
		if err := rows.Scan(&r.Neighborhood, &r.CommunityAreaName, &r.UnemploymentRate, &r.PovertyLevel); err != nil {
			handleError(fmt.Errorf("failed to scan row: %w", err), "Internal Server Error", http.StatusInternalServerError)
			return
		}
		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		handleError(fmt.Errorf("error during row iteration: %w", err), "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
