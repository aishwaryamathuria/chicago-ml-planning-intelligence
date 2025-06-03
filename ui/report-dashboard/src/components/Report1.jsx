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
  { field: "zip_code", headerName: "ZIP CODE", width: 300 },
  { field: "ccvi_score", headerName: "CCVI Score", width: 300 },
  { field: "ccvi_category", headerName: "CCVI", width: 300 },
];

export default function DataTable() {
  const [rows, setRows] = React.useState([]);
  const [searchText, setSearchText] = React.useState("");
  const [filteredRows, setFilteredRows] = React.useState([]);
  const [barChartData, setBarChartData] = React.useState({
    labels: [],
    datasets: [],
  });

  React.useEffect(() => {
    fetch(`${import.meta.env.VITE_API_URL}/api/report1`)
      .then((res) => res.json())
      .then((data) => {
        const mappedData = data.map((item, index) => ({
          id: index,
          zip_code: item.zip_code,
          ccvi_score: item.ccvi_score,
          ccvi_category: item.ccvi_category,
        }));
        setRows(mappedData);
        setFilteredRows(mappedData);
        const labels = data.map((item) => item.zip_code);
        const counts = data.map((item) => Number(item.ccvi_score));
        setBarChartData({
          labels,
          datasets: [
            {
              label: "CCVI Score",
              data: counts,
              backgroundColor: "hsl(12, 65.80%, 52.90%)",
            },
          ],
        });
      })
      .catch((error) => {
        console.error("Failed to fetch data:", error);
      });
  }, []);

  React.useEffect(() => {
    const filtered = rows.filter((row) => {
      return (
        row.zip_code.toLowerCase().includes(searchText.toLowerCase()) ||
        row.ccvi_score.toLowerCase().includes(searchText.toLowerCase()) ||
        row.ccvi_category.toLowerCase().includes(searchText.toLowerCase())
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
        text: "CCVI Score per ZIP code",
      },
    },
  };

  return (
    <div className="w-full h-[1000px] flex flex-wrap items-center justify-center">
      <div className="text-2xl font-semibold w-full pt-[20px]">
        Alert Sent to ZIP Codes
      </div>
      <div className="w-[950px] h-[500px">
        <Bar options={options} data={barChartData} />
      </div>

      <div className="w-full h-[500px]">
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
