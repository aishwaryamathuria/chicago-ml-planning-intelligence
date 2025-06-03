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
  { field: "pickup_location", headerName: "Pickup ZIP Code", width: 300 },
  { field: "dropoff_location", headerName: "Dropoff ZIP Code", width: 300 },
  { field: "case_count", headerName: "Case Count", width: 300, type: "number" },
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
  const [barChart3Data, setBarChart3Data] = React.useState({
    labels: [],
    datasets: [],
  });
  const [barChart4Data, setBarChart4Data] = React.useState({
    labels: [],
    datasets: [],
  });

  React.useEffect(() => {
    fetch(`${import.meta.env.VITE_API_URL}/api/report3`)
      .then((res) => res.json())
      .then((data) => {
        const mappedData = data.map((item, index) => ({
          id: index,
          pickup_location: item.pickup_location,
          dropoff_location: item.dropoff_location,
          case_count: Number(item.case_count),
        }));
        setRows(mappedData);
        setFilteredRows(mappedData);
      })
      .catch((error) => {
        console.error("Failed to fetch data:", error);
      });

    fetch(`${import.meta.env.VITE_API_URL}/api/report3a`)
      .then((res) => res.json())
      .then((data) => {
        const labels = data.map((item) => item.zip_code);
        const counts = data.map((item) => Number(item.case_count));
        setBarChart1Data({
          labels,
          datasets: [
            {
              label: "Trip from O'Hare",
              data: counts,
              backgroundColor: "hsl(215, 71.30%, 50.80%)",
            },
          ],
        });
      })
      .catch((error) => {
        console.error("Failed to fetch bar chart data:", error);
      });

    fetch(`${import.meta.env.VITE_API_URL}/api/report3b`)
      .then((res) => res.json())
      .then((data) => {
        const labels = data.map((item) => item.zip_code);
        const counts = data.map((item) => Number(item.case_count));
        setBarChart2Data({
          labels,
          datasets: [
            {
              label: "Trip from Midway",
              data: counts,
              backgroundColor: "hsl(89, 70.20%, 48.60%)",
            },
          ],
        });
      })
      .catch((error) => {
        console.error("Failed to fetch bar chart data:", error);
      });

    fetch(`${import.meta.env.VITE_API_URL}/api/report3c`)
      .then((res) => res.json())
      .then((data) => {
        const labels = data.map((item) => item.zip_code);
        const counts = data.map((item) => Number(item.case_count));
        setBarChart3Data({
          labels,
          datasets: [
            {
              label: "Trip to O'Hare",
              data: counts,
              backgroundColor: "hsl(8, 60.30%, 54.50%)",
            },
          ],
        });
      })
      .catch((error) => {
        console.error("Failed to fetch bar chart data:", error);
      });

    fetch(`${import.meta.env.VITE_API_URL}/api/report3d`)
      .then((res) => res.json())
      .then((data) => {
        const labels = data.map((item) => item.zip_code);
        const counts = data.map((item) => Number(item.case_count));
        setBarChart4Data({
          labels,
          datasets: [
            {
              label: "Trip to Midway",
              data: counts,
              backgroundColor: "hsl(299, 52.80%, 51.00%)",
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
        row.pickup_location.toLowerCase().includes(searchText.toLowerCase()) ||
        row.dropoff_location.toLowerCase().includes(searchText.toLowerCase()) ||
        row.case_count.toString().includes(searchText.toLowerCase())
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
        text: "Trips from O'Hare",
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
        text: "Trips from Midway",
      },
    },
  };

  const options3 = {
    responsive: true,
    plugins: {
      legend: {
        position: "top",
      },
      title: {
        display: true,
        text: "Trips to O'Hare",
      },
    },
  };

  const options4 = {
    responsive: true,
    plugins: {
      legend: {
        position: "top",
      },
      title: {
        display: true,
        text: "Trips to Midway",
      },
    },
  };

  return (
    <div className="w-full h-[1000px] flex flex-wrap">
      <div className="text-2xl font-semibold w-full pt-[20px] pb-[20px]">
        Trips to and from O'Hare & Midway Airports
      </div>
      <div style={{ width: "600px", height: 300, marginBottom: 24 }}>
        <Bar options={options1} data={barChart1Data} />
      </div>

      <div style={{ width: "600px", height: 300, marginBottom: 24 }}>
        <Bar options={options2} data={barChart2Data} />
      </div>

      <div style={{ width: "600px", height: 300, marginBottom: 24 }}>
        <Bar options={options3} data={barChart3Data} />
      </div>

      <div style={{ width: "600px", height: 300, marginBottom: 24 }}>
        <Bar options={options4} data={barChart4Data} />
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
