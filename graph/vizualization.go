package graph

import (
	"fmt"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"log"
	"os"
)

// Visualization writes the simple graph to a dot file, so it could be visualized with GraphViz. This includes only Ids
func Visualization(graph *DirectedGraph, name string) {
	result, _ := dot.Marshal(graph, name, "", "  ")

	file, err := os.Create(name + ".dot")

	if err != nil {
		log.Fatal("Error!", err)
	}
	defer file.Close()

	fmt.Fprint(file, string(result))

}

// VisualizationNodeInfo writes to dot file manually from the NodeInfoMap to include the Node info in the graphViz
func VisualizationNodeInfo(iDToNodeInfo map[int64]NodeInfo, graph *DirectedGraph, name string) {
	file, err := os.Create(name + ".dot")
	d1 := []byte("strict digraph" + " " + name + " " + "{\n")
	d2 := []byte("}")
	lab := string("[label = \" ")
	edgIt := graph.Edges()

	fmt.Fprint(file, string(d1))

	for _, element := range iDToNodeInfo {
		fmt.Fprintf(file, fmt.Sprint(element.id)+lab+element.Name+` \n `+element.Version+` \n `+element.Timestamp+"\""+"];\n")

	}

	for edgIt.Next() {
		fmt.Fprintf(file, fmt.Sprint(edgIt.Edge().From().ID())+" -> "+fmt.Sprint(edgIt.Edge().To().ID())+";\n")
	}

	fmt.Fprint(file, string(d2))

	if err != nil {
		panic(err)
	}

	defer file.Close()

}
