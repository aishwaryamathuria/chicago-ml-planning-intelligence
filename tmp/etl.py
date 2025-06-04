import pandas as pd
import geopandas as gpd
from shapely.geometry import Point, shape
import fiona
import os

os.makedirs("curated", exist_ok=True)

# --------------------------------------
# Load and sanitize ZIP code polygons
# --------------------------------------
def load_zip_geometries(path):
    features = []
    with fiona.open(path) as src:
        for feature in src:
            try:
                geom = shape(feature["geometry"])
                if geom.is_valid:
                    if geom.geom_type == "MultiPolygon":
                        for part in geom.geoms:
                            features.append({"geometry": part, **feature["properties"]})
                    else:
                        features.append({"geometry": geom, **feature["properties"]})
            except Exception as e:
                print(f"Skipping geometry due to error: {e}")
    return gpd.GeoDataFrame(features, crs="EPSG:4326").to_crs(epsg=3435)

zip_codes = load_zip_geometries("boundary_zipcode.geojson")

# --------------------------------------
# Load mapping from community area to ZIP
# --------------------------------------
community_zip_map = pd.read_csv("community_area_to_zip_mapping.csv")
area_to_zip = dict(zip(
    community_zip_map["area_numbe"].astype(int).astype(str),
    community_zip_map["zip"].astype(str)
))

# --------------------------------------
# Utility functions
# --------------------------------------
def get_zip_from_coords(lat, lon):
    """Return ZIP code from lat/lon via spatial join."""
    try:
        point = gpd.GeoDataFrame(geometry=[Point(lon, lat)], crs="EPSG:4326").to_crs(epsg=3435)
        match = gpd.sjoin(point, zip_codes, how="left", predicate="within")
        if match.empty:
            print(f"[WARN] No ZIP match for point: ({lat}, {lon})")
        return match.iloc[0]["zip"] if not match.empty else None
    except Exception as e:
        print(f"[ERROR] Failed on point ({lat}, {lon}): {e}")
        return None

def get_zip_from_area(area_numbe):
    """Return ZIP code using community area mapping."""
    try:
        return area_to_zip.get(str(int(float(area_numbe))), None)
    except:
        return None

def enrich_dataframe(df):
    """Main transformation logic to enrich with ZIP code."""
    zipcodes = []

    for _, row in df.iterrows():
        if pd.notna(row.get("zipcode")):
            zipcodes.append(row["zipcode"])
        elif pd.notna(row.get("latitude")) and pd.notna(row.get("longitude")):
            zip_from_coords = get_zip_from_coords(row["latitude"], row["longitude"])
            zipcodes.append(zip_from_coords)
        elif pd.notna(row.get("area_numbe")):
            zip_from_area = get_zip_from_area(row["area_numbe"])
            zipcodes.append(zip_from_area)
        else:
            zipcodes.append(None)

    df["zipcode"] = zipcodes
    return df

def process_dataset(input_path, output_path):
    df = pd.read_csv(input_path)
    enriched_df = enrich_dataframe(df)
    enriched_df.to_csv(output_path, index=False)
    print(f"[DONE] Processed and saved: {output_path}")

# --------------------------------------
# Process staging datasets
# --------------------------------------
datasets = {
    "staging/reviews.csv": "curated/reviews_enriched.csv",
    "staging/businesses.csv": "curated/businesses_enriched.csv",
    # Add more files here if needed
}

for input_file, output_file in datasets.items():
    process_dataset(input_file, output_file)
