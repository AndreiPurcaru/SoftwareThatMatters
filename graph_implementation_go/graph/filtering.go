package graph

import (
	"github.com/Masterminds/semver"
	"time"
)

// FilterNoTraversal filters the nodes that have timestamps between beginTime and endTime. WARNING: This method is destructive,
// meaning that after running it, the nodes and their associated edges that do not correspond to the filter WILL BE REMOVED
// from the graph
func FilterNoTraversal(g *DirectedGraph, nodeMap map[int64]NodeInfo, beginTime, endTime time.Time) {
	nodes := g.Nodes()

	nodesInInterval := make(map[int64]struct{}, len(nodeMap))
	removeIDs := make(map[int64]struct{}, len(nodeMap))

	for nodes.Next() { // Find nodes that are in the correct time interval
		n := nodes.Node()
		id := n.ID()
		publishTime, err := time.Parse(time.RFC3339, nodeMap[id].Timestamp)
		if err != nil {
			panic(err)
		}
		if InInterval(publishTime, beginTime, endTime) {
			nodesInInterval[id] = struct{}{}
		}
	}

	for id := range nodeMap {
		if _, ok := nodesInInterval[id]; !ok { // If the node id was not on the list, kick it out
			removeIDs[id] = struct{}{}
		}
	}

	keepSelectedNodes(g, removeIDs)
}

// FilterLatestNoTraversal filters the nodes in the graph to their latest/newest releases. If interested in finding
// the latest packages in a timeframe, FilterNoTraversal needs to be called first. WARNING: This method is destructive,
// meaning that after running it, the nodes and their associated edges that do not correspond to the filter WILL BE REMOVED
// from the graph
func FilterLatestNoTraversal(g *DirectedGraph, nodeMap map[int64]NodeInfo) {
	length := g.Nodes().Len() / 2
	newestPackageVersion := make(map[uint32]NodeInfo, length)
	keepIDs := make(map[int64]struct{}, length)
	removeIDs := make(map[int64]struct{}, length)
	nodes := g.Nodes()

	for nodes.Next() {
		n := nodes.Node()
		current := nodeMap[n.ID()]
		currentDate, err := time.Parse(time.RFC3339, nodeMap[n.ID()].Timestamp)
		if err != nil {
			panic(err)
		}
		hash := hashPackageName(current.Name)

		if latest, ok := newestPackageVersion[hash]; ok {
			latestDate, err := time.Parse(time.RFC3339, latest.Timestamp)
			if err != nil {
				panic(err)
			}
			if currentDate.After(latestDate) { // If the key exists, and current date is later than the one stored
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

	for _, v := range newestPackageVersion {
		keepIDs[v.id] = struct{}{}
	}

	for id := range nodeMap {
		if _, ok := keepIDs[id]; !ok { // If the node id was not on the list, kick it out
			removeIDs[id] = struct{}{}
		}
	}

	keepSelectedNodes(g, removeIDs)

}

func keepSelectedNodes(g *DirectedGraph, removeIDs map[int64]struct{}) {
	edges := g.Edges()
	for edges.Next() {
		e := edges.Edge()
		fid := e.From().ID()
		tid := e.To().ID()

		if _, ok := removeIDs[fid]; ok {
			g.RemoveEdge(fid, tid)
		}
		if _, ok := removeIDs[tid]; ok {
			g.RemoveEdge(fid, tid)
		}
	}

	for id := range removeIDs {
		g.RemoveNode(id)
	}
}
