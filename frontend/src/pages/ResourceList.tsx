import React, { useState, useEffect } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { resourceAPI } from '../services/api';
import './ResourceList.css';

type ResourceType = 'instances' | 'projects' | 'networks' | 'hypervisors' | 'flavors' | 'volumes';

const ResourceList: React.FC = () => {
  const location = useLocation();
  // Extract resource type from pathname (e.g., "/instances" -> "instances")
  const resourceType = location.pathname.replace('/', '') as ResourceType;
  const [resources, setResources] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!resourceType) return;

    const loadResources = async () => {
      try {
        setLoading(true);
        setError(null);

        let response;
        switch (resourceType) {
          case 'instances':
            response = await resourceAPI.listInstances();
            break;
          case 'projects':
            response = await resourceAPI.listProjects();
            break;
          case 'networks':
            response = await resourceAPI.listNetworks();
            break;
          case 'hypervisors':
            response = await resourceAPI.listHypervisors();
            break;
          case 'flavors':
            response = await resourceAPI.listFlavors();
            break;
          case 'volumes':
            response = await resourceAPI.listVolumes();
            break;
          default:
            setError('Invalid resource type');
            return;
        }

        setResources(response.data.data || []);
        setLoading(false);
      } catch (err: any) {
        setError(err.response?.data?.error || 'Failed to load resources');
        setLoading(false);
      }
    };

    loadResources();
  }, [resourceType]);

  const getResourceTitle = () => {
    if (!resourceType) return 'Resources';
    return resourceType.charAt(0).toUpperCase() + resourceType.slice(1);
  };

  const getResourceDetailPath = (id: string) => {
    if (!resourceType) return '#';
    return `/${resourceType}/${id}`;
  };

  const renderResourceRow = (resource: any) => {
    const commonFields = ['id', 'name', 'status', 'openstack_id'];
    
    return (
      <tr key={resource.id} className="resource-row">
        <td>
          <Link to={getResourceDetailPath(resource.id)} className="resource-link">
            {resource.name || resource.id}
          </Link>
        </td>
        <td>{resource.status || 'N/A'}</td>
        <td className="resource-id">{resource.id.substring(0, 8)}...</td>
        {resourceType === 'instances' && (
          <>
            <td>{resource.project_id ? resource.project_id.substring(0, 8) + '...' : 'N/A'}</td>
            <td>{resource.hypervisor_id ? resource.hypervisor_id.substring(0, 8) + '...' : 'N/A'}</td>
          </>
        )}
        {resourceType === 'hypervisors' && (
          <>
            <td>{resource.state || 'N/A'}</td>
            <td>{resource.hostname || 'N/A'}</td>
            <td>{resource.running_vms || 0}</td>
          </>
        )}
        {resourceType === 'networks' && (
          <>
            <td>{resource.project_id ? resource.project_id.substring(0, 8) + '...' : 'N/A'}</td>
            <td>{resource.shared ? 'Yes' : 'No'}</td>
          </>
        )}
        {resourceType === 'projects' && (
          <>
            <td>{resource.description || 'N/A'}</td>
          </>
        )}
        {resourceType === 'flavors' && (
          <>
            <td>{resource.v_cpus || 0}</td>
            <td>{resource.ram || 0} MB</td>
            <td>{resource.disk || 0} GB</td>
          </>
        )}
        {resourceType === 'volumes' && (
          <>
            <td>{resource.size || 0} GB</td>
            <td>{resource.volume_type || 'N/A'}</td>
            <td>{resource.attached_to ? resource.attached_to.substring(0, 8) + '...' : 'N/A'}</td>
          </>
        )}
      </tr>
    );
  };

  const getTableHeaders = () => {
    const baseHeaders = ['Name', 'Status', 'ID'];
    
    switch (resourceType) {
      case 'instances':
        return [...baseHeaders, 'Project', 'Hypervisor'];
      case 'hypervisors':
        return [...baseHeaders, 'State', 'Hostname', 'Running VMs'];
      case 'networks':
        return [...baseHeaders, 'Project', 'Shared'];
      case 'projects':
        return [...baseHeaders, 'Description'];
      case 'flavors':
        return [...baseHeaders, 'vCPUs', 'RAM', 'Disk'];
      case 'volumes':
        return [...baseHeaders, 'Size', 'Type', 'Attached To'];
      default:
        return baseHeaders;
    }
  };

  if (loading) {
    return (
      <div className="resource-list-page">
        <header className="page-header">
          <h1>{getResourceTitle()}</h1>
          <Link to="/" className="back-link">← Back to Dashboard</Link>
        </header>
        <main className="resource-list-content">
          <div className="loading-message">Loading {resourceType}...</div>
        </main>
      </div>
    );
  }

  if (error) {
    return (
      <div className="resource-list-page">
        <header className="page-header">
          <h1>{getResourceTitle()}</h1>
          <Link to="/" className="back-link">← Back to Dashboard</Link>
        </header>
        <main className="resource-list-content">
          <div className="error-message">Error: {error}</div>
        </main>
      </div>
    );
  }

  return (
    <div className="resource-list-page">
      <header className="page-header">
        <h1>{getResourceTitle()}</h1>
        <Link to="/" className="back-link">← Back to Dashboard</Link>
      </header>
      <main className="resource-list-content">
        <div className="resource-list-header">
          <span className="resource-count">Total: {resources.length}</span>
        </div>
        {resources.length === 0 ? (
          <div className="empty-message">No {resourceType} found</div>
        ) : (
          <div className="resource-table-wrapper">
            <table className="resource-table">
              <thead>
                <tr>
                  {getTableHeaders().map((header) => (
                    <th key={header}>{header}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {resources.map(renderResourceRow)}
              </tbody>
            </table>
          </div>
        )}
      </main>
    </div>
  );
};

export default ResourceList;

