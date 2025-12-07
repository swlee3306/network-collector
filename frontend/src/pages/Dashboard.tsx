import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { authAPI, resourceAPI } from '../services/api';
import { useEventSource } from '../hooks/useEventSource';
import './Dashboard.css';

interface ResourceCounts {
  instances: number;
  projects: number;
  networks: number;
  hypervisors: number;
  flavors: number;
  volumes: number;
}

interface ResourceStatus {
  instances: { active: number; total: number };
  networks: { active: number; total: number };
  hypervisors: { up: number; total: number };
}

const Dashboard: React.FC = () => {
  const [counts, setCounts] = useState<ResourceCounts>({
    instances: 0,
    projects: 0,
    networks: 0,
    hypervisors: 0,
    flavors: 0,
    volumes: 0,
  });
  const [status, setStatus] = useState<ResourceStatus>({
    instances: { active: 0, total: 0 },
    networks: { active: 0, total: 0 },
    hypervisors: { up: 0, total: 0 },
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const { event, connected } = useEventSource();

  useEffect(() => {
    loadDashboardData();
  }, []);

  // Reload data when collection end event is received
  useEffect(() => {
    if (event && event.type === 'collection.end') {
      console.log('Collection completed, reloading dashboard data...');
      loadDashboardData();
    }
  }, [event]);

  const loadDashboardData = async () => {
    try {
      setLoading(true);
      setError(null);

      const [instancesRes, projectsRes, networksRes, hypervisorsRes, flavorsRes, volumesRes] = await Promise.all([
        resourceAPI.listInstances(),
        resourceAPI.listProjects(),
        resourceAPI.listNetworks(),
        resourceAPI.listHypervisors(),
        resourceAPI.listFlavors(),
        resourceAPI.listVolumes(),
      ]);

      const instances = instancesRes.data.data || [];
      const projects = projectsRes.data.data || [];
      const networks = networksRes.data.data || [];
      const hypervisors = hypervisorsRes.data.data || [];
      const flavors = flavorsRes.data.data || [];
      const volumes = volumesRes.data.data || [];

      setCounts({
        instances: instances.length,
        projects: projects.length,
        networks: networks.length,
        hypervisors: hypervisors.length,
        flavors: flavors.length,
        volumes: volumes.length,
      });

      // Calculate status
      const activeInstances = instances.filter((i: any) => i.status === 'ACTIVE').length;
      const activeNetworks = networks.filter((n: any) => n.status === 'ACTIVE').length;
      // Hypervisor state field is "state" not "status" - state can be "up" or "down"
      const upHypervisors = hypervisors.filter((h: any) => h.state === 'up').length;

      setStatus({
        instances: { active: activeInstances, total: instances.length },
        networks: { active: activeNetworks, total: networks.length },
        hypervisors: { up: upHypervisors, total: hypervisors.length },
      });

      setLoading(false);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load dashboard data');
      setLoading(false);
    }
  };

  const handleLogout = () => {
    authAPI.logout();
    window.location.href = '/login';
  };

  if (loading) {
    return (
      <div className="dashboard">
        <header className="dashboard-header">
          <h1>OpenStack Monitoring Dashboard</h1>
          <button onClick={handleLogout} className="logout-btn">
            Logout
          </button>
        </header>
        <nav className="dashboard-nav">
          <Link to="/" className="nav-link active">
            Dashboard
          </Link>
          <Link to="/topology" className="nav-link">
            Topology
          </Link>
          <Link to="/metrics" className="nav-link">
            Metrics
          </Link>
        </nav>
        <main className="dashboard-content">
          <div className="loading-message">Loading dashboard data...</div>
        </main>
      </div>
    );
  }

  if (error) {
    return (
      <div className="dashboard">
        <header className="dashboard-header">
          <h1>OpenStack Monitoring Dashboard</h1>
          <button onClick={handleLogout} className="logout-btn">
            Logout
          </button>
        </header>
        <nav className="dashboard-nav">
          <Link to="/" className="nav-link active">
            Dashboard
          </Link>
          <Link to="/topology" className="nav-link">
            Topology
          </Link>
          <Link to="/metrics" className="nav-link">
            Metrics
          </Link>
        </nav>
        <main className="dashboard-content">
          <div className="error-message">
            <p>Error: {error}</p>
            <button onClick={loadDashboardData} className="retry-btn">
              Retry
            </button>
          </div>
        </main>
      </div>
    );
  }

  return (
    <div className="dashboard">
      <header className="dashboard-header">
        <div>
          <h1>OpenStack Monitoring Dashboard</h1>
          <div className="connection-status">
            <span className={`status-indicator ${connected ? 'connected' : 'disconnected'}`}></span>
            <span>{connected ? 'Connected' : 'Disconnected'}</span>
          </div>
        </div>
        <button onClick={handleLogout} className="logout-btn">
          Logout
        </button>
      </header>
      <nav className="dashboard-nav">
        <Link to="/" className="nav-link active">
          Dashboard
        </Link>
        <Link to="/topology" className="nav-link">
          Topology
        </Link>
        <Link to="/metrics" className="nav-link">
          Metrics
        </Link>
      </nav>
      <main className="dashboard-content">
        <div className="dashboard-grid">
          <div className="dashboard-card resource-card">
            <div className="card-header">
              <h2>Instances</h2>
              <span className="card-count">{counts.instances}</span>
            </div>
            <div className="card-status">
              <span className="status-label">Active:</span>
              <span className="status-value">{status.instances.active} / {status.instances.total}</span>
            </div>
            <Link to="/instances" className="card-link">
              View Details →
            </Link>
          </div>

          <div className="dashboard-card resource-card">
            <div className="card-header">
              <h2>Projects</h2>
              <span className="card-count">{counts.projects}</span>
            </div>
            <p className="card-description">OpenStack projects</p>
            <Link to="/projects" className="card-link">
              View Details →
            </Link>
          </div>

          <div className="dashboard-card resource-card">
            <div className="card-header">
              <h2>Networks</h2>
              <span className="card-count">{counts.networks}</span>
            </div>
            <div className="card-status">
              <span className="status-label">Active:</span>
              <span className="status-value">{status.networks.active} / {status.networks.total}</span>
            </div>
            <Link to="/networks" className="card-link">
              View Details →
            </Link>
          </div>

          <div className="dashboard-card resource-card">
            <div className="card-header">
              <h2>Hypervisors</h2>
              <span className="card-count">{counts.hypervisors}</span>
            </div>
            <div className="card-status">
              <span className="status-label">Up:</span>
              <span className="status-value">{status.hypervisors.up} / {status.hypervisors.total}</span>
            </div>
            <Link to="/hypervisors" className="card-link">
              View Details →
            </Link>
          </div>

          <div className="dashboard-card resource-card">
            <div className="card-header">
              <h2>Flavors</h2>
              <span className="card-count">{counts.flavors}</span>
            </div>
            <p className="card-description">VM instance flavors</p>
            <Link to="/flavors" className="card-link">
              View Details →
            </Link>
          </div>

          <div className="dashboard-card resource-card">
            <div className="card-header">
              <h2>Volumes</h2>
              <span className="card-count">{counts.volumes}</span>
            </div>
            <p className="card-description">Block storage volumes</p>
            <Link to="/volumes" className="card-link">
              View Details →
            </Link>
          </div>

          <div className="dashboard-card action-card">
            <h2>Topology</h2>
            <p>Visualize network topology</p>
            <Link to="/topology" className="card-link">
              View Topology →
            </Link>
          </div>

          <div className="dashboard-card action-card">
            <h2>Metrics</h2>
            <p>View historical metrics</p>
            <Link to="/metrics" className="card-link">
              View Metrics →
            </Link>
          </div>

          <div className="dashboard-card action-card">
            <h2>Project Comparison</h2>
            <p>Compare resource usage across projects</p>
            <Link to="/projects/compare" className="card-link">
              Compare Projects →
            </Link>
          </div>
        </div>
      </main>
    </div>
  );
};

export default Dashboard;

