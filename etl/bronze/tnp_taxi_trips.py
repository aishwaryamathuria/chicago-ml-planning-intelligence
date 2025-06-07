import requests
import soda_operations as soda_operations
import db_operations as db_operations
import pandas as pd

API_ENDPOINT = "https://data.cityofchicago.org/resource/m6dm-c72p.json"
DATABASE_NAME = "ChicagoBusinessIntelligence_BRONZE"
TABLE_NAME = "tnp_trips"
DESIRED_KEYS = [
                    "trip_id",
                    "trip_start_timestamp",
                    "trip_end_timestamp",
                    "pickup_community_area",
                    "dropoff_community_area",
                    "pickup_centroid_location",
                    "dropoff_centroid_location",
                    "last_updated"
                ]

def to_wkt_string(loc):
    if isinstance(loc, dict):
        coords = loc.get("coordinates")
        if isinstance(coords, list) and len(coords) == 2:
            return f'POINT({coords[0]}, {coords[1]})'
    return None

if __name__ == "__main__":
    filename = "taxi_trips.txt"
    offset = 0
    try:
        with open(filename, "r") as f:
            offset = int(f.read())
    except:
        offset = 0
        with open(filename, "w") as f:
            f.write("0")
    url = API_ENDPOINT
    record_count = soda_operations.fetch_total_record_count(url)
    while record_count > 0:
        params = {
            "$offset": offset,
            "$limit": 5000,
            "$where": "trip_start_timestamp between '2021-01-01T00:00:00' and '2023-12-31T23:59:59'",
            "$order": "trip_start_timestamp ASC"
        }
        response = requests.get(url, params=params)
        data = response.json()
        df = pd.DataFrame(data)
        df["pickup_centroid_location"] = df["pickup_centroid_location"].apply(to_wkt_string)
        df["dropoff_centroid_location"] = df["dropoff_centroid_location"].apply(to_wkt_string)
        df["last_updated"] = pd.Timestamp.now()
        db_operations.add_data(DATABASE_NAME, TABLE_NAME, df, DESIRED_KEYS)
        record_count = record_count - 5000
        print("Inserted 5000 records. Remaining ", record_count)
        offset = offset + 5000
        with open(filename, "w") as f:
            f.write(offset)