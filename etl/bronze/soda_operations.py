import requests

def fetch_total_record_count(url):
    params = {
        "$select": "count(*)"
    }
    response = requests.get(url, params=params, timeout=200000)
    data = response.json()
    record_count = int(data[0]['count'])
    print("Total records for ", url, " -> ", record_count)
    return record_count
