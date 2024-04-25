package dto

import (
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/dto/dag"
	"github.com/buildbeaver/buildbeaver/server/dto/dag/tfdiags"
)

// RootNode is a special node that is inserted in to the root of the dag
// to ensure there is only ever one root node.
const RootNode = "root"

// GraphNode represents a node in the DAG.
type GraphNode interface {
	// GetFQN returns the unique name/identifier for this node.
	GetFQN() models.NodeFQN
	// GetFQNDependencies returns a list of nodes by FQN that this node depends on.
	GetFQNDependencies() []models.NodeFQN
}

// DAG represents a directed acyclic graph useful for expressing dependencies.
type DAG struct {
	graph *dag.AcyclicGraph
}

// NewDAG creates a new DAG containing the specified nodes.
// The DAG is validated after construction and any validation errors are returned.
func NewDAG(vertices []GraphNode) (*DAG, error) {

	var (
		graph          = &dag.AcyclicGraph{}
		verticesByName = make(map[models.NodeFQN]interface{})
	)

	graph.SetDebugWriter(ioutil.Discard)

	// First pass - add all vertices
	graph.Add(RootNode)

	for _, vertex := range vertices {
		verticesByName[vertex.GetFQN()] = vertex
		graph.Add(vertex)
	}

	// Second pass - add all edges
	for _, vertex := range vertices {
		jobV, ok := verticesByName[vertex.GetFQN()]
		if !ok {
			return nil, fmt.Errorf("error unknown vertex: %s", vertex.GetFQN())
		}
		nrEdgesForVertex := 0
		for _, dependency := range vertex.GetFQNDependencies() {
			depV, ok := verticesByName[dependency]
			if ok {
				edge := dag.BasicEdge(depV, jobV)
				graph.Connect(edge)
				nrEdgesForVertex++
			} else {
				// Dependency doesn't exist. This is only an error if it should be in the same workflow as the
				// dependent job; otherwise just ignore the dependency and don't add an edge.
				if vertex.GetFQN().WorkflowName == dependency.WorkflowName {
					return nil, fmt.Errorf("error unknown vertex: %s", dependency)
				}
			}
		}
		// Ensure we connect everything to the root node, directly or indirectly
		if nrEdgesForVertex == 0 {
			edge := dag.BasicEdge(RootNode, verticesByName[vertex.GetFQN()])
			graph.Connect(edge)
		}
	}

	err := graph.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "error validating dependencies")
	}

	graph.TransitiveReduction()

	return &DAG{graph: graph}, nil
}

// Ancestors returns all ancestors of the specified vertex.
func (m *DAG) Ancestors(of interface{}) ([]interface{}, error) {
	set, err := m.graph.Ancestors(of)
	if err != nil {
		return nil, err
	}
	return set.List(), nil
}

// Walk the DAG visiting each node once, after that node's dependencies have been visited.
// If parallel is true, the walk will be performed in parallel, and errors (if any) will be
// accumulated and returned at the end. If parallel is false, the walk will be performed in
// series, and the first error (if any) will immediately cause the walk to fail and that error
// will be returned.
func (m *DAG) Walk(parallel bool, callback func(interface{}) error) error {

	// NOTE: The underlying DAG library:
	//
	// * Does the walk in parallel where possible. We optionally serialize this using a mutex.
	//
	// * Continues on error. We disable this when running synchronously by not calling the supplied
	//   callback for nodes after the error occurred, ensuring additional nodes are a no-op and
	//   appearing to the outside world like we stopped as soon as the error occurred.
	//
	// * Uses a special error type that we're not interested in. We track errors in the outer scope instead.

	var (
		walkLock   sync.Mutex
		resultLock sync.Mutex
		result     *multierror.Error
	)

	innerCallback := func(vertex dag.Vertex) tfdiags.Diagnostics {

		var diags tfdiags.Diagnostics

		if !parallel {
			walkLock.Lock()
			defer walkLock.Unlock()
		}

		if vertex == RootNode {
			return nil
		}

		if parallel || result.ErrorOrNil() == nil {
			err := callback(vertex)
			if err != nil {
				resultLock.Lock()
				result = multierror.Append(result, err)
				resultLock.Unlock()
				diags = diags.Append(err)
			}
		}

		return diags
	}

	walker := &dag.Walker{Callback: innerCallback, Reverse: false}
	walker.Update(m.graph)
	walker.Wait()

	return result.ErrorOrNil()
}
