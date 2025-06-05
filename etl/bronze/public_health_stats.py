import requests
import soda_operations as soda_operations
import db_operations as db_operations
import pandas as pd

API_ENDPOINT = "https://data.cityofchicago.org/resource/iqnk-2tcu.json"
DATABASE_NAME = "ChicagoBusinessIntelligence_BRONZE"
TABLE_NAME = "public_health_stats"
DESIRED_KEYS = [
                    "community_area",
                    "community_area_name",
                    "per_capita_income",
                    "unemployment",
                    "below_poverty_level",
                    "last_updated"
                ]

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
        df["last_updated"] = pd.Timestamp.now()
        db_operations.add_data(DATABASE_NAME, TABLE_NAME, df, DESIRED_KEYS)
        print("Inserted 1000 records.")
        record_count = record_count - 1000
        offset = offset + 1000