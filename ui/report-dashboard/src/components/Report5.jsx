import React, { useState } from "react";

const CONSTANT_ZIP_CODES = [
  60601, 60602, 60603, 60604, 60605, 60606, 60607, 60608, 60609, 60610, 60611,
  60612, 60613, 60614, 60615, 60616, 60617, 60618, 60619, 60620, 60621, 60622,
  60623, 60624, 60625, 60626, 60628, 60629, 60630, 60631, 60632, 60633, 60634,
  60636, 60637, 60638, 60639, 60640, 60641, 60642, 60643, 60644, 60645, 60646,
  60647, 60649, 60651, 60652, 60653, 60654, 60655, 60656, 60657, 60659, 60660,
  60661,
];

export default function TripPredictionRange() {
  const [selectedZip, setSelectedZip] = useState("");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [totalTrips, setTotalTrips] = useState(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const getISOWeek = (date) => {
    const tmp = new Date(
      Date.UTC(date.getFullYear(), date.getMonth(), date.getDate())
    );
    const dayNum = tmp.getUTCDay() || 7;
    tmp.setUTCDate(tmp.getUTCDate() + 4 - dayNum);
    const yearStart = new Date(Date.UTC(tmp.getUTCFullYear(), 0, 1));
    return Math.ceil(((tmp - yearStart) / 86400000 + 1) / 7);
  };

  const getDatesInRange = (start, end) => {
    const dateArray = [];
    const current = new Date(start);
    while (current <= end) {
      dateArray.push(new Date(current));
      current.setDate(current.getDate() + 1);
    }
    return dateArray;
  };

  const handlePredict = async () => {
    setError("");
    setTotalTrips(null);

    if (!selectedZip || !startDate || !endDate) {
      setError("Please select a ZIP code and a date range.");
      return;
    }

    const start = new Date(startDate);
    const end = new Date(endDate);
    if (start > end) {
      setError("Start date must be before end date.");
      return;
    }

    setLoading(true);

    const dates = getDatesInRange(start, end);
    let total = 0;

    try {
      for (let date of dates) {
        const requestBody = {
          records: [
            {
              zip: Number(selectedZip),
              month: date.getUTCMonth() + 1,
              day: date.getUTCDate(),
              day_of_week: date.getUTCDay(),
              week: getISOWeek(date),
            },
          ],
        };

        const res = await fetch(
          "http://localhost:8083/predict-traffic-pattern",
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(requestBody),
          }
        );

        if (!res.ok) {
          const errText = await res.text();
          throw new Error(errText);
        }

        const data = await res.json();
        const dailyPrediction = data[0].predicted_trip_count;
        total += dailyPrediction;
      }

      setTotalTrips(total);
    } catch (err) {
      setError(`Prediction failed: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-6 max-w-xl mx-auto bg-white rounded shadow space-y-4">
      <h2 className="text-xl font-bold text-gray-700">
        Trip Prediction (Date Range)
      </h2>

      <label className="block">
        <span className="text-gray-700">ZIP Code:</span>
        <select
          value={selectedZip}
          onChange={(e) => setSelectedZip(e.target.value)}
          className="mt-1 block w-full p-2 border rounded"
        >
          <option value="">-- Select ZIP --</option>
          {CONSTANT_ZIP_CODES.map((zip) => (
            <option key={zip} value={zip}>
              {zip}
            </option>
          ))}
        </select>
      </label>

      <div className="flex gap-4">
        <label className="block w-1/2">
          <span className="text-gray-700">Start Date:</span>
          <input
            type="date"
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
            className="mt-1 block w-full p-2 border rounded"
          />
        </label>

        <label className="block w-1/2">
          <span className="text-gray-700">End Date:</span>
          <input
            type="date"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            className="mt-1 block w-full p-2 border rounded"
          />
        </label>
      </div>

      <button
        onClick={handlePredict}
        className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 disabled:opacity-50"
        disabled={loading}
      >
        {loading ? "Predicting..." : "Predict Total Trips"}
      </button>

      {totalTrips !== null && (
        <div className="text-green-700 font-semibold">
          Total Predicted Trips: {totalTrips.toFixed(2)}
        </div>
      )}

      {error && <div className="text-red-600">{error}</div>}
    </div>
  );
}
