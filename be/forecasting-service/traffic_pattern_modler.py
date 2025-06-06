import pandas as pd
import pickle
import xgboost as xgb
from sklearn.model_selection import train_test_split
from sklearn.metrics import mean_squared_error
from sqlalchemy import create_engine
import numpy as np

DB_USER = 'postgres'
DB_PASSWORD = 'root'
DB_HOST = 'localhost'
DB_PORT = '5432'
DB_NAME = 'ChicagoBusinessIntelligence_SILVER'
MODEL_FILENAME = 'xgb_trip_model.pkl'

engine = create_engine(f'postgresql+psycopg2://{DB_USER}:{DB_PASSWORD}@{DB_HOST}:{DB_PORT}/{DB_NAME}')
query = """
SELECT pickup_centroid_location, dropoff_centroid_location
FROM public.taxi_trips
"""
df = pd.read_sql(query, engine)

def extract_zip(location_str):
    if location_str and 'ZIP=' in location_str:
        return location_str.split('ZIP=')[-1].strip()
    return None

df['pickup_zip'] = df['pickup_centroid_location'].apply(extract_zip)
df['dropoff_zip'] = df['dropoff_centroid_location'].apply(extract_zip)

trip_zips = pd.concat([
    df[['pickup_zip']].rename(columns={'pickup_zip': 'zip'}),
    df[['dropoff_zip']].rename(columns={'dropoff_zip': 'zip'})
], ignore_index=True)

trip_zips = trip_zips.dropna()
zip_trip_counts = trip_zips['zip'].value_counts().reset_index()
zip_trip_counts.columns = ['zip', 'trip_count']

np.random.seed(42)
zip_trip_counts['month'] = np.random.randint(1, 13, size=len(zip_trip_counts))
zip_trip_counts['dow'] = np.random.randint(0, 7, size=len(zip_trip_counts))

X = zip_trip_counts[['month', 'dow']]
y = zip_trip_counts['trip_count']

X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42)
model = xgb.XGBRegressor(objective='reg:squarederror', n_estimators=100)
model.fit(X_train, y_train)

with open(MODEL_FILENAME, 'wb') as f:
    pickle.dump(model, f)

print(f"Model saved to {MODEL_FILENAME}")
