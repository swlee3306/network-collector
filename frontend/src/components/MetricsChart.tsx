import React from 'react';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js';
import { Line, Bar } from 'react-chartjs-2';
import './MetricsChart.css';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend,
  Filler
);

interface MetricsChartProps {
  data: any[];
  type?: 'line' | 'bar';
  title?: string;
  xAxisLabel?: string;
  yAxisLabel?: string;
  metricFields?: string[];
}

const MetricsChart: React.FC<MetricsChartProps> = ({
  data,
  type = 'line',
  title,
  xAxisLabel = 'Time',
  yAxisLabel = 'Value',
  metricFields = [],
}) => {
  if (!data || data.length === 0) {
    return (
      <div className="metrics-chart-empty">
        <p>No data available</p>
      </div>
    );
  }

  // Extract timestamps for x-axis
  const labels = data.map((item) => {
    const timestamp = item.timestamp || item.collected_at;
    if (!timestamp) return '';
    const date = new Date(timestamp);
    return date.toLocaleString();
  });

  // Prepare datasets
  const datasets = metricFields.map((field, index) => {
    const colors = [
      { border: 'rgb(102, 126, 234)', background: 'rgba(102, 126, 234, 0.1)' },
      { border: 'rgb(72, 187, 120)', background: 'rgba(72, 187, 120, 0.1)' },
      { border: 'rgb(237, 137, 54)', background: 'rgba(237, 137, 54, 0.1)' },
      { border: 'rgb(245, 101, 101)', background: 'rgba(245, 101, 101, 0.1)' },
      { border: 'rgb(159, 122, 234)', background: 'rgba(159, 122, 234, 0.1)' },
    ];

    const color = colors[index % colors.length];

    return {
      label: field.replace(/_/g, ' ').replace(/\b\w/g, (l) => l.toUpperCase()),
      data: data.map((item) => {
        const value = item[field];
        return value !== null && value !== undefined ? Number(value) : null;
      }),
      borderColor: color.border,
      backgroundColor: type === 'line' ? color.background : color.border,
      fill: type === 'line',
      tension: 0.4,
    };
  });

  // If no metric fields specified, try to auto-detect numeric fields
  if (metricFields.length === 0 && data.length > 0) {
    const firstItem = data[0];
    const numericFields = Object.keys(firstItem).filter((key) => {
      const value = firstItem[key];
      return (
        key !== 'id' &&
        key !== 'timestamp' &&
        key !== 'collected_at' &&
        key !== 'instance_id' &&
        key !== 'network_id' &&
        key !== 'hypervisor_id' &&
        (typeof value === 'number' || (typeof value === 'string' && !isNaN(Number(value))))
      );
    });

    numericFields.forEach((field, index) => {
      const colors = [
        { border: 'rgb(102, 126, 234)', background: 'rgba(102, 126, 234, 0.1)' },
        { border: 'rgb(72, 187, 120)', background: 'rgba(72, 187, 120, 0.1)' },
        { border: 'rgb(237, 137, 54)', background: 'rgba(237, 137, 54, 0.1)' },
        { border: 'rgb(245, 101, 101)', background: 'rgba(245, 101, 101, 0.1)' },
      ];

      const color = colors[index % colors.length];
      datasets.push({
        label: field.replace(/_/g, ' ').replace(/\b\w/g, (l) => l.toUpperCase()),
        data: data.map((item) => {
          const value = item[field];
          return value !== null && value !== undefined ? Number(value) : null;
        }),
        borderColor: color.border,
        backgroundColor: type === 'line' ? color.background : color.border,
        fill: type === 'line',
        tension: 0.4,
      });
    });
  }

  const chartData = {
    labels,
    datasets,
  };

  const options = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: 'top' as const,
      },
      title: {
        display: !!title,
        text: title,
      },
      tooltip: {
        mode: 'index' as const,
        intersect: false,
      },
    },
    scales: {
      x: {
        title: {
          display: true,
          text: xAxisLabel,
        },
      },
      y: {
        title: {
          display: true,
          text: yAxisLabel,
        },
        beginAtZero: true,
      },
    },
    interaction: {
      mode: 'nearest' as const,
      axis: 'x' as const,
      intersect: false,
    },
  };

  return (
    <div className="metrics-chart-container">
      {type === 'line' ? (
        <Line data={chartData} options={options} />
      ) : (
        <Bar data={chartData} options={options} />
      )}
    </div>
  );
};

export default MetricsChart;

