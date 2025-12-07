package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/network-collector/backend/internal/services/topology"
)

// GetHostTopology gets network topology for a specific hypervisor/host
func GetHostTopology(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		log.Printf("GetHostTopology called with ID: %s", id)

		// Get hypervisor
		hypervisor, err := repo.GetHypervisorByID(id)
		if err != nil {
			log.Printf("Hypervisor not found for ID: %s, error: %v", id, err)
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Hypervisor not found",
			})
			return
		}
		log.Printf("Found hypervisor: %s (ID: %s)", hypervisor.Hostname, hypervisor.ID)

		// Get topology node for hypervisor
		hostNode, err := repo.GetTopologyNodeByHypervisorID(hypervisor.ID)
		if err != nil {
			log.Printf("Topology node not found for hypervisor ID: %s, error: %v", hypervisor.ID, err)
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Topology not found for this hypervisor",
			})
			return
		}
		log.Printf("Found topology node: %s (ID: %s)", hostNode.Name, hostNode.ID)

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

		graph, err := pathFinder.GetTopologyGraph(hostNode.ID, maxDepth)
		if err != nil {
			log.Printf("Failed to get topology graph for node %s: %v", hostNode.ID, err)
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

		log.Printf("Returning topology graph: %d nodes, %d edges", len(nodes), len(graph.Edges))
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"nodes": nodes,
				"edges": graph.Edges,
			},
		})
	}
}

