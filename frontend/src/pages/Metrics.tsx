import React, { useState, useEffect, useCallback } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { metricsAPI, resourceAPI } from '../services/api';
import MetricsChart from '../components/MetricsChart';
import './Metrics.css';

const Metrics: React.FC = () => {
  const [searchParams] = useSearchParams();
  const [resourceType, setResourceType] = useState<'instance' | 'network' | 'hypervisor'>('instance');
  const [resourceId, setResourceId] = useState('');
  const [resources, setResources] = useState<any[]>([]);
  const [metrics, setMetrics] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [timeRange, setTimeRange] = useState<{ start: string; end: string }>({
    start: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
    end: new Date().toISOString(),
  });

  useEffect(() => {
    // Check URL params
    const instanceId = searchParams.get('instanceId');
    const networkId = searchParams.get('networkId');
    const hypervisorId = searchParams.get('hypervisorId');

    if (instanceId) {
      setResourceType('instance');
      setResourceId(instanceId);
    } else if (networkId) {
      setResourceType('network');
      setResourceId(networkId);
    } else if (hypervisorId) {
      setResourceType('hypervisor');
      setResourceId(hypervisorId);
    }
  }, [searchParams]);

  const loadResources = useCallback(async () => {
    try {
      let response;
      if (resourceType === 'instance') {
        response = await resourceAPI.listInstances();
      } else if (resourceType === 'network') {
        response = await resourceAPI.listNetworks();
      } else {
        response = await resourceAPI.listHypervisors();
      }
      setResources(response.data.data || []);
    } catch (err) {
      console.error('Failed to load resources:', err);
    }
  }, [resourceType]);

  const loadMetrics = useCallback(async () => {
    if (!resourceId) return;
    
    try {
      setLoading(true);
      setError(null);

      let response;
      if (resourceType === 'instance') {
        response = await metricsAPI.getInstanceMetrics(resourceId, timeRange.start, timeRange.end);
      } else if (resourceType === 'network') {
        response = await metricsAPI.getNetworkMetrics(resourceId, timeRange.start, timeRange.end);
      } else {
        response = await metricsAPI.getHypervisorMetrics(resourceId, timeRange.start, timeRange.end);
      }

      setMetrics(response.data.data || []);
      setLoading(false);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load metrics');
      setLoading(false);
    }
  }, [resourceId, resourceType, timeRange.start, timeRange.end]);

  useEffect(() => {
    loadResources();
  }, [loadResources]);

  useEffect(() => {
    if (resourceId) {
      loadMetrics();
    }
  }, [resourceId, loadMetrics]);

  return (
    <div className="metrics-page">
      <header className="page-header">
        <h1>Metrics</h1>
        <Link to="/" className="back-link">
          ← Back to Dashboard
        </Link>
      </header>
      <main className="metrics-content">
        <div className="metrics-controls">
          <div className="control-group">
            <label htmlFor="resource-type">Resource Type:</label>
            <select
              id="resource-type"
              value={resourceType}
              onChange={(e) => {
                setResourceType(e.target.value as 'instance' | 'network' | 'hypervisor');
                setResourceId('');
              }}
            >
              <option value="instance">Instance</option>
              <option value="network">Network</option>
              <option value="hypervisor">Hypervisor</option>
            </select>
          </div>
          <div className="control-group">
            <label htmlFor="resource-id">Resource:</label>
            <select
              id="resource-id"
              value={resourceId}
              onChange={(e) => setResourceId(e.target.value)}
            >
              <option value="">Select {resourceType}</option>
              {resources.map((resource) => (
                <option key={resource.id} value={resource.id}>
                  {resource.name || resource.id} ({resource.id.substring(0, 8)}...)
                </option>
              ))}
            </select>
          </div>
          <div className="control-group">
            <label htmlFor="start-time">Start Time:</label>
            <input
              type="datetime-local"
              id="start-time"
              value={timeRange.start.substring(0, 16)}
              onChange={(e) => setTimeRange({ ...timeRange, start: new Date(e.target.value).toISOString() })}
            />
          </div>
          <div className="control-group">
            <label htmlFor="end-time">End Time:</label>
            <input
              type="datetime-local"
              id="end-time"
              value={timeRange.end.substring(0, 16)}
              onChange={(e) => setTimeRange({ ...timeRange, end: new Date(e.target.value).toISOString() })}
            />
          </div>
        </div>

        {loading && <div className="loading-message">Loading metrics...</div>}
        {error && <div className="error-message">Error: {error}</div>}

        {!loading && !error && resourceId && (
          <div className="metrics-chart-wrapper">
            <MetricsChart
              data={metrics}
              type="line"
              title={`${resourceType.charAt(0).toUpperCase() + resourceType.slice(1)} Metrics`}
              xAxisLabel="Time"
              yAxisLabel="Value"
            />
          </div>
        )}

        {!resourceId && (
          <div className="metrics-placeholder">
            <p>Please select a {resourceType} to view metrics</p>
          </div>
        )}
      </main>
    </div>
  );
};

export default Metrics;

