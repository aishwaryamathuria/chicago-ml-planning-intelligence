import React, { useEffect, useState } from "react";
import { MapContainer, TileLayer, GeoJSON } from "react-leaflet";
import "leaflet/dist/leaflet.css";

export default function Report2() {
  const [geoJsonData, setGeoJsonData] = useState(null);
  const [zipIndexData, setZipIndexData] = useState({});
  const [selectedDate, setSelectedDate] = useState("");

  useEffect(() => {
    fetch("http://localhost:5173/public/boundary_zipcode.geojson")
      .then((response) => response.json())
      .then((data) => setGeoJsonData(data))
      .catch((error) => console.error("Error loading GeoJSON:", error));
  }, []);

  useEffect(() => {
    if (!selectedDate) return;
    fetch(`http://localhost:8083/predict-covid-cases?date=${selectedDate}`)
      .then((res) => res.json())
      .then((data) => {
        console.log(data);
        setZipIndexData(data["predicted_case_counts"]);
      })
      .catch((err) => {
        console.error("Error fetching zip index data:", err);
        setZipIndexData({});
      });
  }, [selectedDate]);

  const style = (feature) => {
    const zip = feature.properties.zip;
    const index = zipIndexData[zip];
    const idxColor =
      index === undefined
        ? "transparent"
        : index < 0.32
        ? "transparent"
        : index < 0.48
        ? "orange"
        : "red";

    return {
      fillColor: idxColor,
      weight: 1,
      color: "black",
      fillOpacity: 0.7,
    };
  };

  const chicagoCenter = [41.8781, -87.6298];

  return (
    <div>
      <label className="inline-flex items-center space-x-3 text-lg font-semibold text-gray-700">
        <span>Select Date:</span>
        <input
          type="date"
          value={selectedDate}
          onChange={(e) => setSelectedDate(e.target.value)}
          className="p-3 text-lg rounded-md border border-gray-300 shadow-md
                    focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
        />
      </label>

      <MapContainer
        center={chicagoCenter}
        zoom={10}
        style={{ height: "600px", width: "100%", marginTop: "10px" }}
      >
        <TileLayer
          attribution="&copy; OpenStreetMap contributors"
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        {geoJsonData && (
          <GeoJSON
            data={geoJsonData}
            style={style}
            onEachFeature={(feature, layer) => {
              const zip = String(feature.properties.zip);
              layer.bindTooltip(`ZIP: ${zip}`);
            }}
          />
        )}
      </MapContainer>
    </div>
  );
}
