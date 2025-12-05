import React, { useState, useEffect } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { resourceAPI } from '../services/api';
import './ResourceDetail.css';

const InstanceDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [instance, setInstance] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (id) {
      loadInstance();
    }
  }, [id]);

  const loadInstance = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await resourceAPI.getInstance(id!);
      setInstance(response.data.data);
      setLoading(false);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load instance');
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="resource-detail-page">
        <div className="loading-message">Loading instance details...</div>
      </div>
    );
  }

  if (error || !instance) {
    return (
      <div className="resource-detail-page">
        <div className="error-message">
          <p>Error: {error || 'Instance not found'}</p>
          <button onClick={() => navigate('/')} className="back-btn">
            Back to Dashboard
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="resource-detail-page">
      <header className="page-header">
        <div>
          <Link to="/" className="back-link">← Back to Dashboard</Link>
          <h1>Instance: {instance.name || instance.id}</h1>
        </div>
      </header>
      <main className="detail-content">
        <div className="detail-section">
          <h2>Basic Information</h2>
          <div className="detail-grid">
            <div className="detail-item">
              <label>ID:</label>
              <span>{instance.id}</span>
            </div>
            <div className="detail-item">
              <label>OpenStack ID:</label>
              <span>{instance.openstack_id}</span>
            </div>
            <div className="detail-item">
              <label>Name:</label>
              <span>{instance.name || 'N/A'}</span>
            </div>
            <div className="detail-item">
              <label>Status:</label>
              <span className={`status-badge status-${instance.status?.toLowerCase()}`}>
                {instance.status}
              </span>
            </div>
            <div className="detail-item">
              <label>Created At:</label>
              <span>{instance.created_at ? new Date(instance.created_at).toLocaleString() : 'N/A'}</span>
            </div>
            <div className="detail-item">
              <label>Updated At:</label>
              <span>{instance.updated_at ? new Date(instance.updated_at).toLocaleString() : 'N/A'}</span>
            </div>
          </div>
        </div>

        <div className="detail-section">
          <h2>Relationships</h2>
          <div className="detail-grid">
            <div className="detail-item">
              <label>Project ID:</label>
              <Link to={`/projects/${instance.project_id}`} className="detail-link">
                {instance.project_id}
              </Link>
            </div>
            <div className="detail-item">
              <label>Flavor ID:</label>
              <Link to={`/flavors/${instance.flavor_id}`} className="detail-link">
                {instance.flavor_id}
              </Link>
            </div>
            <div className="detail-item">
              <label>Hypervisor ID:</label>
              {instance.hypervisor_id ? (
                <Link to={`/hypervisors/${instance.hypervisor_id}`} className="detail-link">
                  {instance.hypervisor_id}
                </Link>
              ) : (
                <span>N/A</span>
              )}
            </div>
          </div>
        </div>

        <div className="detail-section">
          <h2>Actions</h2>
          <div className="action-buttons">
            <Link to={`/topology?instanceId=${instance.id}`} className="action-btn">
              View Topology
            </Link>
            <Link to={`/metrics?instanceId=${instance.id}`} className="action-btn">
              View Metrics
            </Link>
          </div>
        </div>
      </main>
    </div>
  );
};

export default InstanceDetail;

