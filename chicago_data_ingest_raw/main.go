package main

import (
	"chicago_data_ingest/fetcher"
	"fmt"
)

func main() {
	fmt.Println("🚀 Starting Chicago Data Ingestion Pipeline")

	// fetcher.LoadCCVI()
	// fetcher.LoadTaxiConcurrent()
	// fetcher.LoadTNPConcurrent()
	// fetcher.LoadBuildingPermitsConcurrent()
	// fetcher.LoadPublicHealthConcurrent()
	// fetcher.LoadCovidZipConcurrent()
	fetcher.LoadZipBoundariesConcurrent()

	fmt.Println("✅ All datasets ingested.")
}
