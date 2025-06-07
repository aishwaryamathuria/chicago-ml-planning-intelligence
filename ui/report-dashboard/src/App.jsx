import React, { useState } from "react";
import Report1 from "./components/Report1";
import Report2 from "./components/Report2";
import Report3 from "./components/Report3";
import Report4 from "./components/Report4";
import Report5 from "./components/Report5";
import Report6 from "./components/Report6";
import Report7 from "./components/Report7";

const reportHierarchy = {
  category1: [
    {
      id: "category1/report1",
      label: "Report 1",
      categoryLabel: "Communities and Businesses Welfare",
      reportLabel:
        "Report 1: Send alerts to taxi drivers to avoid super spreaders",
      reportComponent: <Report1 />,
    },
    {
      id: "category1/report2",
      label: "Report 2",
      categoryLabel: "Communities and Businesses Welfare",
      reportLabel:
        "Report 2: COVID-19 level forecast (taxi trip data and daily COVID-19 case counts)",
      reportComponent: <Report2 />,
    },
    {
      id: "category1/report3",
      label: "Report 3",
      categoryLabel: "Communities and Businesses Welfare",
      reportLabel: "Report 3: Airport taxi trip and COVID-19 spread analysis.",
      reportComponent: <Report3 />,
    },
    {
      id: "category1/report4",
      label: "Report 4",
      categoryLabel: "Communities and Businesses Welfare",
      reportLabel: "Report 4: Taxi trips in high CCVI neighborhoods",
      reportComponent: <Report4 />,
    },
  ],
  category2: [
    {
      id: "category2/report1",
      label: "Report 1",
      categoryLabel:
        "Community Investments, Business Incentives, Forecasting and Strategic Planning",
      reportLabel: "Report 1: Taxi trips forecast for construction planning.",
      reportComponent: <Report5 />,
    },
    {
      id: "category2/report2",
      label: "Report 2",
      categoryLabel:
        "Community Investments, Business Incentives, Forecasting and Strategic Planning",
      reportLabel:
        "Report 2: Neighborhoods with highest unemployment and poverty rate.",
      reportComponent: <Report6 />,
    },
    {
      id: "category2/report3",
      label: "Report 3",
      categoryLabel:
        "Community Investments, Business Incentives, Forecasting and Strategic Planning",
      reportLabel: "Report 3: Low interest eligibility business candidates.",
      reportComponent: <Report7 />,
    },
  ],
};

export default function App() {
  const [activeReport, setActiveReport] = useState({
    category: "category1",
    report: 0,
  });

  const getBreadcrumb = () => {
    const { category, report } = activeReport;
    const reportObj = reportHierarchy[category]?.[report];
    if (!reportObj) return "";
    return `${reportObj.categoryLabel} / ${reportObj.reportLabel}`;
  };

  const renderReportContent = () => {
    const { category, report } = activeReport;
    const reportObj = reportHierarchy[category]?.[report];

    if (!reportObj) {
      return <p>No report found.</p>;
    }

    return (
      <div>
        <div className="pb-2 text-sm text-gray-500 mb-2">{getBreadcrumb()}</div>
        <div>{reportObj.reportComponent}</div>
      </div>
    );
  };

  return (
    <div className="flex h-screen">
      <nav className="w-64 bg-gray-800 text-gray-100 flex flex-col">
        <div className="text-xl font-bold p-4 border-b border-gray-700">
          Reports
        </div>
        <div className="flex-grow overflow-y-auto px-2 py-2 space-y-4">
          {Object.entries(reportHierarchy).map(([category, reports]) => (
            <div key={category}>
              <h3 className="text-sm uppercase text-gray-400 pl-2 mb-1">
                {category.replace("category", "Category ")}
              </h3>
              <ul className="space-y-1">
                {reports.map((reportObj, index) => (
                  <li
                    key={reportObj.id}
                    onClick={() =>
                      setActiveReport({ category: category, report: index })
                    }
                    className={`cursor-pointer px-3 py-2 rounded-md hover:bg-gray-700 ${
                      activeReport.category === category &&
                      activeReport.report === index
                        ? "bg-gray-700"
                        : ""
                    }`}
                  >
                    {reportObj.label}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </nav>

      <div className="flex-1 flex flex-col">
        <header className="h-14 bg-white border-b border-gray-300 flex items-center px-6 shadow-sm">
          <img
            src="/src/assets/chicago-96.png"
            width="40px"
            className="h-auto"
            alt="Logo"
          />
          <div className="h-full text-2xl font-bold text-gray-800 flex items-center justify-center pl-[10px]">
            Chicago Business Intelligence for Strategic Planning
          </div>
        </header>
        <main className="flex-1 pt-4 pb-4 pl-6 pr-6 bg-gray-50 overflow-auto">
          {renderReportContent()}
        </main>
      </div>
    </div>
  );
}
