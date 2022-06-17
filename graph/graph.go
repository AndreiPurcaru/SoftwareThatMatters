package graph

import (
	"fmt"
	"gonum.org/v1/gonum/graph/simple"
	"hash/crc32"
	"hash/crc64"
	"log"
	"os"
	"time"

	"github.com/Masterminds/semver"
	"github.com/mailru/easyjson"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/network"
	"gonum.org/v1/gonum/graph/traverse"
)

type VersionInfo struct {
	Dependencies map[string]string `json:"dependencies"`
	Timestamp    string            `json:"timestamp"`
}

type PackageInfo struct {
	Versions map[string]VersionInfo `json:"versions"`
	Name     string                 `json:"name"`
}

type Doc struct {
	Pkgs []PackageInfo `json:"pkgs"`
}

// NodeInfo is a type structure for nodes. Name and Version can be removed if we find we don't use them often enough
type NodeInfo struct {
	Timestamp string
	Name      string
	Version   string
	id        int64
}

var crcTable *crc64.Table = crc64.MakeTable(crc64.ISO)

// NewNodeInfo constructs a NodeInfo structure and automatically fills the stringID.
func NewNodeInfo(id int64, name string, version string, timestamp string) *NodeInfo {
	return &NodeInfo{
		id:        id,
		Name:      name,
		Version:   version,
		Timestamp: timestamp}
}

func (nodeInfo NodeInfo) String() string {
	return fmt.Sprintf("Package: %v - Version: %v", nodeInfo.Name, nodeInfo.Version)
}

// CreateStringIDToNodeInfoMap takes a list of PackageInfo and a DirectedGraph. For each of the packages,
// it creates a mapping of stringIDs to NodeInfo and also adds a node to the graph. The handling of the IDs is delegated
// to Gonum. These IDs are also included in the mapping for ease of access.
func CreateStringIDToNodeInfoMap(packagesInfo *[]PackageInfo, graph *DirectedGraph) map[string]NodeInfo {
	stringIDToNodeInfoMap := make(map[string]NodeInfo, len(*packagesInfo))
	for _, packageInfo := range *packagesInfo {
		for packageVersion, versionInfo := range packageInfo.Versions {
			packageNameVersionString := fmt.Sprintf("%s-%s", packageInfo.Name, packageVersion)
			// Delegate the work of creating a unique ID to Gonum
			newNode := graph.NewNode()
			newId := newNode.ID()
			stringIDToNodeInfoMap[packageNameVersionString] = *NewNodeInfo(newId, packageInfo.Name, packageVersion, versionInfo.Timestamp)
			graph.AddNode(newNode)
		}
	}
	return stringIDToNodeInfoMap
}

// TODO: Maybe change to something like CreateIdToNodeInfoMap so it's not confusing for other people.

func CreateNodeIdToPackageMap(m map[string]NodeInfo) map[int64]NodeInfo {
	s := make(map[int64]NodeInfo, len(m))
	for _, val := range m {
		s[val.id] = val
	}
	return s
}

func CreateHashedVersionMap(pi *[]PackageInfo) map[uint32][]string {
	result := make(map[uint32][]string, len(*pi))
	for _, pkg := range *pi {
		hashedName := hashPackageName(pkg.Name)
		result[hashedName] = make([]string, 0, len(pkg.Versions))
		for ver := range pkg.Versions {
			result[hashedName] = append(result[hashedName], ver)
		}
	}
	return result
}

func CreateNameToVersionMap(m *[]PackageInfo) map[string][]string {
	newMap := make(map[string][]string, len(*m))
	for _, value := range *m {
		name := value.Name
		for k := range value.Versions {
			newMap[name] = append(newMap[name], k)
		}
	}
	return newMap
}

