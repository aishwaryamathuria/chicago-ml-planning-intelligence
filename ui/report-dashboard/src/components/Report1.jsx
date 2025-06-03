import React from "react";
import { Bar } from "react-chartjs-2";
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend,
} from "chart.js";

ChartJS.register(
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend
);

const data = {
  labels: ["Jan", "Feb", "Mar", "Apr"],
  datasets: [
    {
      label: "Sales",
      data: [120, 190, 300, 500],
      backgroundColor: "rgba(59, 130, 246, 0.6)",
    },
  ],
};

const options = {
  responsive: true,
  plugins: {
    legend: { position: "top" },
    title: { display: true, text: "Monthly Sales" },
  },
};

const Report1 = () => (
  <div>
    <h2 className="text-2xl font-semibold mb-4">Report 1: Sales Chart</h2>
    <div className="bg-white p-4 rounded shadow">
      <Bar data={data} options={options} />
    </div>
  </div>
);

export default Report1;
