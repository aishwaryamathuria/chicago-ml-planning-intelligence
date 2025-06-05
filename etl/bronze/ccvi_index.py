import requests
import soda_operations as soda_operations
import db_operations as db_operations
import pandas as pd

API_ENDPOINT = "https://data.cityofchicago.org/resource/xhc6-88s9.json"
DATABASE_NAME = "ChicagoBusinessIntelligence_BRONZE"
TABLE_NAME = "ccvi_index"
DESIRED_KEYS = [
                    "geography_type",
                    "community_area_or_zip",
                    "community_area_name",
                    "ccvi_score",
                    "ccvi_category",
                    "location",
                    "last_updated"
                ]

def to_wkt_string(loc):
    if isinstance(loc, dict):
        coords = loc.get("coordinates")
        if isinstance(coords, list) and len(coords) == 2:
            return f'POINT({coords[0]}, {coords[1]})'
    return None

if __name__ == "__main__":
    url = API_ENDPOINT
    record_count = soda_operations.fetch_total_record_count(url)
    offset = 0
    while record_count > 0:
        params = {
            "$offset": offset,
            "$limit": 1000
        }
        response = requests.get(url, params=params)
        data = response.json()
        df = pd.DataFrame(data)
        df["location"] = df["location"].apply(to_wkt_string)
        df["last_updated"] = pd.Timestamp.now()
        db_operations.add_data(DATABASE_NAME, TABLE_NAME, df, DESIRED_KEYS)
        print("Inserted 1000 records.")
        record_count = record_count - 1000
        offset = offset + 1000