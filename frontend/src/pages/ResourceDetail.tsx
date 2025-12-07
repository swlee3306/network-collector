import React, { useState, useEffect } from 'react';
import { useParams, Link, useNavigate, useLocation } from 'react-router-dom';
import { resourceAPI } from '../services/api';
import './ResourceDetail.css';

type ResourceType = 'instances' | 'projects' | 'networks' | 'hypervisors' | 'flavors' | 'volumes';

const ResourceDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const location = useLocation();
  const navigate = useNavigate();
  
  // Extract resource type from pathname (e.g., "/projects/123" -> "projects")
  const resourceType = location.pathname.split('/')[1] as ResourceType;
  
  const [resource, setResource] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (id && resourceType) {
      loadResource();
    }
  }, [id, resourceType]);

  const loadResource = async () => {
    try {
      setLoading(true);
      setError(null);

      let response;
      switch (resourceType) {
        case 'instances':
          response = await resourceAPI.getInstance(id!);
          break;
        case 'projects':
          response = await resourceAPI.getProject(id!);
          break;
        case 'networks':
          response = await resourceAPI.getNetwork(id!);
          break;
        case 'hypervisors':
          response = await resourceAPI.getHypervisor(id!);
          break;
        case 'flavors':
          response = await resourceAPI.getFlavor(id!);
          break;
        case 'volumes':
          response = await resourceAPI.getVolume(id!);
          break;
        default:
          setError('Invalid resource type');
          setLoading(false);
          return;
      }

      setResource(response.data.data);
      setLoading(false);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load resource');
      setLoading(false);
    }
  };

  const getResourceTitle = () => {
    if (!resourceType) return 'Resource';
    return resourceType.charAt(0).toUpperCase() + resourceType.slice(1, -1); // Remove 's' from plural
  };

  const renderResourceDetails = () => {
    if (!resource) return null;

    const details: React.ReactElement[] = [];

    // Common fields
    details.push(
      <div key="id" className="detail-item">
        <label>ID:</label>
        <span>{resource.id}</span>
      </div>
    );

    if (resource.openstack_id) {
      details.push(
        <div key="openstack_id" className="detail-item">
          <label>OpenStack ID:</label>
          <span>{resource.openstack_id}</span>
        </div>
      );
    }

    if (resource.name !== undefined) {
      details.push(
        <div key="name" className="detail-item">
          <label>Name:</label>
          <span>{resource.name || 'N/A'}</span>
        </div>
      );
    }

    if (resource.status !== undefined) {
      details.push(
        <div key="status" className="detail-item">
          <label>Status:</label>
          <span className={`status-badge status-${resource.status?.toLowerCase()}`}>
            {resource.status || 'N/A'}
          </span>
        </div>
      );
    }

    if (resource.created_at) {
      details.push(
        <div key="created_at" className="detail-item">
          <label>Created At:</label>
          <span>{new Date(resource.created_at).toLocaleString()}</span>
        </div>
      );
    }

    if (resource.updated_at) {
      details.push(
        <div key="updated_at" className="detail-item">
          <label>Updated At:</label>
          <span>{new Date(resource.updated_at).toLocaleString()}</span>
        </div>
      );
    }

    // Resource-specific fields
    if (resourceType === 'projects') {
      if (resource.description) {
        details.push(
          <div key="description" className="detail-item">
            <label>Description:</label>
            <span>{resource.description}</span>
          </div>
        );
      }
    }

    if (resourceType === 'instances') {
      if (resource.project_id) {
        details.push(
          <div key="project_id" className="detail-item">
            <label>Project ID:</label>
            <Link to={`/projects/${resource.project_id}`} className="detail-link">
              {resource.project_id}
            </Link>
          </div>
        );
      }
      if (resource.flavor_id) {
        details.push(
          <div key="flavor_id" className="detail-item">
            <label>Flavor ID:</label>
            <Link to={`/flavors/${resource.flavor_id}`} className="detail-link">
              {resource.flavor_id}
            </Link>
          </div>
        );
      }
      if (resource.hypervisor_id) {
        details.push(
          <div key="hypervisor_id" className="detail-item">
            <label>Hypervisor ID:</label>
            <Link to={`/hypervisors/${resource.hypervisor_id}`} className="detail-link">
              {resource.hypervisor_id}
            </Link>
          </div>
        );
      }
    }

    if (resourceType === 'networks') {
      if (resource.project_id) {
        details.push(
          <div key="project_id" className="detail-item">
            <label>Project ID:</label>
            <Link to={`/projects/${resource.project_id}`} className="detail-link">
              {resource.project_id}
            </Link>
          </div>
        );
      }
      if (resource.shared !== undefined) {
        details.push(
          <div key="shared" className="detail-item">
            <label>Shared:</label>
            <span>{resource.shared ? 'Yes' : 'No'}</span>
          </div>
        );
      }
    }

    if (resourceType === 'hypervisors') {
      if (resource.hostname) {
        details.push(
          <div key="hostname" className="detail-item">
            <label>Hostname:</label>
            <span>{resource.hostname}</span>
          </div>
        );
      }
      if (resource.host_ip) {
        details.push(
          <div key="host_ip" className="detail-item">
            <label>Host IP:</label>
            <span>{resource.host_ip}</span>
          </div>
        );
      }
      if (resource.state) {
        details.push(
          <div key="state" className="detail-item">
            <label>State:</label>
            <span>{resource.state}</span>
          </div>
        );
      }
      if (resource.running_vms !== undefined) {
        details.push(
          <div key="running_vms" className="detail-item">
            <label>Running VMs:</label>
            <span>{resource.running_vms}</span>
          </div>
        );
      }
      if (resource.v_cpus_total !== undefined) {
        details.push(
          <div key="v_cpus" className="detail-item">
            <label>vCPUs:</label>
            <span>{resource.v_cpus_used || 0} / {resource.v_cpus_total}</span>
          </div>
        );
      }
      if (resource.memory_total !== undefined) {
        details.push(
          <div key="memory" className="detail-item">
            <label>Memory:</label>
            <span>{resource.memory_used || 0} MB / {resource.memory_total} MB</span>
          </div>
        );
      }
    }

    if (resourceType === 'flavors') {
      if (resource.v_cpus !== undefined) {
        details.push(
          <div key="v_cpus" className="detail-item">
            <label>vCPUs:</label>
            <span>{resource.v_cpus}</span>
          </div>
        );
      }
      if (resource.ram !== undefined) {
        details.push(
          <div key="ram" className="detail-item">
            <label>RAM:</label>
            <span>{resource.ram} MB</span>
          </div>
        );
      }
      if (resource.disk !== undefined) {
        details.push(
          <div key="disk" className="detail-item">
            <label>Disk:</label>
            <span>{resource.disk} GB</span>
          </div>
        );
      }
      if (resource.is_public !== undefined) {
        details.push(
          <div key="is_public" className="detail-item">
            <label>Public:</label>
            <span>{resource.is_public ? 'Yes' : 'No'}</span>
          </div>
        );
      }
    }

    if (resourceType === 'volumes') {
      if (resource.size !== undefined) {
        details.push(
          <div key="size" className="detail-item">
            <label>Size:</label>
            <span>{resource.size} GB</span>
          </div>
        );
      }
      if (resource.volume_type) {
        details.push(
          <div key="volume_type" className="detail-item">
            <label>Volume Type:</label>
            <span>{resource.volume_type}</span>
          </div>
        );
      }
      if (resource.project_id) {
        details.push(
          <div key="project_id" className="detail-item">
            <label>Project ID:</label>
            <Link to={`/projects/${resource.project_id}`} className="detail-link">
              {resource.project_id}
            </Link>
          </div>
        );
      }
      if (resource.attached_to) {
        details.push(
          <div key="attached_to" className="detail-item">
            <label>Attached To:</label>
            <Link to={`/instances/${resource.attached_to}`} className="detail-link">
              {resource.attached_to}
            </Link>
          </div>
        );
      }
    }

    return details;
  };

  const renderActions = () => {
    if (!resource) return null;

    const actions: JSX.Element[] = [];

    // Topology actions
    if (resourceType === 'instances') {
      actions.push(
        <Link key="topology" to={`/topology?instanceId=${resource.id}`} className="action-btn">
          View Topology
        </Link>
      );
    }
    if (resourceType === 'hypervisors') {
      actions.push(
        <Link key="topology" to={`/topology?hostId=${resource.id}`} className="action-btn">
          View Topology
        </Link>
      );
    }
    if (resourceType === 'networks') {
      actions.push(
        <Link key="topology" to={`/topology?networkId=${resource.id}`} className="action-btn">
          View Topology
        </Link>
      );
    }

    // Metrics actions
    if (resourceType === 'instances' || resourceType === 'networks' || resourceType === 'hypervisors') {
      actions.push(
        <Link 
          key="metrics" 
          to={`/metrics?${resourceType === 'instances' ? 'instanceId' : resourceType === 'networks' ? 'networkId' : 'hypervisorId'}=${resource.id}`} 
          className="action-btn"
        >
          View Metrics
        </Link>
      );
    }

    return actions.length > 0 ? (
      <div className="action-buttons">
        {actions}
      </div>
    ) : null;
  };

  if (loading) {
    return (
      <div className="resource-detail-page">
        <header className="page-header">
          <Link to={`/${resourceType}`} className="back-link">← Back to {getResourceTitle()} List</Link>
        </header>
        <main className="detail-content">
          <div className="loading-message">Loading {getResourceTitle().toLowerCase()} details...</div>
        </main>
      </div>
    );
  }

  if (error || !resource) {
    return (
      <div className="resource-detail-page">
        <header className="page-header">
          <Link to={`/${resourceType}`} className="back-link">← Back to {getResourceTitle()} List</Link>
        </header>
        <main className="detail-content">
          <div className="error-message">
            <p>Error: {error || `${getResourceTitle()} not found`}</p>
            <button onClick={() => navigate(`/${resourceType}`)} className="back-btn">
              Back to {getResourceTitle()} List
            </button>
          </div>
        </main>
      </div>
    );
  }

  return (
    <div className="resource-detail-page">
      <header className="page-header">
        <div>
          <Link to={`/${resourceType}`} className="back-link">← Back to {getResourceTitle()} List</Link>
          <h1>{getResourceTitle()}: {resource.name || resource.id}</h1>
        </div>
      </header>
      <main className="detail-content">
        <div className="detail-section">
          <h2>Basic Information</h2>
          <div className="detail-grid">
            {renderResourceDetails()}
          </div>
        </div>

        {renderActions() && (
          <div className="detail-section">
            <h2>Actions</h2>
            {renderActions()}
          </div>
        )}
      </main>
    </div>
  );
};

export default ResourceDetail;

