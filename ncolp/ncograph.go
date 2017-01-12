package ncolp

import "strconv"

type Sender int

func (s Sender) Name() string {
	return "s" + strconv.Itoa(int(s))
}

type Relayer int

func (r Relayer) Name() string {
	return "r" + strconv.Itoa(int(r))
}

type Receiver int

func (t Receiver) Name() string {
	return "t" + strconv.Itoa(int(t))
}

// EdgeSR is an edge from a sender to a relayer.
type EdgeSR struct {
	From Sender
	To   Relayer
}

func (e EdgeSR) Name() string {
	return "e" + e.From.Name() + e.To.Name()
}

// EdgeST is an edge from a sender to a receiver.
type EdgeST struct {
	From Sender
	To   Receiver
}

func (e EdgeST) Name() string {
	return "e" + e.From.Name() + e.To.Name()
}

// EdgeRT is an edge from a relayer to a receiver.
type EdgeRT struct {
	From Relayer
	To   Receiver
}

func (e EdgeRT) Name() string {
	return "e" + e.From.Name() + e.To.Name()
}

// EdgeRR is an edge from a relayer to a relayer.
type EdgeRR struct {
	From Relayer
	To   Relayer
}

func (e EdgeRR) Name() string {
	return "e" + e.From.Name() + e.To.Name()
}
