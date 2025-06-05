import psycopg2
from datetime import datetime, timedelta
import pandas as pd
from sqlalchemy import create_engine
from shapely import wkt
import geopandas as gpd

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

TABLE_NAME = "taxi_trips"

zip_gdf = gpd.read_file('./geo_jsons/boundary_zipcode.geojson')
if zip_gdf.crs != 'EPSG:4326':
    zip_gdf = zip_gdf.to_crs('EPSG:4326')


neigh_gdf = gpd.read_file('./geo_jsons/neighborhoods.geojson')
if neigh_gdf.crs != 'EPSG:4326':
    neigh_gdf = neigh_gdf.to_crs('EPSG:4326')

community_areas = gpd.read_file("./geo_jsons/community_areas.geojson").to_crs(epsg=3435)
zip_codes = gpd.read_file("./geo_jsons/boundary_zipcode.geojson").to_crs(epsg=3435)
intersections = gpd.overlay(community_areas, zip_codes, how='intersection')
intersections["intersection_area"] = intersections.geometry.area
dominant_zip_map = dict(
    zip(
        intersections.sort_values(by="intersection_area", ascending=False)
        .drop_duplicates(subset="AREA_NUMBE")["AREA_NUMBE"],
        intersections.sort_values(by="intersection_area", ascending=False)
        .drop_duplicates(subset="AREA_NUMBE")["zip"]
    )
)

neighborhood_areas = gpd.read_file("./geo_jsons/neighborhoods.geojson").to_crs(epsg=3435)
intersections_neigh = gpd.overlay(neighborhood_areas, zip_codes, how='intersection')
intersections_neigh["intersection_area"] = intersections_neigh.geometry.area
dominant_neigh_zip_df = dict(
    zip(
        intersections_neigh.sort_values(by="intersection_area", ascending=False)
        .drop_duplicates(subset="PRI_NEIGH")["PRI_NEIGH"],
        intersections_neigh.sort_values(by="intersection_area", ascending=False)
        .drop_duplicates(subset="PRI_NEIGH")["zip"]
    )
)

def enrich_dataframe(df):
    enriched_rows = []
    for _, row in df.iterrows():
        record = row.to_dict()
        pickup_loc = row.get('pickup_centroid_location')
        if pickup_loc:
            point_geom = wkt.loads(record['pickup_centroid_location'].replace(',', ' '))
            matching_zip = zip_gdf[zip_gdf.contains(point_geom)]
            record["pickup_zip_code"] = int(matching_zip.iloc[0]['zip'])
            matching_neigh = neigh_gdf[neigh_gdf.contains(point_geom)]
            record["pickup_neighborhood"] = matching_neigh.iloc[0]['PRI_NEIGH']

        dropoff_loc = row.get('dropoff_centroid_location')
        if dropoff_loc:
            point_geom = wkt.loads(record['dropoff_centroid_location'].replace(',', ' '))
            matching_zip = zip_gdf[zip_gdf.contains(point_geom)]
            record["dropoff_zip_code"] = int(matching_zip.iloc[0]['zip'])
            matching_neigh = neigh_gdf[neigh_gdf.contains(point_geom)]
            record["dropoff_neighborhood"] = matching_neigh.iloc[0]['PRI_NEIGH']

        if not record["pickup_zip_code"] and row.get('pickup_community_area') and dominant_zip_map.get(row.get('pickup_community_area')):
            record["pickup_zip_code"] = dominant_zip_map.get(row.get('pickup_community_area'))

        if not record["dropoff_zip_code"] and row.get('dropoff_community_area') and dominant_zip_map.get(row.get('dropoff_community_area')):
            record["dropoff_zip_code"] = dominant_zip_map.get(row.get('dropoff_community_area'))

        if (not record["pickup_zip_code"]) and (not record["dropoff_zip_code"]):
            continue
        
        enriched_rows.append(record)

    return pd.DataFrame(enriched_rows)

def fetch_data(cur, table_name):
    time_diff = datetime.now() - timedelta(hours=12)
    query = f"SELECT * FROM {table_name} WHERE last_updated >= '{time_diff}'"
    cur.execute(query)
    colnames = [desc[0] for desc in cur.description]
    return pd.DataFrame(cur.fetchall(), columns=colnames)   

if __name__ == "__main__":
    try:
        conn = psycopg2.connect(**SILVER_DATABASE)
        cur = conn.cursor()
        df = fetch_data(cur, TABLE_NAME)

        df['pickup_zip_code'] = 0
        df['dropoff_zip_code'] = 0
        df['pickup_neighborhood'] = "Unknown"
        df['dropoff_neighborhood'] = "Unknown"

        enriched_df = enrich_dataframe(df)

        engine = create_engine(
            f"postgresql://{GOLD_DATABASE['user']}:{GOLD_DATABASE['password']}"
            f"@{GOLD_DATABASE['host']}:{GOLD_DATABASE['port']}/{GOLD_DATABASE['dbname']}"
        )

        enriched_df.to_sql(TABLE_NAME, engine, if_exists='append', index=False)
        print("Data enriched and inserted into GOLD.")

    except Exception as e:
        print("Error occurred:", e)

    finally:
        if conn:
            conn.close()
