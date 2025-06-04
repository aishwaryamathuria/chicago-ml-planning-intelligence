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

type Report6Result struct {
	ZipCode         string `json:"zip_code"`
	PermitNumber    string `json:"permit_number"`
	PermitStatus    string `json:"permit_status"`
	Neighborhood    string `json:"neighborhood"`
	CommunityArea   string `json:"community_area"`
	PerCapitaIncome string `json:"per_capita_income"`
}

func HandleReport6(w http.ResponseWriter, r *http.Request) {
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

	query := `WITH new_construction_counts AS (
					SELECT
						zip_code,
						COUNT(*) AS permit_count
					FROM
						public.stg_building_permits
					WHERE
						permit_type = 'PERMIT - NEW CONSTRUCTION'
					AND
						zip_code::text != '[null]'
					GROUP BY
						zip_code
				),
				zip_with_min_permits AS (
					SELECT
						zip_code
					FROM
						new_construction_counts
					ORDER BY
						permit_count ASC
					LIMIT 1
				)
				SELECT
					p.zip_code AS zip_code,
					p.permit_number AS permit_num,
					p.neighborhood AS neighborhood,
					u.community_area_name AS community_area,
					p.permit_status AS permit_status,
					u.per_capita_income AS per_capita_income
				FROM
					public.stg_building_permits p
				JOIN
					public.stg_unemployment u
					ON p.community_area = u.community_area_number
				JOIN
					zip_with_min_permits zmin
					ON p.zip_code = zmin.zip_code
				WHERE
					p.permit_type = 'PERMIT - NEW CONSTRUCTION'
					AND u.per_capita_income < 30000;
			`

	rows, err := db.Query(query)
	if err != nil {
		handleError(fmt.Errorf("failed to execute query: %w", err), "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []Report6Result
	for rows.Next() {
		var r Report6Result
		if err := rows.Scan(&r.ZipCode, &r.PermitNumber, &r.PermitStatus, &r.Neighborhood, &r.CommunityArea, &r.PerCapitaIncome); err != nil {
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
