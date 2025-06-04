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
  { field: "zip_code", headerName: "ZIP Code", width: 200 },
  { field: "permit_number", headerName: "Permit Number", width: 350 },
  { field: "permit_status", headerName: "Permit Status", width: 200 },
  { field: "neighborhood", headerName: "Neighborhood", width: 200 },
  { field: "community_area", headerName: "Community Area", width: 200 },
  {
    field: "per_capita_income",
    headerName: "Per Capita Income",
    width: 150,
    type: "number",
  },
];

export default function DataTable() {
  const [rows, setRows] = React.useState([]);
  const [searchText, setSearchText] = React.useState("");
  const [filteredRows, setFilteredRows] = React.useState([]);

  React.useEffect(() => {
    fetch(`${import.meta.env.VITE_API_URL}/api/report6`)
      .then((res) => res.json())
      .then((data) => {
        const mappedData = data.map((item, index) => ({
          id: index,
          zip_code: item.zip_code,
          permit_number: item.permit_number,
          permit_status: item.permit_status,
          neighborhood: item.neighborhood,
          community_area: item.community_area,
          per_capita_income: item.per_capita_income,
        }));
        setRows(mappedData);
        setFilteredRows(mappedData);
      })
      .catch((error) => {
        console.error("Failed to fetch data:", error);
      });
  }, []);

  React.useEffect(() => {
    const filtered = rows.filter((row) => {
      return (
        row.zip_code.toLowerCase().includes(searchText.toLowerCase()) ||
        row.permit_number.toLowerCase().includes(searchText.toLowerCase()) ||
        row.permit_status.toLowerCase().includes(searchText.toLowerCase()) ||
        row.neighborhood.toLowerCase().includes(searchText.toLowerCase()) ||
        row.community_area.toString().includes(searchText.toLowerCase()) ||
        row.per_capita_income.toString().includes(searchText.toLowerCase())
      );
    });
    setFilteredRows(filtered);
  }, [searchText, rows]);

  return (
    <div className="w-full h-[500px]">
      <div className="text-2xl font-semibold w-full pt-[20px] pb-[20px]">
        Low interest loan candidate permits
      </div>
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
  );
}
