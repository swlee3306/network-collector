import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { resourceAPI } from '../services/api';
import MetricsChart from '../components/MetricsChart';
import './ProjectComparison.css';

interface ProjectComparison {
  project_id: string;
  project_name: string;
  instances: number;
  networks: number;
  volumes: number;
  active_instances: number;
}

const ProjectComparisonPage: React.FC = () => {
  const [projects, setProjects] = useState<any[]>([]);
  const [selectedProjects, setSelectedProjects] = useState<string[]>([]);
  const [comparisonData, setComparisonData] = useState<ProjectComparison[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadProjects();
  }, []);

  useEffect(() => {
    if (selectedProjects.length > 0) {
      loadComparison();
    } else {
      setComparisonData([]);
    }
  }, [selectedProjects]);

  const loadProjects = async () => {
    try {
      const response = await resourceAPI.listProjects();
      setProjects(response.data.data || []);
    } catch (err) {
      console.error('Failed to load projects:', err);
    }
  };

  const loadComparison = async () => {
    try {
      setLoading(true);
      setError(null);

      const projectIds = selectedProjects.join(',');
      const response = await fetch(
        `${process.env.REACT_APP_API_URL || 'http://localhost:8080'}/api/v1/projects/compare?project_ids=${projectIds}`,
        {
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('auth_token')}`,
          },
        }
      );

      if (!response.ok) {
        throw new Error('Failed to load comparison data');
      }

      const data = await response.json();
      setComparisonData(data.data || []);
      setLoading(false);
    } catch (err: any) {
      setError(err.message || 'Failed to load comparison data');
      setLoading(false);
    }
  };

  const handleProjectToggle = (projectId: string) => {
    setSelectedProjects((prev) => {
      if (prev.includes(projectId)) {
        return prev.filter((id) => id !== projectId);
      } else {
        return [...prev, projectId];
      }
    });
  };

  // Prepare chart data for comparison
  const chartData = comparisonData.map((project) => ({
    timestamp: project.project_name,
    instances: project.instances,
    active_instances: project.active_instances,
    networks: project.networks,
    volumes: project.volumes,
  }));

  return (
    <div className="project-comparison-page">
      <header className="page-header">
        <h1>Project Comparison</h1>
        <Link to="/" className="back-link">
          ← Back to Dashboard
        </Link>
      </header>
      <main className="comparison-content">
        <div className="comparison-controls">
          <h2>Select Projects to Compare</h2>
          <div className="project-selection">
            {projects.map((project) => (
              <label key={project.id} className="project-checkbox">
                <input
                  type="checkbox"
                  checked={selectedProjects.includes(project.id)}
                  onChange={() => handleProjectToggle(project.id)}
                />
                <span>{project.name || project.id}</span>
              </label>
            ))}
          </div>
        </div>

        {loading && <div className="loading-message">Loading comparison data...</div>}
        {error && <div className="error-message">Error: {error}</div>}

        {!loading && !error && comparisonData.length > 0 && (
          <>
            <div className="comparison-table-wrapper">
              <table className="comparison-table">
                <thead>
                  <tr>
                    <th>Project</th>
                    <th>Instances</th>
                    <th>Active Instances</th>
                    <th>Networks</th>
                    <th>Volumes</th>
                  </tr>
                </thead>
                <tbody>
                  {comparisonData.map((project) => (
                    <tr key={project.project_id}>
                      <td>
                        <Link to={`/projects/${project.project_id}`} className="project-link">
                          {project.project_name || project.project_id}
                        </Link>
                      </td>
                      <td>{project.instances}</td>
                      <td>{project.active_instances}</td>
                      <td>{project.networks}</td>
                      <td>{project.volumes}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="comparison-charts">
              <div className="chart-section">
                <h3>Resource Comparison</h3>
                <MetricsChart
                  data={chartData}
                  type="bar"
                  title="Resource Count by Project"
                  xAxisLabel="Project"
                  yAxisLabel="Count"
                  metricFields={['instances', 'networks', 'volumes']}
                />
              </div>
            </div>
          </>
        )}

        {!loading && !error && selectedProjects.length === 0 && (
          <div className="comparison-placeholder">
            <p>Select at least one project to compare</p>
          </div>
        )}
      </main>
    </div>
  );
};

export default ProjectComparisonPage;

