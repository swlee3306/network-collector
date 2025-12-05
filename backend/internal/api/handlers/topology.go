package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/network-collector/backend/internal/services/topology"
)

// GetInstanceTopology gets network topology for a specific instance
func GetInstanceTopology(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Get instance
		instance, err := repo.GetInstanceByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Instance not found",
			})
			return
		}

		// Get topology nodes starting from instance
		vmNode, err := repo.GetTopologyNodeByInstanceID(instance.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Topology not found for this instance",
			})
			return
		}

		// Use PathFinder to get complete topology graph
		pathFinder := topology.NewPathFinder(repo)
		maxDepth := 10
		if depthStr := c.Query("max_depth"); depthStr != "" {
			if depth, err := strconv.Atoi(depthStr); err == nil && depth > 0 && depth <= 100 {
				maxDepth = depth
			} else {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "max_depth must be a positive integer between 1 and 100",
				})
				return
			}
		}

		graph, err := pathFinder.GetTopologyGraph(vmNode.ID, maxDepth)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get topology graph",
			})
			return
		}

		// Convert nodes map to slice
		var nodes []*models.TopologyNode
		for _, node := range graph.Nodes {
			nodes = append(nodes, node)
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"nodes": nodes,
				"edges": graph.Edges,
			},
		})
	}
}

// GetNetworkTopology gets network topology for a specific network
func GetNetworkTopology(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Get network
		network, err := repo.GetNetworkByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Network not found",
			})
			return
		}

		// Get topology node for network
		networkNode, err := repo.GetTopologyNodeByNetworkID(network.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Topology not found for this network",
			})
			return
		}

		// Get all edges connected to this node
		edges, err := repo.GetTopologyEdgesByNodeID(networkNode.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get topology edges",
			})
			return
		}

		// Get all connected nodes
		nodeIDs := make(map[string]bool)
		for _, edge := range edges {
			nodeIDs[edge.SourceNodeID] = true
			nodeIDs[edge.TargetNodeID] = true
		}

		var nodeIDList []string
		for id := range nodeIDs {
			nodeIDList = append(nodeIDList, id)
		}

		nodes, err := repo.GetTopologyNodesByIDs(nodeIDList)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get topology nodes",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"nodes": nodes,
				"edges": edges,
			},
		})
	}
}

