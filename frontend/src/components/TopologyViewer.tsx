import React, { useEffect, useRef, useState } from 'react';
import cytoscape, { Core, NodeSingular, EdgeSingular } from 'cytoscape';
import { topologyAPI } from '../services/api';
import './TopologyViewer.css';

interface TopologyViewerProps {
  instanceId?: string;
  hostId?: string;
  networkId?: string;
  maxDepth?: number;
}

interface TopologyData {
  nodes: any[];
  edges: any[];
}

const TopologyViewer: React.FC<TopologyViewerProps> = ({
  instanceId,
  hostId,
  networkId,
  maxDepth = 10,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<Core | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<any>(null);
  const [highlightedPath, setHighlightedPath] = useState<string[]>([]);

  useEffect(() => {
    if (!containerRef.current) return;

    // Initialize Cytoscape
    const cy = cytoscape({
      container: containerRef.current,
      elements: [],
      style: [
        {
          selector: 'node',
          style: {
            'background-color': '#667eea',
            'label': 'data(label)',
            'width': 30,
            'height': 30,
            'text-valign': 'center',
            'text-halign': 'center',
            'color': '#fff',
            'font-size': '10px',
            'text-wrap': 'wrap',
            'text-max-width': '100px',
          },
        },
        {
          selector: 'node[type="VM"]',
          style: {
            'background-color': '#48bb78',
            'shape': 'ellipse',
          },
        },
        {
          selector: 'node[type="PORT"]',
          style: {
            'background-color': '#4299e1',
            'shape': 'diamond',
            'width': 20,
            'height': 20,
          },
        },
        {
          selector: 'node[type="NETWORK"]',
          style: {
            'background-color': '#ed8936',
            'shape': 'rectangle',
          },
        },
        {
          selector: 'node[type="ROUTER"]',
          style: {
            'background-color': '#9f7aea',
            'shape': 'hexagon',
          },
        },
        {
          selector: 'node[type="HOST"]',
          style: {
            'background-color': '#f56565',
            'shape': 'round-rectangle',
          },
        },
        {
          selector: 'node[accessible="false"]',
          style: {
            'background-color': '#a0aec0',
            'border-color': '#718096',
            'border-width': 2,
            'opacity': 0.5,
          },
        },
        {
          selector: 'edge',
          style: {
            'width': 2,
            'line-color': '#cbd5e0',
            'target-arrow-color': '#cbd5e0',
            'target-arrow-shape': 'triangle',
            'curve-style': 'bezier',
          },
        },
        {
          selector: 'edge[type="PHYSICAL"]',
          style: {
            'line-color': '#f56565',
            'target-arrow-color': '#f56565',
            'width': 3,
          },
        },
        {
          selector: 'edge[type="VIRTUAL"]',
          style: {
            'line-color': '#4299e1',
            'target-arrow-color': '#4299e1',
          },
        },
        {
          selector: '.highlighted',
          style: {
            'border-width': 3,
            'border-color': '#fbbf24',
            'z-index': 999,
          },
        },
        {
          selector: 'edge.highlighted',
          style: {
            'line-color': '#fbbf24',
            'target-arrow-color': '#fbbf24',
            'width': 4,
            'z-index': 999,
          },
        },
      ],
      layout: {
        name: 'breadthfirst',
        directed: true,
        spacingFactor: 1.5,
      },
    });

    cyRef.current = cy;

    // Load topology data
    const loadTopology = async () => {
      try {
        setLoading(true);
        setError(null);

        let response;
        if (instanceId) {
          response = await topologyAPI.getInstanceTopology(instanceId, maxDepth);
        } else if (hostId) {
          response = await topologyAPI.getHostTopology(hostId, maxDepth);
        } else if (networkId) {
          response = await topologyAPI.getNetworkTopology(networkId);
        } else {
          setError('No resource ID provided');
          setLoading(false);
          return;
        }

        const data: TopologyData = response.data.data;

        // Transform data for Cytoscape
        const elements: any[] = [];

        // Add nodes
        data.nodes.forEach((node: any) => {
          elements.push({
            data: {
              id: node.id,
              label: node.name || node.id,
              type: node.node_type,
              accessible: node.is_accessible,
            },
          });
        });

        // Add edges
        data.edges.forEach((edge: any) => {
          elements.push({
            data: {
              id: edge.id,
              source: edge.source_node_id,
              target: edge.target_node_id,
              type: edge.edge_type,
            },
          });
        });

        // Update graph
        cy.elements().remove();
        cy.add(elements);

        // Apply layout
        cy.layout({
          name: 'breadthfirst',
          directed: true,
          spacingFactor: 1.5,
        }).run();

        // Node click handler
        cy.on('tap', 'node', (evt) => {
          const node = evt.target;
          setSelectedNode({
            id: node.id(),
            label: node.data('label'),
            type: node.data('type'),
            accessible: node.data('accessible'),
          });

          // Highlight connected edges
          const connectedEdges = node.connectedEdges();
          cy.elements().removeClass('highlighted');
          node.addClass('highlighted');
          connectedEdges.addClass('highlighted');
        });

        // Edge click handler
        cy.on('tap', 'edge', (evt) => {
          const edge = evt.target;
          cy.elements().removeClass('highlighted');
          edge.addClass('highlighted');
          edge.source().addClass('highlighted');
          edge.target().addClass('highlighted');
        });

        // Background click - deselect
        cy.on('tap', (evt) => {
          if (evt.target === cy) {
            cy.elements().removeClass('highlighted');
            setSelectedNode(null);
            setHighlightedPath([]);
          }
        });

        // Zoom and pan controls
        cy.userPanningEnabled(true);
        cy.userZoomingEnabled(true);
        cy.boxSelectionEnabled(true);

        setLoading(false);
      } catch (err: any) {
        setError(err.response?.data?.error || 'Failed to load topology');
        setLoading(false);
      }
    };

    loadTopology();

    // Cleanup
    return () => {
      if (cyRef.current) {
        cyRef.current.destroy();
        cyRef.current = null;
      }
    };
  }, [instanceId, hostId, networkId, maxDepth]);

  if (loading) {
    return (
      <div className="topology-viewer-loading">
        <p>Loading topology...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="topology-viewer-error">
        <p>Error: {error}</p>
      </div>
    );
  }

  return (
    <div className="topology-viewer">
      <div ref={containerRef} className="topology-viewer-container" />
      <div className="topology-legend">
        <div className="legend-item">
          <div className="legend-node vm"></div>
          <span>VM</span>
        </div>
        <div className="legend-item">
          <div className="legend-node port"></div>
          <span>Port</span>
        </div>
        <div className="legend-item">
          <div className="legend-node network"></div>
          <span>Network</span>
        </div>
        <div className="legend-item">
          <div className="legend-node router"></div>
          <span>Router</span>
        </div>
        <div className="legend-item">
          <div className="legend-node host"></div>
          <span>Host</span>
        </div>
        <div className="legend-item">
          <div className="legend-edge virtual"></div>
          <span>Virtual</span>
        </div>
        <div className="legend-item">
          <div className="legend-edge physical"></div>
          <span>Physical</span>
        </div>
      </div>
      {selectedNode && (
        <div className="topology-node-details">
          <h3>Node Details</h3>
          <div className="detail-item">
            <strong>ID:</strong> {selectedNode.id}
          </div>
          <div className="detail-item">
            <strong>Name:</strong> {selectedNode.label}
          </div>
          <div className="detail-item">
            <strong>Type:</strong> {selectedNode.type}
          </div>
          <div className="detail-item">
            <strong>Accessible:</strong> {selectedNode.accessible !== false ? 'Yes' : 'No'}
          </div>
          <button
            className="close-details-btn"
            onClick={() => {
              if (cyRef.current) {
                cyRef.current.elements().removeClass('highlighted');
              }
              setSelectedNode(null);
            }}
          >
            Close
          </button>
        </div>
      )}
      <div className="topology-controls-panel">
        <button
          onClick={() => {
            if (cyRef.current) {
              cyRef.current.fit();
            }
          }}
          className="control-btn"
        >
          Fit
        </button>
        <button
          onClick={() => {
            if (cyRef.current) {
              cyRef.current.zoom(cyRef.current.zoom() * 1.2);
            }
          }}
          className="control-btn"
        >
          Zoom In
        </button>
        <button
          onClick={() => {
            if (cyRef.current) {
              cyRef.current.zoom(cyRef.current.zoom() * 0.8);
            }
          }}
          className="control-btn"
        >
          Zoom Out
        </button>
        <button
          onClick={() => {
            if (cyRef.current) {
              cyRef.current.center();
            }
          }}
          className="control-btn"
        >
          Center
        </button>
      </div>
    </div>
  );
};

export default TopologyViewer;

