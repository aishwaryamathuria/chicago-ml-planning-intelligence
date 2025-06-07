import requests
import json

SOURCE_ENDPOINT = 'http://localhost:8080/api/report1'
CONSUMER_ENDPOINTS = [
    'http://localhost:8083/alert-receiver'
]

def fetch_ccvi_data(url):
    response = requests.get(url)
    return response.json()

def push_to_consumers(data, endpoints):
    headers = {'Content-Type': 'application/json'}
    for endpoint in endpoints:
        response = requests.post(endpoint, data=json.dumps(data), headers=headers)

def main():
    data = fetch_ccvi_data(SOURCE_ENDPOINT)
    if data:
        push_to_consumers(data, CONSUMER_ENDPOINTS)

if __name__ == '__main__':
    main()
