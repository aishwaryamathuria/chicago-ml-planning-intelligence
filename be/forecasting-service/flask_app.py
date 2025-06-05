# app.py
from flask import Flask, request, jsonify
import pickle
import pandas as pd
from datetime import datetime

CONSTANT_ZIP_CODES = [
    60601, 60602, 60603, 60604, 60605, 60606, 60607, 60608, 60609, 60610,
    60611, 60612, 60613, 60614, 60615, 60616, 60617, 60618, 60619, 60620,
    60621, 60622, 60623, 60624, 60625, 60626, 60628, 60629, 60630, 60631,
    60632, 60633, 60634, 60636, 60637, 60638, 60639, 60640, 60641, 60642,
    60643, 60644, 60645, 60646, 60647, 60649, 60651, 60652, 60653, 60654,
    60655, 60656, 60657, 60659, 60660, 60661
]

# Covid case predictor model
with open("rf_covid_model_clean.pkl", "rb") as f_model:
    model = pickle.load(f_model)

app = Flask(__name__)

@app.after_request
def add_cors_headers(response):
    response.headers['Access-Control-Allow-Origin'] = '*'
    response.headers['Access-Control-Allow-Headers'] = '*'
    return response

@app.route("/")
def index():
    return "<h1>MSDS-432 Group 5: Chicago Business Intelligence for Strategic Planning</h1>"

@app.route("/predict-covid-cases", methods=["GET", "OPTIONS"])
def predict():
    if request.method == "OPTIONS":
        return '', 204
    try:
        date_str = request.args.get("date")
        date = pd.to_datetime(date_str)
        date_ordinal = date.toordinal()
    except (KeyError, ValueError) as e:
        return jsonify({"error": "Invalid input. Expected 'date' (YYYY-MM-DD)."}), 400

    input_df = pd.DataFrame([{
        "zip_code_num": zip_code,
        "date_ordinal": date_ordinal
    } for zip_code in CONSTANT_ZIP_CODES])

    predictions = model.predict(input_df)

    results = {
        str(zip_code): round(pred, 2)
        for zip_code, pred in zip(CONSTANT_ZIP_CODES, predictions)
    }

    return jsonify({
        "date": date.strftime("%Y-%m-%d"),
        "predicted_case_counts": results
    })

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=8083, debug=True)
