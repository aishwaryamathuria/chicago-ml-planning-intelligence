import pandas as pd
import numpy as np
import pickle
import xgboost as xgb
from sklearn.model_selection import train_test_split
from sklearn.metrics import mean_squared_error
from sqlalchemy import create_engine

DB_USER = 'postgres'
DB_PASSWORD = 'root'
DB_HOST = 'localhost'
DB_PORT = '5432'
DB_NAME = 'ChicagoBusinessIntelligence_GOLD'
MODEL_FILENAME = 'xgb_trip_model.pkl'

engine = create_engine(f'postgresql+psycopg2://{DB_USER}:{DB_PASSWORD}@{DB_HOST}:{DB_PORT}/{DB_NAME}')
query = """
SELECT trip_start_timestamp, pickup_zip_code, dropoff_zip_code
FROM public.stg_taxi_trips
WHERE trip_start_timestamp IS NOT NULL
"""
df = pd.read_sql(query, engine)

def extract_zip(location_str):
    if location_str and 'ZIP=' in location_str:
        return location_str.split('ZIP=')[-1].strip()
    return None

df['pickup_zip'] = df['pickup_zip_code']
df['dropoff_zip'] = df['dropoff_zip_code']
df['trip_date'] = pd.to_datetime(df['trip_start_timestamp']).dt.date
print(df)
pickup_df = df[['trip_date', 'pickup_zip']].dropna().rename(columns={'pickup_zip': 'zip'})
dropoff_df = df[['trip_date', 'dropoff_zip']].dropna().rename(columns={'dropoff_zip': 'zip'})
all_trips = pd.concat([pickup_df, dropoff_df], ignore_index=True)

trip_counts = all_trips.groupby(['trip_date', 'zip']).size().reset_index(name='trip_count')

trip_counts['trip_date'] = pd.to_datetime(trip_counts['trip_date'])
trip_counts['month'] = trip_counts['trip_date'].dt.month
trip_counts['day'] = trip_counts['trip_date'].dt.day
trip_counts['day_of_week'] = trip_counts['trip_date'].dt.dayofweek
trip_counts['week'] = trip_counts['trip_date'].dt.isocalendar().week

trip_counts['zip'] = trip_counts['zip'].astype(str)
zip_codes = trip_counts['zip'].unique()
zip_to_int = {z: i for i, z in enumerate(zip_codes)}
trip_counts['zip_encoded'] = trip_counts['zip'].map(zip_to_int)

X = trip_counts[['zip_encoded', 'month', 'day', 'day_of_week', 'week']]
y = trip_counts['trip_count']

X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42)

model = xgb.XGBRegressor(objective='reg:squarederror', n_estimators=100)
model.fit(X_train, y_train)

y_pred = model.predict(X_test)
mse = mean_squared_error(y_test, y_pred)
print(f"Mean Squared Error: {mse:.2f}")

with open(MODEL_FILENAME, 'wb') as f:
    pickle.dump(model, f)

print(f"Model saved as {MODEL_FILENAME}")