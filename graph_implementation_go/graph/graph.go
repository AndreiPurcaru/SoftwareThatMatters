package graph

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/mailru/easyjson"
	"gonum.org/v1/gonum/graph/simple"
	"log"
	"os"
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

// CreateEdges takes a graph, a list of packages and their dependencies, a map of stringIDs to NodeInfo and
// a map of names to versions and creates directed edges between the dependent library and its dependencies.
func CreateEdges(graph *DirectedGraph, inputList *[]PackageInfo, hashToNodeId map[uint64]int64, hashToVersionMap map[uint32][]string, isMaven bool) {
	packagesLength := len(*inputList)
	edgesAmount := 0
	channel := make(chan int, 1000)
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
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}(f)

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
