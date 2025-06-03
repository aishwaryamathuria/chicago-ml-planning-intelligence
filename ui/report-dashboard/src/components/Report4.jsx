import * as React from "react";
import { DataGrid } from "@mui/x-data-grid";
import { TextField } from "@mui/material";
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend,
} from "chart.js";
import { Bar } from "react-chartjs-2";

ChartJS.register(
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend
);

const columns = [
  { field: "trip_id", headerName: "Trip ID", width: 100 },
  { field: "trip_start_timestamp", headerName: "Start Time", width: 200 },
  { field: "trip_end_timestamp", headerName: "End Time", width: 200 },
  { field: "pickup_zip_code", headerName: "Pickup Zip Code", width: 150 },
  { field: "dropoff_zip_code", headerName: "Dropoff Zip Code", width: 150 },
  {
    field: "pickup_neighborhood",
    headerName: "Pickup Neighborhood",
    width: 200,
  },
  {
    field: "dropoff_neighborhood",
    headerName: "Dropoff Neighborhood",
    width: 200,
  },
];

export default function DataTable() {
  const [rows, setRows] = React.useState([]);
  const [searchText, setSearchText] = React.useState("");
  const [filteredRows, setFilteredRows] = React.useState([]);
  const [barChart1Data, setBarChart1Data] = React.useState({
    labels: [],
    datasets: [],
  });
  const [barChart2Data, setBarChart2Data] = React.useState({
    labels: [],
    datasets: [],
  });

  React.useEffect(() => {
    fetch(`${import.meta.env.VITE_API_URL}/api/report4`)
      .then((res) => res.json())
      .then((data) => {
        const mappedData = data.map((item, index) => ({
          id: index,
          trip_id: item.trip_id,
          trip_start_timestamp: item.trip_start_timestamp,
          trip_end_timestamp: item.trip_end_timestamp,
          pickup_zip_code: item.pickup_zip_code,
          dropoff_zip_code: item.dropoff_zip_code,
          pickup_neighborhood: item.pickup_neighborhood,
          dropoff_neighborhood: item.dropoff_neighborhood,
        }));
        setRows(mappedData);
        setFilteredRows(mappedData);
      })
      .catch((error) => {
        console.error("Failed to fetch data:", error);
      });

    fetch(`${import.meta.env.VITE_API_URL}/api/report4a`)
      .then((res) => res.json())
      .then((data) => {
        const labels = data.map((item) => item.pickup_neighborhood);
        const counts = data.map((item) => Number(item.trip_count));
        setBarChart1Data({
          labels,
          datasets: [
            {
              label: "Trip Count",
              data: counts,
              backgroundColor: "hsl(215, 71.30%, 50.80%)",
            },
          ],
        });
      })
      .catch((error) => {
        console.error("Failed to fetch bar chart data:", error);
      });

    fetch(`${import.meta.env.VITE_API_URL}/api/report4b`)
      .then((res) => res.json())
      .then((data) => {
        const labels = data.map((item) => item.dropoff_neighborhood);
        const counts = data.map((item) => Number(item.trip_count));
        setBarChart2Data({
          labels,
          datasets: [
            {
              label: "Trip Count",
              data: counts,
              backgroundColor: "hsl(84, 67.20%, 47.80%)",
            },
          ],
        });
      })
      .catch((error) => {
        console.error("Failed to fetch bar chart data:", error);
      });
  }, []);

  React.useEffect(() => {
    const filtered = rows.filter((row) => {
      return (
        row.trip_id.toLowerCase().includes(searchText.toLowerCase()) ||
        row.trip_start_timestamp
          .toLowerCase()
          .includes(searchText.toLowerCase()) ||
        row.trip_end_timestamp
          .toLowerCase()
          .includes(searchText.toLowerCase()) ||
        row.pickup_zip_code.toString().includes(searchText) ||
        row.dropoff_zip_code.toString().includes(searchText) ||
        row.pickup_neighborhood
          .toLowerCase()
          .includes(searchText.toLowerCase()) ||
        row.dropoff_neighborhood
          .toLowerCase()
          .includes(searchText.toLowerCase())
      );
    });
    setFilteredRows(filtered);
  }, [searchText, rows]);

  const options1 = {
    responsive: true,
    plugins: {
      legend: {
        position: "top",
      },
      title: {
        display: true,
        text: "Trips by Pickup Neighborhood",
      },
    },
  };

  const options2 = {
    responsive: true,
    plugins: {
      legend: {
        position: "top",
      },
      title: {
        display: true,
        text: "Trips by Dropoff Neighborhood",
      },
    },
  };

  return (
    <div className="w-full h-[1000px] flex flex-wrap">
      <div style={{ width: "600px", height: 300, marginBottom: 24 }}>
        <Bar options={options1} data={barChart1Data} />
      </div>

      <div style={{ width: "600px", height: 300, marginBottom: 24 }}>
        <Bar options={options2} data={barChart2Data} />
      </div>
      <div className="h-[700px]">
        <TextField
          label="Search Name or City"
          variant="outlined"
          size="small"
          onChange={(e) => setSearchText(e.target.value)}
          style={{ marginBottom: 16 }}
        />

        <DataGrid
          rows={filteredRows}
          columns={columns}
          pageSize={5}
          rowsPerPageOptions={[5, 10]}
          pagination
        />
      </div>
    </div>
  );
}
