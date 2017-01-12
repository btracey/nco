package main

import (
	"fmt"
	"log"
	"os"

	"github.com/btracey/nco/broadcasts"
	"github.com/btracey/nco/ncolp"

	"github.com/btracey/lpwrap"
)

func main() {

	// CPU profiling

	nSender := 10
	nRelayer := 8
	nReceiver := 10
	listen := 10.0
	tell := 10.0

	// Get all of the broadcasts
	bcs := broadcasts.SenderReceiverIndivPairs(nSender, nReceiver)

	graph := &ncolp.RelayerGraph{nSender, nRelayer, nReceiver, 1e-2}

	lp := ncolp.BasicLP(bcs, graph)

	_, _ = listen, tell
	respCons := ncolp.ResponsibilityConstraints(graph, listen, tell)
	lp.Constraints = append(lp.Constraints, respCons...)

	fmt.Println("num Constraints", len(lp.Constraints))
	/*
		// Solving with Gonum
			fmt.Println("Starting solve")
			result, err := lpwrap.Gonum{}.Solve(lp)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("num Variables", len(result.Ordered()))
			fmt.Println("Optimal value is:", result.Value)
			ordered := result.Ordered()
			for _, v := range ordered {
				if v.Value == 0 {
					continue
				}
				fmt.Println(v.Var, "=", v.Value)
			}
	*/

	// Solving with Gurobi
	f, err := os.Create("problem.lp")
	if err != nil {
		log.Fatal(err)
	}
	gur := lpwrap.Gurobi{}

	err = gur.WriteGurobi(f, lp)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
}
