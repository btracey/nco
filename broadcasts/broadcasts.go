package broadcasts

import "github.com/btracey/nco/ncolp"

// SenderReceiverIndivPairs means that each sender must be able to send to each receiver.
func SenderReceiverIndivPairs(nSender, nReceiver int) []ncolp.Broadcast {
	var broadcasts []ncolp.Broadcast
	// Each broadcast is only the one message.
	for i := 0; i < nSender; i++ {
		for j := 0; j < nReceiver; j++ {
			mess := ncolp.Message{
				Sender:    ncolp.Sender(i),
				Receivers: []ncolp.Node{ncolp.Receiver(j)},
				Weight:    1,
			}
			broadcast := ncolp.Broadcast{mess}
			broadcasts = append(broadcasts, broadcast)
		}
	}
	return broadcasts
}
