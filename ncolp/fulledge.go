package ncolp

// RelayerGraph forces communication through Relayers.
type RelayerGraph struct {
	NumSender      int
	NumRelayer     int
	NumReceiver    int
	RelayerEpsilon float64
}

func (rg *RelayerGraph) Edges() []Edge {
	var edges []Edge
	// First, the senders can all talk to the relayers. Give the higher
	// relayers slightly higher costs
	for i := 0; i < rg.NumSender; i++ {
		e := edgesSenderToRelayers(Sender(i), rg.NumRelayer)
		edges = append(edges, e...)
	}

	// Next, the relayers to the receivers.
	for i := 0; i < rg.NumRelayer; i++ {
		e := edgesRelayerToReceivers(Relayer(i), rg.NumReceiver)
		edges = append(edges, e...)
	}

	// Now, add an edge from relayer i to relayer j for j > i,
	for i := 0; i < rg.NumRelayer; i++ {
		e := edgesRelayerToRelayers(Relayer(i), rg.NumRelayer)
		edges = append(edges, e...)
	}
	return edges
}

func (rg *RelayerGraph) EdgeCost(edge Edge) float64 {
	switch t := edge.(type) {
	default:
		panic("uncoded edge type")
	case EdgeST:
		return 1
	case EdgeRT:
		return 1 + float64(t.From)*rg.RelayerEpsilon
	case EdgeSR:
		return 1 + float64(t.To)*rg.RelayerEpsilon
	case EdgeRR:
		return 1 + float64(t.From)*rg.RelayerEpsilon + float64(t.To)*rg.RelayerEpsilon
	}
}

func (rg *RelayerGraph) Nodes() []Node {
	var nodes []Node
	for i := 0; i < rg.NumSender; i++ {
		nodes = append(nodes, Sender(i))
	}
	for i := 0; i < rg.NumRelayer; i++ {
		nodes = append(nodes, Relayer(i))
	}
	for i := 0; i < rg.NumReceiver; i++ {
		nodes = append(nodes, Receiver(i))
	}
	return nodes
}

func edgesSenderToRelayers(sender Sender, nRelayer int) []Edge {
	var edges []Edge
	for j := 0; j < nRelayer; j++ {
		e := EdgeSR{
			From: sender,
			To:   Relayer(j),
		}
		edges = append(edges, e)
	}
	return edges
}

func edgesRelayerToReceivers(relayer Relayer, nReceiver int) []Edge {
	var edges []Edge
	for j := 0; j < nReceiver; j++ {
		e := EdgeRT{
			From: relayer,
			To:   Receiver(j),
		}
		edges = append(edges, e)
	}
	return edges
}

// only talk to higher indexed relayers
func edgesRelayerToRelayers(relayer Relayer, nRelayer int) []Edge {
	var edges []Edge
	for j := int(relayer) + 1; j < nRelayer; j++ {
		e := EdgeRR{
			From: relayer,
			To:   Relayer(j),
		}
		edges = append(edges, e)
	}
	return edges
}

func (rg *RelayerGraph) EdgesFrom(node Node) []Edge {
	switch t := node.(type) {
	default:
		panic("unknown node type")
	case Sender:
		return edgesSenderToRelayers(t, rg.NumRelayer)
	case Relayer:
		var edges []Edge
		e := edgesRelayerToReceivers(t, rg.NumReceiver)
		edges = append(edges, e...)
		e = edgesRelayerToRelayers(t, rg.NumRelayer)
		edges = append(edges, e...)
		return edges
	case Receiver:
		return nil
	}
}

func (rg *RelayerGraph) EdgesTo(node Node) []Edge {
	switch t := node.(type) {
	default:
		panic("unknown node type")
	case Sender:
		return nil
	case Relayer:
		var edges []Edge
		// The relayer is talked to by all senders, and by all receivers with
		// a lower index.
		for i := 0; i < rg.NumSender; i++ {
			edges = append(edges, EdgeSR{Sender(i), t})
		}
		for i := 0; i < int(t); i++ {
			edges = append(edges, EdgeRR{Relayer(i), t})
		}
		return edges
	case Receiver:
		var edges []Edge
		// Talked to by all relayers
		for i := 0; i < rg.NumRelayer; i++ {
			edges = append(edges, EdgeRT{Relayer(i), t})
		}
		return edges
	}
}
