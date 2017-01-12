package ncolp

import (
	"strconv"

	"github.com/btracey/lpwrap"
)

type Graph interface {
	Nodes() []Node
	Edges() []Edge
	EdgeCost(edge Edge) float64
	EdgesTo(node Node) []Edge
	EdgesFrom(node Node) []Edge
}

// Constructs the basic LP with objective and flow constraints.
func BasicLP(broadcasts []Broadcast, graph Graph) lpwrap.LP {
	edges := graph.Edges()
	nodes := graph.Nodes()

	obj := makeObjective(edges, graph)

	var cons []lpwrap.Constraint

	// Add non-negativity constraints for all of the edge capacities
	for _, edge := range edges {
		c := nonNegativity(edge.Name())
		cons = append(cons, c)
	}

	// Add non-negativity constraints for all of the message flows
	for i, bcast := range broadcasts {
		for j := range bcast {
			for _, edge := range edges {
				flowname := MessageFlowName(i, j, edge)
				cons = append(cons, nonNegativity(flowname))
			}
		}
	}

	// Add constraint that the outflow has to equal the message weight.
	for i, bcast := range broadcasts {
		for j, mess := range bcast {
			for _, node := range mess.Receivers {
				con := lpwrap.Constraint{
					Comp:  lpwrap.EQ,
					Right: []lpwrap.Term{{lpwrap.Constant, mess.Weight}},
				}
				edges := graph.EdgesTo(node)
				for _, edge := range edges {
					flowname := MessageFlowName(i, j, edge)
					term := lpwrap.Term{flowname, 1}
					con.Left = append(con.Left, term)
				}
				cons = append(cons, con)
			}
		}
	}

	// Add the constraint that the outlet edges from the source have to be less than
	// the message weight.
	for i, bcast := range broadcasts {
		for j, mess := range bcast {
			src := mess.Sender
			children := graph.EdgesFrom(src)
			for _, edge := range children {
				flowname := MessageFlowName(i, j, edge)
				con := lpwrap.Constraint{
					Left:  []lpwrap.Term{{flowname, 1}},
					Comp:  lpwrap.LE,
					Right: []lpwrap.Term{{lpwrap.Constant, mess.Weight}},
				}
				cons = append(cons, con)
			}
		}
	}

	// For each node, add the constraint that each outflow has to be less than
	// or equal to the sum of the inflows
	for _, node := range nodes {
		parents := graph.EdgesTo(node)
		children := graph.EdgesFrom(node)
		if len(children) == 0 {
			continue
		}
		if len(parents) == 0 {
			// No flow from the oulets UNLESS this is the source for that message
			for i, bcast := range broadcasts {
				for j, mess := range bcast {
					if node.Name() == mess.Sender.Name() {
						continue
					}
					for _, child := range children {
						childFlowName := MessageFlowName(i, j, child)
						con := lpwrap.Constraint{
							Left:  []lpwrap.Term{{childFlowName, 1}},
							Comp:  lpwrap.EQ,
							Right: []lpwrap.Term{{lpwrap.Constant, 0}},
						}
						cons = append(cons, con)
					}
				}
			}
			continue
		}
		for i, bcast := range broadcasts {
			for j, _ := range bcast {
				for _, child := range children {
					childFlowName := MessageFlowName(i, j, child)
					con := lpwrap.Constraint{
						Left: []lpwrap.Term{{childFlowName, 1}},
						Comp: lpwrap.LE,
					}
					for _, parent := range parents {
						parentFlowName := MessageFlowName(i, j, parent)
						con.Right = append(con.Right, lpwrap.Term{parentFlowName, 1})
					}
					cons = append(cons, con)
				}
			}
		}
	}

	// Add constraints that the sum of per-message flows has to be less than
	// the edge capacity
	// TODO(btracey): There are optimizations if there is only one receiver (but
	// more complicated).
	for _, edge := range edges {
		for i, bcast := range broadcasts {
			con := lpwrap.Constraint{
				Comp:  lpwrap.LE,
				Right: []lpwrap.Term{{edge.Name(), 1}},
			}
			for j := range bcast {
				name := MessageFlowName(i, j, edge)
				con.Left = append(con.Left, lpwrap.Term{name, 1})
			}
			cons = append(cons, con)
		}
	}

	// Enforce the max flow constraints per message per receiver.

	// First, non-negativity, and that the flow must be less than the message flow
	for i, bcast := range broadcasts {
		for j, mess := range bcast {
			for _, edge := range edges {
				mfn := MessageFlowName(i, j, edge)
				for _, receiver := range mess.Receivers {
					mefn := MessageReceiverFlowName(i, j, receiver, edge)
					con := lpwrap.Constraint{
						Left:  []lpwrap.Term{{mefn, 1}},
						Comp:  lpwrap.LE,
						Right: []lpwrap.Term{{mfn, 1}},
					}
					cons = append(cons, con)
					con = nonNegativity(mefn)
					cons = append(cons, con)
				}
			}
		}
	}

	// Add constraint that outlets of source from this receiver must equal omega
	for i, bcast := range broadcasts {
		for j, mess := range bcast {
			src := mess.Sender
			children := graph.EdgesFrom(src)
			for _, receiver := range mess.Receivers {
				con := lpwrap.Constraint{
					Comp:  lpwrap.EQ,
					Right: []lpwrap.Term{{lpwrap.Constant, mess.Weight}},
				}
				for _, edge := range children {
					mefn := MessageReceiverFlowName(i, j, receiver, edge)
					con.Left = append(con.Left, lpwrap.Term{mefn, 1})
				}
				cons = append(cons, con)
			}
		}
	}

	// Add constraint that parents of receiver must sum to omega
	for i, bcast := range broadcasts {
		for j, mess := range bcast {
			for _, receiver := range mess.Receivers {
				parents := graph.EdgesTo(receiver)
				con := lpwrap.Constraint{
					Comp:  lpwrap.EQ,
					Right: []lpwrap.Term{{lpwrap.Constant, mess.Weight}},
				}
				for _, edge := range parents {
					mefn := MessageReceiverFlowName(i, j, receiver, edge)
					con.Left = append(con.Left, lpwrap.Term{mefn, 1})
				}
				cons = append(cons, con)
			}
		}
	}

	// Add flow continuity constraints for all of the middle nodes
	for i, bcast := range broadcasts {
		for j, mess := range bcast {
			for _, receiver := range mess.Receivers {
				for _, node := range nodes {
					parents := graph.EdgesTo(node)
					children := graph.EdgesFrom(node)
					if len(parents) == 0 || len(children) == 0 {
						continue
					}
					con := lpwrap.Constraint{
						Comp: lpwrap.EQ,
					}
					for _, edge := range parents {
						mefn := MessageReceiverFlowName(i, j, receiver, edge)
						con.Left = append(con.Left, lpwrap.Term{mefn, 1})
					}
					for _, edge := range children {
						mefn := MessageReceiverFlowName(i, j, receiver, edge)
						con.Right = append(con.Right, lpwrap.Term{mefn, 1})
					}
					cons = append(cons, con)
				}
			}
		}
	}

	return lpwrap.LP{
		Objective:   obj,
		Constraints: cons,
	}
}