// CreateEdges takes a graph, a list of packages and their dependencies, a map of stringIDs to NodeInfo and
// a map of names to versions and creates directed edges between the dependent library and its dependencies.
func CreateEdges(graph *DirectedGraph, inputList *[]PackageInfo, hashToNodeId map[uint64]int64, hashToVersionMap map[uint32][]string, isMaven bool) {
	packagesLength := len(*inputList)
	edgesAmount := 0
	channel := make(chan int, 2)
	go func(n int, ch chan int) {
		for {
			for i := range ch {
				fmt.Printf("\u001b[1A \u001b[2K \r") // Clear the last line
				fmt.Printf("%.2f%% done (%d / %d packages connected to their dependencies)\n", float64(i)/float64(n)*100, i, n)
			}
		}
	}(packagesLength, channel)
	for id, packageInfo := range *inputList {
		for version, dependencyInfo := range packageInfo.Versions {
			for dependencyName, dependencyVersion := range dependencyInfo.Dependencies {
				dependencySemanticVersioning := dependencyVersion
				if isMaven {
					dependencySemanticVersioning = ParseMultipleMavenSemanticVersions(dependencyVersion)
				}
				constraint, err := semver.NewConstraint(dependencySemanticVersioning)

				if err != nil {
					// A lot of packages don't respect semver. This ensures that we don't crash when we encounter them.
					continue
				}
				for _, v := range LookupVersions(dependencyName, hashToVersionMap) {
					newVersion, err := semver.NewVersion(v)
					if err != nil {
						continue
					}
					if constraint.Check(newVersion) {
						dependencyStringId := fmt.Sprintf("%s-%s", dependencyName, v)
						dependencyGoId := LookupByStringId(dependencyStringId, hashToNodeId)

						packageStringId := fmt.Sprintf("%s-%s", packageInfo.Name, version)
						packageGoId := LookupByStringId(packageStringId, hashToNodeId)

						// Ensure that we do not create edgesAmount to self because some packages do that...
						if dependencyGoId != packageGoId {
							packageNode := graph.Node(packageGoId)
							dependencyNode := graph.Node(dependencyGoId)
							graph.SetEdge(simple.Edge{F: packageNode, T: dependencyNode})
							edgesAmount++
						}

					}
				}
			}
		}
		channel <- id
	}
	close(channel)
	fmt.Printf("Nodes: %d, Edges: %d\n", len(hashToNodeId), edgesAmount)
}

