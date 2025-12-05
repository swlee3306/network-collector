package topology

import (
	"fmt"

	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
)

// PathFinder finds paths in network topology using graph algorithms
type PathFinder struct {
	repository *storage.Repository
}

// NewPathFinder creates a new path finder
func NewPathFinder(repository *storage.Repository) *PathFinder {
	return &PathFinder{
		repository: repository,
	}
}

// FindPathFromVMToHost finds all paths from a VM to a physical host using BFS
func (p *PathFinder) FindPathFromVMToHost(vmNodeID, hostNodeID string) ([][]string, error) {
	// BFS to find all paths
	type queueItem struct {
		nodeID string
		path   []string
	}

	queue := []queueItem{{nodeID: vmNodeID, path: []string{vmNodeID}}}
	visited := make(map[string]bool)
	var allPaths [][]string

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.nodeID == hostNodeID {
			// Found a path
			allPaths = append(allPaths, current.path)
			continue
		}

		// Mark as visited for this path
		visitedKey := fmt.Sprintf("%s-%d", current.nodeID, len(current.path))
		if visited[visitedKey] {
			continue
		}
		visited[visitedKey] = true

		// Get all edges from current node
		edges, err := p.repository.GetTopologyEdgesByNodeID(current.nodeID)
		if err != nil {
			continue
		}

		// Add neighbors to queue
		for _, edge := range edges {
			nextNodeID := edge.TargetNodeID
			if edge.SourceNodeID == current.nodeID {
				nextNodeID = edge.TargetNodeID
			} else {
				nextNodeID = edge.SourceNodeID
			}

			// Check if already in path (avoid cycles)
			inPath := false
			for _, nodeID := range current.path {
				if nodeID == nextNodeID {
					inPath = true
					break
				}
			}

			if !inPath {
				newPath := make([]string, len(current.path))
				copy(newPath, current.path)
				newPath = append(newPath, nextNodeID)
				queue = append(queue, queueItem{nodeID: nextNodeID, path: newPath})
			}
		}
	}

	return allPaths, nil
}

// GetTopologyGraph gets the complete topology graph for a starting node
func (p *PathFinder) GetTopologyGraph(startNodeID string, maxDepth int) (*TopologyGraph, error) {
	if maxDepth <= 0 {
		maxDepth = 10 // Default max depth
	}

	graph := &TopologyGraph{
		Nodes: make(map[string]*models.TopologyNode),
		Edges: []models.TopologyEdge{},
	}

	// BFS to collect all connected nodes
	type queueItem struct {
		nodeID string
		depth  int
	}

	queue := []queueItem{{nodeID: startNodeID, depth: 0}}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.nodeID] || current.depth > maxDepth {
			continue
		}
		visited[current.nodeID] = true

		// Get node
		node, err := p.repository.GetTopologyNodeByID(current.nodeID)
		if err != nil {
			continue
		}
		graph.Nodes[current.nodeID] = node

		// Get edges
		edges, err := p.repository.GetTopologyEdgesByNodeID(current.nodeID)
		if err != nil {
			continue
		}

		for _, edge := range edges {
			// Add edge if not already added
			edgeExists := false
			for _, existingEdge := range graph.Edges {
				if existingEdge.ID == edge.ID {
					edgeExists = true
					break
				}
			}
			if !edgeExists {
				graph.Edges = append(graph.Edges, edge)
			}

			// Add target node to queue
			targetNodeID := edge.TargetNodeID
			if edge.SourceNodeID == current.nodeID {
				targetNodeID = edge.TargetNodeID
			} else {
				targetNodeID = edge.SourceNodeID
			}

			if !visited[targetNodeID] && current.depth < maxDepth {
				queue = append(queue, queueItem{nodeID: targetNodeID, depth: current.depth + 1})
			}
		}
	}

	return graph, nil
}

// TopologyGraph represents a topology graph structure
type TopologyGraph struct {
	Nodes map[string]*models.TopologyNode
	Edges []models.TopologyEdge
}