// ResponsibilityConstraints says that each node can only listen overall to certain
// number (not per message)
func ResponsibilityConstraints(graph Graph, listen, tell float64) []lpwrap.Constraint {
	var cons []lpwrap.Constraint
	nodes := graph.Nodes()
	for _, node := range nodes {
		parents := graph.EdgesTo(node)
		if len(parents) != 0 {
			con := lpwrap.Constraint{
				Comp:  lpwrap.LE,
				Right: []lpwrap.Term{{lpwrap.Constant, listen}},
			}
			for _, parent := range parents {
				con.Left = append(con.Left, lpwrap.Term{parent.Name(), 1})
			}
			cons = append(cons, con)
		}

		children := graph.EdgesFrom(node)
		if len(children) != 0 {
			con := lpwrap.Constraint{
				Comp:  lpwrap.LE,
				Right: []lpwrap.Term{{lpwrap.Constant, tell}},
			}
			for _, child := range children {
				con.Left = append(con.Left, lpwrap.Term{child.Name(), 1})
			}
			cons = append(cons, con)
		}

	}
	return cons
}

func MessageFlowName(bcastIdx, messIdx int, edge Edge) string {
	return "b" + strconv.Itoa(bcastIdx) + "m" + strconv.Itoa(messIdx) + edge.Name()
}

func MessageReceiverFlowName(bcastIdx, messIdx int, receiver Node, edge Edge) string {
	return "b" + strconv.Itoa(bcastIdx) + "m" + strconv.Itoa(messIdx) + edge.Name() + "_" + receiver.Name()
}

// nonNegativity makes a non-negativity constraint for the given variable name.
func nonNegativity(name string) lpwrap.Constraint {
	return lpwrap.Constraint{
		Left:  []lpwrap.Term{{name, 1}},
		Comp:  lpwrap.GE,
		Right: []lpwrap.Term{{lpwrap.Constant, 0}},
	}
}

func makeObjective(edges []Edge, graph Graph) lpwrap.Objective {
	terms := make([]lpwrap.Term, len(edges))
	for i, edge := range edges {
		terms[i] = lpwrap.Term{edge.Name(), graph.EdgeCost(edge)}
	}
	return lpwrap.Objective{
		terms,
		lpwrap.Minimize,
	}
}

type Message struct {
	Sender    Node
	Receivers []Node
	Weight    float64
}

type Broadcast []Message

type Edge interface {
	Name() string
}

type Node interface {
	Name() string
}
