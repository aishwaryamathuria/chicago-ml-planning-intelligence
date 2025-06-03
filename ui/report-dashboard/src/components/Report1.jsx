import * as React from "react";
import { DataGrid } from "@mui/x-data-grid";
import { TextField } from "@mui/material";

const columns = [
  { field: "zip_code", headerName: "ZIP CODE", width: 400 },
  { field: "ccvi_category", headerName: "CCVI", width: 400 },
];

export default function DataTable() {
  const [rows, setRows] = React.useState([]);
  const [searchText, setSearchText] = React.useState("");
  const [filteredRows, setFilteredRows] = React.useState([]);

  React.useEffect(() => {
    fetch(`${import.meta.env.VITE_API_URL}/api/report1`)
      .then((res) => res.json())
      .then((data) => {
        const mappedData = data.map((item, index) => ({
          id: index,
          zip_code: item.zip_code,
          ccvi_category: item.ccvi_category,
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
        row.ccvi_category.toLowerCase().includes(searchText.toLowerCase())
      );
    });
    setFilteredRows(filtered);
  }, [searchText, rows]);

  return (
    <div class="w-[800px] h-[500px]">
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
