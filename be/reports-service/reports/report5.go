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

type Report5Result struct {
	PermitNumber        string `json:"permit_number"`
	PermitType          string `json:"permit_type"`
	CommunityAreaNumber string `json:"community_area"`
	CommunityAreaName   string `json:"community_area_name"`
	Neighborhood        string `json:"neighborhood"`
	UnemploymentRate    string `json:"unemployment_rate"`
	PovertyLevel        string `json:"below_poverty_level"`
}

func HandleReport5(w http.ResponseWriter, r *http.Request) {
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

	query := `SELECT 
				bp.permit_number,
				bp.permit_type,
				bp.community_area,
				ue.community_area_name,
				bp.neighborhood,
				ue.unemployment_rate,
				ue.below_poverty_level
			FROM 
				public.stg_building_permits bp
			JOIN 
				public.stg_unemployment ue
			ON 
				bp.community_area = ue.community_area_number
			ORDER BY ue.unemployment_rate DESC, ue.below_poverty_level DESC
			LIMIT 1000;
			`

	rows, err := db.Query(query)
	if err != nil {
		handleError(fmt.Errorf("failed to execute query: %w", err), "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []Report5Result
	for rows.Next() {
		var r Report5Result
		if err := rows.Scan(&r.PermitNumber, &r.PermitType, &r.CommunityAreaNumber, &r.CommunityAreaName, &r.Neighborhood, &r.UnemploymentRate, &r.PovertyLevel); err != nil {
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
