import requests
import pandas as pd
from sqlalchemy import create_engine

db_user = 'postgres'
db_password = 'root'
db_host = 'localhost'
db_port = '5432'

def add_data(db_name, table_name, df, desired_keys):
    df_filtered = df[desired_keys]
    engine = create_engine(f'postgresql://{db_user}:{db_password}@{db_host}:{db_port}/{db_name}')
    df_filtered.to_sql(table_name, engine, if_exists="append", index=False)