import * as React from "react";
import { DataGrid } from "@mui/x-data-grid";
import { TextField } from "@mui/material";
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
} from "chart.js";
import { Line } from "react-chartjs-2";

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend
);

const columns = [
  { field: "permit_number", headerName: "Permit Number", width: 200 },
  { field: "permit_type", headerName: "Permit Type", width: 350 },
  { field: "community_area_name", headerName: "Community Area", width: 200 },
  { field: "neighborhood", headerName: "Neighborhood", width: 200 },
  {
    field: "unemployment_rate",
    headerName: "Unemployment Rate",
    width: 150,
    type: "number",
  },
  {
    field: "below_poverty_level",
    headerName: "Poverty Level",
    width: 150,
    type: "number",
  },
];

export default function DataTable() {
  const [rows, setRows] = React.useState([]);
  const [searchText, setSearchText] = React.useState("");
  const [filteredRows, setFilteredRows] = React.useState([]);
  const [lineChartData, setLineChartData] = React.useState({
    labels: [],
    datasets: [],
  });

  React.useEffect(() => {
    fetch(`${import.meta.env.VITE_API_URL}/api/report5`)
      .then((res) => res.json())
      .then((data) => {
        const mappedData = data.map((item, index) => ({
          id: index,
          permit_number: item.permit_number,
          permit_type: item.permit_type,
          community_area_name: item.community_area_name,
          neighborhood: item.neighborhood,
          unemployment_rate: item.unemployment_rate,
          below_poverty_level: item.below_poverty_level,
        }));
        setRows(mappedData);
        setFilteredRows(mappedData);
      })
      .catch((error) => {
        console.error("Failed to fetch data:", error);
      });

    fetch(`${import.meta.env.VITE_API_URL}/api/report5a`)
      .then((res) => res.json())
      .then((data) => {
        const neighborhoods = data.map((item) => item.neighborhood);
        const unemploymentRates = data.map((item) =>
          parseFloat(item.max_unemployment_rate)
        );
        const povertyLevels = data.map((item) =>
          parseFloat(item.max_below_poverty_level)
        );

        setLineChartData({
          labels: neighborhoods,
          datasets: [
            {
              label: "Unemployment Rate",
              data: unemploymentRates,
              borderColor: "rgba(255, 99, 132, 1)",
              backgroundColor: "rgba(255, 99, 132, 0.2)",
              fill: false,
              tension: 0.3,
            },
            {
              label: "Poverty Level",
              data: povertyLevels,
              borderColor: "rgba(54, 162, 235, 1)",
              backgroundColor: "rgba(54, 162, 235, 0.2)",
              fill: false,
              tension: 0.3,
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
        row.permit_number.toLowerCase().includes(searchText.toLowerCase()) ||
        row.permit_type.toLowerCase().includes(searchText.toLowerCase()) ||
        row.community_area_name
          .toLowerCase()
          .includes(searchText.toLowerCase()) ||
        row.neighborhood.toLowerCase().includes(searchText.toLowerCase()) ||
        row.unemployment_rate.toString().includes(searchText.toLowerCase()) ||
        row.below_poverty_level.toString().includes(searchText.toLowerCase())
      );
    });
    setFilteredRows(filtered);
  }, [searchText, rows]);

  const options = {
    responsive: true,
    plugins: {
      legend: {
        position: "top",
      },
      title: {
        display: true,
        text: "Unemployment vs Poverty Level by Neighborhood",
      },
    },
    scales: {
      y: {
        min: 0,
        max: 100,
      },
    },
  };

  return (
    <div className="w-full flex flex-wrap items-center justify-center">
      <div className="text-2xl font-semibold w-full pt-[20px] pb-[20px]">
        Neighborhoods | High Unemployment and Poverty
      </div>
      <div style={{ width: "900px", height: "auto", marginBottom: 24 }}>
        <Line options={options} data={lineChartData} />
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
