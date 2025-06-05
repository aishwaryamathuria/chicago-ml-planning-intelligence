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

TABLE_NAME = "ccvi_index"

zip_gdf = gpd.read_file('./geo_jsons/boundary_zipcode.geojson')
if zip_gdf.crs != 'EPSG:4326':
    zip_gdf = zip_gdf.to_crs('EPSG:4326')

def process_zip(row):
    if row['geography_type'] == 'CA':
        wkt_point_str = row['location']
        wkt_point_str = wkt_point_str.replace(',', ' ')
        point_geom = wkt.loads(wkt_point_str)
        matching_zip = zip_gdf[zip_gdf.contains(point_geom)]
        return int(matching_zip.iloc[0]['zip'])
    else:
        return int(row['community_area_or_zip'])

def fetch_data(cur, table_name):
    time_diff = datetime.now() - timedelta(hours=12)
    query = f"SELECT * FROM {table_name} WHERE last_updated >= '{time_diff}'"
    cur.execute(query)
    colnames = [desc[0] for desc in cur.description]
    return pd.DataFrame(cur.fetchall(), columns=colnames)   

if __name__ == "__main__":
    try:
        conn1 = psycopg2.connect(**SILVER_DATABASE)
        cur1 = conn1.cursor()
        df = fetch_data(cur1, TABLE_NAME)
        df['zip_code'] = df.apply(process_zip, axis=1)
        new_df = df[['zip_code', 'ccvi_score', 'ccvi_category']]
        engine = create_engine(
            f"postgresql://{GOLD_DATABASE['user']}:{GOLD_DATABASE['password']}"
            f"@{GOLD_DATABASE['host']}:{GOLD_DATABASE['port']}/{GOLD_DATABASE['dbname']}"
        )
        new_df.to_sql(TABLE_NAME, engine, if_exists='append', index=False)

    except Exception as e:
        print("Error occurred:", e)

    finally:
        cur1.close()
        conn1.close()
