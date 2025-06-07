import requests
import soda_operations as soda_operations
import db_operations as db_operations
import pandas as pd

API_ENDPOINT = "https://data.cityofchicago.org/resource/yhhz-zm2v.json"
DATABASE_NAME = "ChicagoBusinessIntelligence_BRONZE"
TABLE_NAME = "covid_cases"
DESIRED_KEYS = [
                    "zip_code",
                    "week_start",
                    "week_end",
                    "cases_weekly",
                    "percent_tested_positive_weekly",
                    "last_updated"
                ]

if __name__ == "__main__":
    filename = "covid_cases.txt"
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
        with open(filename, "w") as f:
            f.write(offset)