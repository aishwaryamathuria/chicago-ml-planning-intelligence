import * as React from "react";
import { DataGrid } from "@mui/x-data-grid";
import { TextField } from "@mui/material";

const columns = [
  { field: "pickup_location", headerName: "Pickup ZIP Code", width: 300 },
  { field: "dropoff_location", headerName: "Dropoff ZIP Code", width: 300 },
  { field: "case_count", headerName: "Case Count", width: 300, type: "number" },
];

export default function DataTable() {
  const [rows, setRows] = React.useState([]);
  const [searchText, setSearchText] = React.useState("");
  const [filteredRows, setFilteredRows] = React.useState([]);

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

  return (
    <div className="w-[900px] h-[500px]">
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
