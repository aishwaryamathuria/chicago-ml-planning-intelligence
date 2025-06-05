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

TABLE_NAME = "covid_cases"

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
        df = df.rename(columns={'week_start': 'start_date'})
        df = df.rename(columns={'week_end': 'end_date'})
        df = df.rename(columns={'cases_weekly': 'case_count'})
        df = df.rename(columns={'percent_tested_positive_weekly': 'positive_test_percent'})
        new_df = df[['zip_code', 'start_date', 'end_date', 'case_count', 'positive_test_percent']]
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
