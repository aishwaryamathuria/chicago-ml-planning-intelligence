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

type Report4BResult struct {
	DropoffNeighborhood string `json:"dropoff_neighborhood"`
	TripCount           string `json:"trip_count"`
}

func HandleReport4B(w http.ResponseWriter, r *http.Request) {
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
				dropoff_neighborhood,
				COUNT(*) AS trip_count
			FROM public.stg_taxi_trips t
			LEFT JOIN public.stg_covid_ccvi p
				ON t.pickup_zip_code = p.zip_code AND p.ccvi_category = 'HIGH'
			LEFT JOIN public.stg_covid_ccvi d
				ON t.dropoff_zip_code = d.zip_code AND d.ccvi_category = 'HIGH'
			WHERE (p.ccvi_category = 'HIGH'OR d.ccvi_category = 'HIGH')
				AND dropoff_neighborhood IS NOT NULL
				AND dropoff_neighborhood::text != '[null]'
			GROUP BY dropoff_neighborhood;
			`

	rows, err := db.Query(query)
	if err != nil {
		handleError(fmt.Errorf("failed to execute query: %w", err), "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []Report4BResult
	for rows.Next() {
		var r Report4BResult
		if err := rows.Scan(&r.DropoffNeighborhood, &r.TripCount); err != nil {
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
