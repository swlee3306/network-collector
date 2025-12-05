import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import TopologyViewer from '../components/TopologyViewer';
import { resourceAPI } from '../services/api';
import './Topology.css';

const Topology: React.FC = () => {
  const [resourceType, setResourceType] = useState<'instance' | 'host'>('instance');
  const [resourceId, setResourceId] = useState('');
  const [instances, setInstances] = useState<any[]>([]);
  const [hypervisors, setHypervisors] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [maxDepth, setMaxDepth] = useState(10);

  React.useEffect(() => {
    // Load instances and hypervisors for selection
    const loadResources = async () => {
      try {
        const [instancesRes, hypervisorsRes] = await Promise.all([
          resourceAPI.listInstances(),
          resourceAPI.listHypervisors(),
        ]);
        setInstances(instancesRes.data.data || []);
        setHypervisors(hypervisorsRes.data.data || []);
      } catch (err) {
        console.error('Failed to load resources:', err);
      }
    };

    loadResources();
  }, []);

  return (
    <div className="topology-page">
      <header className="page-header">
        <h1>Network Topology</h1>
        <Link to="/" className="back-link">
          ← Back to Dashboard
        </Link>
      </header>
      <main className="topology-content">
        <div className="topology-controls">
          <div className="control-group">
            <label htmlFor="resource-type">Resource Type:</label>
            <select
              id="resource-type"
              value={resourceType}
              onChange={(e) => {
                setResourceType(e.target.value as 'instance' | 'host');
                setResourceId('');
              }}
            >
              <option value="instance">Instance (VM)</option>
              <option value="host">Host (Hypervisor)</option>
            </select>
          </div>
          <div className="control-group">
            <label htmlFor="resource-id">
              {resourceType === 'instance' ? 'Instance' : 'Hypervisor'}:
            </label>
            <select
              id="resource-id"
              value={resourceId}
              onChange={(e) => setResourceId(e.target.value)}
            >
              <option value="">Select {resourceType === 'instance' ? 'Instance' : 'Hypervisor'}</option>
              {(resourceType === 'instance' ? instances : hypervisors).map((resource) => (
                <option key={resource.id} value={resource.id}>
                  {resource.name} ({resource.id.substring(0, 8)}...)
                </option>
              ))}
            </select>
          </div>
          <div className="control-group">
            <label htmlFor="max-depth">Max Depth:</label>
            <input
              type="number"
              id="max-depth"
              min="1"
              max="20"
              value={maxDepth}
              onChange={(e) => setMaxDepth(parseInt(e.target.value) || 10)}
            />
          </div>
        </div>
        {resourceId && (
          <div className="topology-viewer-wrapper">
            {resourceType === 'instance' ? (
              <TopologyViewer instanceId={resourceId} maxDepth={maxDepth} />
            ) : (
              <TopologyViewer hostId={resourceId} maxDepth={maxDepth} />
            )}
          </div>
        )}
        {!resourceId && (
          <div className="topology-placeholder">
            <p>Please select a {resourceType === 'instance' ? 'instance' : 'hypervisor'} to view topology</p>
          </div>
        )}
      </main>
    </div>
  );
};

export default Topology;

