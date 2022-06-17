package graph

import (
	"fmt"
	"github.com/Masterminds/semver"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/network"
	"gonum.org/v1/gonum/graph/traverse"
	"time"
)

// GetTransitiveDependenciesNode returns the specified node and its dependencies
func GetTransitiveDependenciesNode(g *DirectedGraph, nodeMap map[int64]NodeInfo, hashMap map[uint64]int64, stringId string) *[]NodeInfo {
	var nodeId int64
	result := make([]NodeInfo, 0, len(nodeMap)/2)
	if id, ok := findNode(hashMap, nodeMap, stringId); ok {
		nodeId = id
	} else {
		return &result // This function is a no-op if we don't have a correct string id
	}

	w := traverse.BreadthFirst{
		Visit: func(n graph.Node) {
			result = append(result, nodeMap[n.ID()])
		},
	}

	_ = w.Walk(g, g.Node(nodeId), nil)
	return &result
}

// GetLatestTransitiveDependenciesNode gets the latest dependencies matching the node's version constraints.
// If interested in finding this within a specific timeframe, use FilterNoTraversal first
func GetLatestTransitiveDependenciesNode(g *DirectedGraph, nodeMap map[int64]NodeInfo, hashMap map[uint64]int64, stringId string) *[]NodeInfo {
	var rootNode NodeInfo
	allDeps := GetTransitiveDependenciesNode(g, nodeMap, hashMap, stringId)
	result := make([]NodeInfo, 0, len(*allDeps)/2)
	if len(*allDeps) > 1 {
		rootNode = (*allDeps)[0]
	} else {
		return &result // No-op if no dependencies were found for whatever reason
	}

	newestPackageVersion := make(map[uint32]NodeInfo, len(*allDeps)/2)

	result = append(result, rootNode)

	// This for loop does the actual filtering
	for _, current := range *allDeps {

		if current.id == rootNode.id {
			continue
		}

		hash := hashPackageName(current.Name)
		currentDate, err := time.Parse(time.RFC3339, current.Timestamp)
		if err != nil {
			continue
		}
		if latest, ok := newestPackageVersion[hash]; ok {
			latestDate, err := time.Parse(time.RFC3339, latest.Timestamp)
			if err != nil {
				fmt.Println(err)
				continue
			} else if currentDate.After(latestDate) { // If the key exists, and current date is later than the one stored
				newestPackageVersion[hash] = current // Set to the current package
			} else if currentDate.Equal(latestDate) { // If the dates are somehow equal, compare version numbers
				currentVersion, _ := semver.NewVersion(current.Version)
				latestVersion, _ := semver.NewVersion(latest.Version)

				if currentVersion.GreaterThan(latestVersion) {
					newestPackageVersion[hash] = current
				}
			}
		} else { // If the key doesn't exist yet
			newestPackageVersion[hash] = current
		}
	}

	for _, v := range newestPackageVersion { // Add all latest package versions to the result
		result = append(result, v)
	}

	return &result
}

// PageRank uses the sparse page rank algorithm to find the Page ranks of all nodes
func PageRank(graph *DirectedGraph) map[int64]float64 {
	pr := network.PageRankSparse(graph, 0.85, 0.001)
	return pr
}

func Betweenness(graph *DirectedGraph) map[int64]float64 {
	betweenness := network.Betweenness(graph)
	return betweenness
}
