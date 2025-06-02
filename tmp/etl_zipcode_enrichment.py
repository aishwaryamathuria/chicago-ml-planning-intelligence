# etl_zipcode_enrichment.py

import pandas as pd
import geopandas as gpd
from shapely.geometry import Point
from sqlalchemy import create_engine
from mapping import get_zip_for_community_area

# --- Config ---
ZIP_GEOJSON = "boundary_zipcode.geojson"
COMMUNITY_GEOJSON = "community_areas.geojson"
DB_URI = "postgresql://user:password@localhost:5432/production_database"

# --- Load GeoDataFrames ---
zip_gdf = gpd.read_file(ZIP_GEOJSON)
community_gdf = gpd.read_file(COMMUNITY_GEOJSON)

# Ensure CRS matches
zip_gdf = zip_gdf.to_crs(epsg=4326)
community_gdf = community_gdf.to_crs(epsg=4326)

# --- Helper Functions ---
def get_zip_from_coords(lat, lon):
    if pd.isna(lat) or pd.isna(lon):
        return None
    point = Point(lon, lat)
    for _, row in zip_gdf.iterrows():
        if row['geometry'].contains(point):
            return row.get('zip', row.get('ZIP', row.get('zip_code')))
    return None

def enrich_zipcode(row):
    if pd.notna(row.get('zipcode')):
        return row['zipcode']
    elif pd.notna(row.get('latitude')) and pd.notna(row.get('longitude')):
        return get_zip_from_coords(row['latitude'], row['longitude'])
    elif pd.notna(row.get('community_area')):
        return get_zip_for_community_area(row['community_area'])
    return None

# --- Dataset Processor ---
def process_and_load(df, table_name, engine):
    print(f"Processing table: {table_name}")
    df['zipcode'] = df.apply(enrich_zipcode, axis=1)
    df.to_sql(table_name, engine, if_exists='append', index=False)
    print(f"Loaded {len(df)} records into {table_name}")

# --- Main Driver ---
def main():
    engine = create_engine(DB_URI)

    # List of staging datasets to enrich (example file paths)
    datasets = [
        ("data/stg_taxi_trips.csv", "prd_taxi_trips"),
        ("data/stg_building_permits.csv", "prd_building_permits"),
        ("data/stg_covid_ccvi.csv", "prd_covid_data"),
        # Add more datasets as needed
    ]

    for csv_path, table_name in datasets:
        df = pd.read_csv(csv_path)
        process_and_load(df, table_name, engine)

if __name__ == "__main__":
    main()