func ParseJSON(inPath string) []PackageInfo {

	f, err := os.Open(inPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var result Doc
	err = easyjson.UnmarshalFromReader(f, &result)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Read %d packages\n", len(result.Pkgs))

	return result.Pkgs
}

func CreateMaps(packageList *[]PackageInfo, graph *DirectedGraph) (map[uint64]int64, map[int64]NodeInfo) {
	hashToNodeId := make(map[uint64]int64, len(*packageList)*10)
	idToNodeInfo := make(map[int64]NodeInfo, len(*packageList)*10)
	for _, packageInfo := range *packageList {
		for packageVersion, versionInfo := range packageInfo.Versions {
			stringID := fmt.Sprintf("%s-%s", packageInfo.Name, packageVersion)
			hashed := hashStringId(stringID)
			// Delegate the work of creating a unique ID to Gonum
			newNode := graph.NewNode()
			newId := newNode.ID()
			hashToNodeId[hashed] = newId
			idToNodeInfo[newId] = *NewNodeInfo(newId, packageInfo.Name, packageVersion, versionInfo.Timestamp)
			graph.AddNode(newNode)
		}
	}
	return hashToNodeId, idToNodeInfo
}

func hashStringId(stringID string) uint64 {
	hashed := crc64.Checksum([]byte(stringID), crcTable)
	return hashed
}

func hashPackageName(packageName string) uint32 {
	hashed := crc32.ChecksumIEEE([]byte(packageName))
	return hashed
}

func LookupVersions(packageName string, versionMap map[uint32][]string) []string {
	hash := hashPackageName(packageName)
	return versionMap[hash]
}

func LookupByStringId(stringId string, hashTable map[uint64]int64) int64 {
	hash := hashStringId(stringId)
	goId := hashTable[hash]
	return goId
}

func CreateGraph(inputPath string, isUsingMaven bool) (*DirectedGraph, map[uint64]int64, map[int64]NodeInfo) {
	fmt.Println("Parsing input")
	packagesList := ParseJSON(inputPath)

	directedGraph := NewDirectedGraph()

	fmt.Println("Adding nodes and creating indices")

	hashToNodeId, idToNodeInfo := CreateMaps(&packagesList, directedGraph)
	hashToVersions := CreateHashedVersionMap(&packagesList)

	fmt.Println("Creating edges")

	CreateEdges(directedGraph, &packagesList, hashToNodeId, hashToVersions, isUsingMaven)

	fmt.Println("Done creating edges!")

	return directedGraph, hashToNodeId, idToNodeInfo
}

// InInterval returns true when time t lies in the interval [begin, end], false otherwise
func InInterval(t, begin, end time.Time) bool {
	return t.Equal(begin) || t.Equal(end) || t.After(begin) && t.Before(end)
}

// initializeTraversal is a helper function used to initialize all required auxiliary data structures for the graph traversal
func initializeTraversal(g *DirectedGraph, nodeMap map[int64]NodeInfo, withinInterval map[int64]bool, beginTime time.Time, endTime time.Time) {
	nodes := g.Nodes()
	for nodes.Next() { // Initialize withinInterval data structure
		n := nodes.Node()
		id := n.ID()
		publishTime, err := time.Parse(time.RFC3339, nodeMap[id].Timestamp)
		if err != nil {
			panic(err)
		}
		if InInterval(publishTime, beginTime, endTime) {
			withinInterval[id] = true
		}
	}
}

func removeDisconnected(g *DirectedGraph, connected []*graph.Edge) {
	edges := g.Edges()
	for edges.Next() {
		edge := edges.Edge()
		for _, disconnectedEdge := range connected { // Found that it's connected, move on
			if edge == *disconnectedEdge {
				break
			} else {
				g.RemoveEdge(edge.From().ID(), edge.To().ID())
			}
		}
	}
}

// This function removes stale edges from the specified graph by doing a DFS with all packages as the root node in O(n^2)
func traverseAndRemoveEdges(g *DirectedGraph, nodeMap map[int64]NodeInfo, withinInterval map[int64]bool) {
	nodes := g.Nodes()
	// This keeps track of which edges we've connected
	connected := make([]*graph.Edge, 0, len(nodeMap)*2)

	t := traverse.BreadthFirst{
		Traverse: func(e graph.Edge) bool { // The dependent / parent node
			fromId := e.From().ID()
			toId := e.To().ID()
			if withinInterval[toId] {

				fromTime, err1 := time.Parse(time.RFC3339, nodeMap[fromId].Timestamp) // The dependent node's time stamp
				toTime, err2 := time.Parse(time.RFC3339, nodeMap[toId].Timestamp)     // The dependency node's time stamp

				if err1 != nil || err2 != nil {
					panic(err1)
				}

				if fromTime.After(toTime) {
					connected = append(connected, &e)
				} // If the dependency was released before the parent node, add this edge to the connected nodes
			}

			return true
		},
	}

	nodesAmount := len(nodeMap)

	i := 0

	for nodes.Next() {
		n := nodes.Node()
		if withinInterval[n.ID()] { // We'll only consider traversing this subtree if its root was within the specified time interval
			_ = t.Walk(g, n, nil) // Continue walking this subtree until we've visited everything we're allowed to according to Traverse
			t.Reset()             // Clean up for the next iteration
			i++
			fmt.Printf("\u001b[1A \u001b[2K \r") // Clear the last line
			fmt.Printf("%d / %d subtrees walked \n", i, nodesAmount)

		}
	}

	removeDisconnected(g, connected)

}

func traverseOneNode(g *DirectedGraph, nodeMap map[int64]NodeInfo, withinInterval map[int64]bool, nodeId int64) {
	connected := make([]*graph.Edge, 0, len(nodeMap)*2)

	t := traverse.BreadthFirst{
		Traverse: func(e graph.Edge) bool { // The dependent / parent node
			var traversal bool
			fromId := e.From().ID()
			toId := e.To().ID()
			if withinInterval[toId] {
				fromTime, err1 := time.Parse(time.RFC3339, nodeMap[fromId].Timestamp) // The dependent node's time stamp
				toTime, err2 := time.Parse(time.RFC3339, nodeMap[toId].Timestamp)     // The dependency node's time stamp

				if err1 != nil || err2 != nil {
					panic(err1)
				}

				if traversal = fromTime.After(toTime); traversal {
					connected = append(connected, &e)
				} // If the dependency was released before the parent node, add this edge to the connected nodes
			}

			return traversal
		},
	}

	_ = t.Walk(g, g.Node(nodeId), nil)
	removeDisconnected(g, connected)
}

func filterGraph(g *DirectedGraph, nodeMap map[int64]NodeInfo, beginTime, endTime time.Time) {
	// This stores whether the package existed in the specified time range
	withinInterval := make(map[int64]bool, len(nodeMap))

	initializeTraversal(g, nodeMap, withinInterval, beginTime, endTime) // Initialize all auxillary data structures for the traversal

	traverseAndRemoveEdges(g, nodeMap, withinInterval) // Traverse the graph and remove stale edges
}

func FilterGraph(g *DirectedGraph, nodeMap map[int64]NodeInfo, beginTime, endTime time.Time) {
	filterGraph(g, nodeMap, beginTime, endTime)
}

func findNode(hashMap map[uint64]int64, idToNodeInfo map[int64]NodeInfo, stringId string) (int64, bool) {
	var nodeId int64
	var correctOk bool
	if info, ok := idToNodeInfo[LookupByStringId(stringId, hashMap)]; ok {
		nodeId = info.id
		correctOk = true
	} else {
		log.Printf("String id %s was not found \n", stringId)
		correctOk = false
	}
	return nodeId, correctOk
}

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

func FilterNode(g *DirectedGraph, hashMap map[uint64]int64, nodeMap map[int64]NodeInfo, stringId string, beginTime, endTime time.Time) {

	var nodeId int64
	if id, ok := findNode(hashMap, nodeMap, stringId); ok {
		nodeId = id
	} else {
		return // This function is a no-op if we don't have a correct string id
	}

	// This stores whether the package existed in the specified time range
	withinInterval := make(map[int64]bool, len(nodeMap))

	initializeTraversal(g, nodeMap, withinInterval, beginTime, endTime) // Initialize all auxillary data structures for the traversal

	traverseOneNode(g, nodeMap, withinInterval, nodeId)
}

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

// GetLatestTransitiveDependenciesNode gets the latest dependencies matching the node's version constraints. If you want this within a specific time frame, use filterNode first
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

// FilterLatestDepsGraph filters the graph between the two given time stamps and then only keep the latest dependencies
func FilterLatestDepsGraph(g *DirectedGraph, nodeMap map[int64]NodeInfo, hashMap map[uint64]int64, beginTime, endTime time.Time) {
	filterGraph(g, nodeMap, beginTime, endTime)
	length := g.Nodes().Len() / 2

	keepIDs := make(map[int64]struct{}, length)
	removeIDs := make(map[int64]struct{}, length)
	newestPackageVersion := make(map[uint32]NodeInfo, length)
	v := traverse.DepthFirst{
		Visit: func(n graph.Node) {
			current := nodeMap[n.ID()]
			currentDate, _ := time.Parse(time.RFC3339, current.Timestamp)
			hash := hashPackageName(current.Name)

			if latest, ok := newestPackageVersion[hash]; ok {
				latestDate, _ := time.Parse(time.RFC3339, latest.Timestamp)
				if currentDate.After(latestDate) { // If the key exists, and current date is later than the one stored
					newestPackageVersion[hash] = current // Set to the current package
				} else if currentDate.Equal(latestDate) { // If the dates are somehow equal, compare version numbers
					currentversion, _ := semver.NewVersion(current.Version)
					latestVersion, _ := semver.NewVersion(latest.Version)

					if currentversion.GreaterThan(latestVersion) {
						newestPackageVersion[hash] = current
					}
				}
			} else { // If the key doesn't exist yet
				newestPackageVersion[hash] = current
			}
		},
	}
	nodesAmount := len(hashMap)
	nodes := g.Nodes()

	i := 0
	for nodes.Next() {
		n := nodes.Node()
		_ = v.Walk(g, n, nil)
		v.Reset()
		i++
		fmt.Printf("\u001b[1A \u001b[2K \r") // Clear the last line
		fmt.Printf("%d / %d subtrees walked \n", i, nodesAmount)
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

// PageRank uses the sparse page rank algorithm to find the Page ranks of all nodes
func PageRank(graph *DirectedGraph) map[int64]float64 {
	pr := network.PageRankSparse(graph, 0.85, 0.001)
	return pr
}

func Betweenness(graph *DirectedGraph) map[int64]float64 {
	betweenness := network.Betweenness(graph)
	return betweenness
}
