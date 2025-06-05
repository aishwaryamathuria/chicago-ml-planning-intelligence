import requests
import soda_operations as soda_operations
import db_operations as db_operations
import pandas as pd

API_ENDPOINT = "https://data.cityofchicago.org/resource/ydr8-5enu.json"
DATABASE_NAME = "ChicagoBusinessIntelligence_BRONZE"
TABLE_NAME = "building_permits"
DESIRED_KEYS = [
                    "id",
                    "permit_",
                    "permit_status",
                    "permit_type",
                    "community_area",
                    "location",
                    "last_updated"
                ]

def compute_geometry(row):
    if isinstance(row.get("location"), dict):
        x = row["location"]['coordinates'][0]
        y = row["location"]['coordinates'][1]
        return f"POINT({x}, {y})"
    elif row.get('xcoordinate') is not None and row.get('ycoordinate') is not None:
        return f"POINT({row['xcoordinate']}, {row['ycoordinate']})"
    return None

if __name__ == "__main__":
    url = API_ENDPOINT
    record_count = soda_operations.fetch_total_record_count(url)
    offset = 0
    while record_count > 0:
        params = {
            "$offset": offset,
            "$limit": 5000
        }
        response = requests.get(url, params=params)
        data = response.json()
        df = pd.DataFrame(data)
        df["location"] = df.apply(compute_geometry, axis=1)
        df["last_updated"] = pd.Timestamp.now()
        db_operations.add_data(DATABASE_NAME, TABLE_NAME, df, DESIRED_KEYS)
        print("Inserted 5000 records.")
        record_count = record_count - 5000
        offset = offset + 5000

