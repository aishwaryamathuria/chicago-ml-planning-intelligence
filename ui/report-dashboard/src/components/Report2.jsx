import React, { useMemo, useState } from "react";
import {
  useReactTable,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  flexRender,
} from "@tanstack/react-table";
import { Bar } from "react-chartjs-2";
import {
  Chart as ChartJS,
  BarElement,
  CategoryScale,
  LinearScale,
  Tooltip,
  Legend,
} from "chart.js";

ChartJS.register(BarElement, CategoryScale, LinearScale, Tooltip, Legend);

const rawData = [
  { id: 1, city: "Austin", trips: 30, category: "A" },
  { id: 2, city: "Chicago", trips: 45, category: "B" },
  { id: 3, city: "Denver", trips: 25, category: "A" },
  { id: 4, city: "Boston", trips: 40, category: "B" },
  { id: 5, city: "Seattle", trips: 35, category: "A" },
  { id: 6, city: "Miami", trips: 28, category: "C" },
  { id: 7, city: "New York", trips: 50, category: "C" },
];

export default function ReportWithReactTable() {
  const [globalFilter, setGlobalFilter] = useState("");

  const columns = useMemo(
    () => [
      {
        accessorKey: "city",
        header: "City",
        footer: (info) => info.column.id,
      },
      {
        accessorKey: "trips",
        header: "Trips",
        footer: (info) => info.column.id,
      },
      {
        accessorKey: "category",
        header: "Category",
        footer: (info) => info.column.id,
      },
    ],
    []
  );

  const table = useReactTable({
    data: rawData,
    columns,
    state: {
      globalFilter,
    },
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  const filteredData = table.getRowModel().rows.map((row) => row.original);

  const chartData = {
    labels: filteredData.map((row) => row.city),
    datasets: [
      {
        label: "Trips",
        data: filteredData.map((row) => row.trips),
        backgroundColor: "rgba(59, 130, 246, 0.7)", // Tailwind blue-500
        borderRadius: 6,
      },
    ],
  };

  return (
    <div className="p-6 space-y-8 bg-gray-50 min-h-screen">
      <div className="text-2xl font-semibold">Trips Report</div>

      <input
        type="text"
        placeholder="Search..."
        value={globalFilter ?? ""}
        onChange={(e) => setGlobalFilter(e.target.value)}
        className="mb-4 p-2 border border-gray-300 rounded w-full max-w-sm"
      />

      <div className="overflow-x-auto bg-white rounded shadow">
        <table className="min-w-full divide-y divide-gray-200 text-sm text-left">
          <thead className="bg-gray-100 text-gray-700">
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th
                    key={header.id}
                    className="px-4 py-3 cursor-pointer"
                    onClick={header.column.getToggleSortingHandler()}
                  >
                    {flexRender(
                      header.column.columnDef.header,
                      header.getContext()
                    )}
                    {{
                      asc: " ↑",
                      desc: " ↓",
                    }[header.column.getIsSorted()] ?? null}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody className="divide-y divide-gray-100">
            {table.getRowModel().rows.map((row) => (
              <tr key={row.id} className="hover:bg-gray-50">
                {row.getVisibleCells().map((cell) => (
                  <td key={cell.id} className="px-4 py-2">
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Chart */}
      <div className="bg-white p-4 rounded shadow w-full max-w-4xl">
        <h2 className="text-lg font-medium mb-2">Bar Chart: Trips per City</h2>
        <Bar data={chartData} />
      </div>
    </div>
  );
}
