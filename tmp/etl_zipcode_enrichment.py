import pandas as pd
import geopandas as gpd
from shapely.geometry import Point
from sqlalchemy import create_engine, text
from tmp.mapping import get_zip_for_community_area

# --- Configuration ---
ZIP_GEOJSON = "boundary_zipcode.geojson"
COMMUNITY_GEOJSON = "community_areas.geojson"
DB_URI = "postgresql://user:password@localhost:5432/chicago_data"
SCHEMA = "curated"

# --- Load GeoDataFrames ---
zip_gdf = gpd.read_file(ZIP_GEOJSON).to_crs(epsg=4326)
community_gdf = gpd.read_file(COMMUNITY_GEOJSON).to_crs(epsg=4326)

# Normalize zip_gdf columns
zip_gdf.columns = [col.lower() for col in zip_gdf.columns]
if "zip_code" in zip_gdf.columns:
    zip_gdf.rename(columns={"zip_code": "zipcode"}, inplace=True)
elif "zip" in zip_gdf.columns:
    zip_gdf.rename(columns={"zip": "zipcode"}, inplace=True)

# --- Helper Functions ---
def get_zip_from_coords(lat, lon):
    if pd.isna(lat) or pd.isna(lon):
        return None
    point = gpd.GeoDataFrame(geometry=[Point(lon, lat)], crs="EPSG:4326")
    try:
        joined = gpd.sjoin(point, zip_gdf, how="left", predicate="within")
        if not joined.empty:
            return joined.iloc[0].get("zipcode")
    except Exception as e:
        print(f"Error during spatial join: {e}")
    return None

def enrich_zipcode(row):
    if pd.notna(row.get("zipcode")):
        return row["zipcode"], "direct"
    elif pd.notna(row.get("latitude")) and pd.notna(row.get("longitude")):
        zip_code = get_zip_from_coords(row["latitude"], row["longitude"])
        return zip_code, "coordinates" if zip_code else "unknown"
    elif pd.notna(row.get("community_area")):
        zip_code = get_zip_for_community_area(row["community_area"])
        return zip_code, "community_area" if zip_code else "unknown"
    return None, "unknown"

def process_and_load_table(engine, staging_table, curated_table):
    print(f"\nProcessing {staging_table} → {curated_table}")
    try:
        df = pd.read_sql(f"SELECT * FROM {staging_table}", engine)
        df[["zipcode", "zipcode_source"]] = df.apply(
            lambda row: pd.Series(enrich_zipcode(row)), axis=1
        )
        with engine.begin() as conn:
            conn.execute(text(f"DROP TABLE IF EXISTS {SCHEMA}.{curated_table}"))
            df.to_sql(curated_table, con=conn, schema=SCHEMA, if_exists="replace", index=False)
        print(f"✅ Inserted {len(df)} rows into {SCHEMA}.{curated_table}")
    except Exception as e:
        print(f"❌ Failed to process {staging_table}: {e}")

# --- Main ---
def main():
    engine = create_engine(DB_URI)

    tables = [
        ("stg_taxi_trips", "taxi_trips"),
        ("stg_building_permits", "building_permits"),
        ("stg_covid_ccvi", "covid_ccvi")
        # Add more (staging, curated) pairs as needed
    ]

    for staging, curated in tables:
        process_and_load_table(engine, staging, curated)

if __name__ == "__main__":
    main()
