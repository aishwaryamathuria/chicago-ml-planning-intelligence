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

type Report4Result struct {
	TripID              string `json:"trip_id"`
	StartTime           string `json:"trip_start_timestamp"`
	EndTime             string `json:"trip_end_timestamp"`
	PickupZipCode       string `json:"pickup_zip_code"`
	DropoffZipCode      string `json:"dropoff_zip_code"`
	PickupNeighborhood  string `json:"pickup_neighborhood"`
	DropoffNeighborhood string `json:"dropoff_neighborhood"`
	CCVICategory        string `json:"ccvi_category"`
}

func HandleReport4(w http.ResponseWriter, r *http.Request) {
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
				t.trip_id,
				TO_CHAR(t.trip_start_timestamp, 'DD/MM/YY HH24:MI:SS') AS trip_start_timestamp,
				TO_CHAR(t.trip_end_timestamp, 'DD/MM/YY HH24:MI:SS') AS trip_end_timestamp,
				pickup_zip_code,
				dropoff_zip_code,
				pickup_neighborhood,
				dropoff_neighborhood,
				'HIGH' AS ccvi_category
			FROM public.stg_taxi_trips t
			LEFT JOIN public.stg_covid_ccvi p
				ON t.pickup_zip_code = p.zip_code AND p.ccvi_category = 'HIGH'
			LEFT JOIN public.stg_covid_ccvi d
				ON t.dropoff_zip_code = d.zip_code AND d.ccvi_category = 'HIGH'
			WHERE (p.ccvi_category = 'HIGH' OR d.ccvi_category = 'HIGH')
			AND t.pickup_zip_code IS NOT NULL
			AND t.dropoff_zip_code IS NOT NULL
			AND t.pickup_zip_code::text != '[null]'
			AND t.dropoff_zip_code::text != '[null]'
			;
			`

	rows, err := db.Query(query)
	if err != nil {
		handleError(fmt.Errorf("failed to execute query: %w", err), "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []Report4Result
	for rows.Next() {
		var r Report4Result
		if err := rows.Scan(&r.TripID, &r.StartTime, &r.EndTime, &r.PickupZipCode, &r.DropoffZipCode, &r.PickupNeighborhood, &r.DropoffNeighborhood, &r.CCVICategory); err != nil {
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
