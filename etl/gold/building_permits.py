import psycopg2
from datetime import datetime, timedelta
import geopandas as gpd
from shapely import wkt
import pandas as pd
from sqlalchemy import create_engine

SILVER_DATABASE = {
    'dbname': 'ChicagoBusinessIntelligence_SILVER',
    'user': 'postgres',
    'password': 'root',
    'host': 'localhost',
    'port': 5432
}

GOLD_DATABASE = {
    'dbname': 'ChicagoBusinessIntelligence_GOLD',
    'user': 'postgres',
    'password': 'root',
    'host': 'localhost',
    'port': 5432
}

TABLE_NAME = "building_permits"

comma_gdf = gpd.read_file('./geo_jsons/community_areas.geojson')
if comma_gdf.crs != 'EPSG:4326':
    comma_gdf = comma_gdf.to_crs('EPSG:4326')

neigh_gdf = gpd.read_file('./geo_jsons/neighborhoods.geojson')
if neigh_gdf.crs != 'EPSG:4326':
    neigh_gdf = neigh_gdf.to_crs('EPSG:4326')

community_areas = gpd.read_file("./geo_jsons/community_areas.geojson")
zip_codes = gpd.read_file("./geo_jsons/boundary_zipcode.geojson")
community_areas = community_areas.to_crs(epsg=3435)
zip_codes = zip_codes.to_crs(epsg=3435)
intersections = gpd.overlay(community_areas, zip_codes, how='intersection')
intersections["intersection_area"] = intersections.geometry.area
dominant_zip_df = (
    intersections.loc[:, ["AREA_NUMBE", "zip", "intersection_area"]]
    .sort_values(by="intersection_area", ascending=False)
    .drop_duplicates(subset="AREA_NUMBE"))
dominant_zip_map = dict(zip(dominant_zip_df["AREA_NUMBE"], dominant_zip_df["zip"]))

def fetch_data(cur, table_name):
    time_diff = datetime.now() - timedelta(hours=12)
    query = f"SELECT * FROM {table_name} WHERE last_updated >= '{time_diff}'"
    cur.execute(query)
    colnames = [desc[0] for desc in cur.description]
    return pd.DataFrame(cur.fetchall(), columns=colnames)   

def enrich_dataframe(df):
    df["zip_code"] = 0
    df["neighborhood"] = "Unknown"
    new_rows = []
    for _, row in df.iterrows():
        row_dict = row.to_dict()
        loc = row_dict.get("location")
        if loc:
            loc = loc.replace(',', ' ')
            point_geom = wkt.loads(loc)
            ca_match = comma_gdf[comma_gdf.contains(point_geom)]
            if not ca_match.empty:
                ca = ca_match.iloc[0]["AREA_NUMBE"]
                row_dict["community_area"] = ca
                row_dict["zip_code"] = dominant_zip_map.get(ca, 0)
            neigh_match = neigh_gdf[neigh_gdf.contains(point_geom)]
            if not neigh_match.empty:
                row_dict["neighborhood"] = neigh_match.iloc[0]["PRI_NEIGH"]
        new_rows.append(row_dict)

    return pd.DataFrame(new_rows)

if __name__ == "__main__":
    try:
        conn1 = psycopg2.connect(**SILVER_DATABASE)
        cur1 = conn1.cursor()
        df = fetch_data(cur1, TABLE_NAME)
        enriched_df = enrich_dataframe(df)
        enriched_df.drop(columns=['last_updated'], inplace=True)
        enriched_df = enriched_df.convert_dtypes()
        engine = create_engine(
            f"postgresql://{GOLD_DATABASE['user']}:{GOLD_DATABASE['password']}"
            f"@{GOLD_DATABASE['host']}:{GOLD_DATABASE['port']}/{GOLD_DATABASE['dbname']}"
        )
        enriched_df.to_sql(TABLE_NAME, engine, if_exists='append', index=False)

    except Exception as e:
        print("Error occurred:", e)

    finally:
        cur1.close()
        conn1.close()
