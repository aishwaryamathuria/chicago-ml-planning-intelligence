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

type Report3CResult struct {
	ZipCode   string `json:"zip_code"`
	CaseCount string `json:"case_count"`
}

func HandleReport3C(w http.ResponseWriter, r *http.Request) {
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

	query := `With new_table AS (
				SELECT
					CASE
						WHEN t.pickup_zip_code IN (60656, 60666) THEN 'O''Hare'
						WHEN t.pickup_zip_code = 60638 THEN 'Midway'
						ELSE t.pickup_zip_code::text
					END AS pickup_location,
					CASE
						WHEN t.dropoff_zip_code IN (60656, 60666) THEN 'O''Hare'
						WHEN t.dropoff_zip_code = 60638 THEN 'Midway'
						ELSE t.dropoff_zip_code::text
					END AS dropoff_location,
					c.case_count
				FROM stg_taxi_trips t
				JOIN stg_covid_cases c
				ON (
					t.trip_start_timestamp::date BETWEEN c.start_date AND c.end_date AND
					(t.pickup_zip_code = c.zip_code OR t.dropoff_zip_code = c.zip_code)
				)
				WHERE 
				(t.pickup_zip_code IN (60656, 60666, 60638)
				OR t.dropoff_zip_code IN (60656, 60666, 60638))
				AND t.pickup_zip_code IS NOT NULL
				AND t.dropoff_zip_code IS NOT NULL
				AND t.pickup_zip_code::text != '[null]'
				AND t.dropoff_zip_code::text != '[null]'
			)
			SELECT 
				pickup_location AS zip_code,
				SUM(case_count) AS case_count
			FROM new_table
			WHERE dropoff_location = 'O''Hare'
			GROUP BY pickup_location;
			`

	rows, err := db.Query(query)
	if err != nil {
		handleError(fmt.Errorf("failed to execute query: %w", err), "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []Report3AResult
	for rows.Next() {
		var r Report3AResult
		if err := rows.Scan(&r.ZipCode, &r.CaseCount); err != nil {
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
