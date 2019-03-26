package cpuminer

import (
	"fmt"
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"github.com/drcsuite/drc/vote"
	"testing"
)

func TestEstimateScale(t *testing.T) {
	// 节点数少于300，上次收到的票数与实际节点数差不多，最新的节点数为47
	fmt.Println(vote.EstimateScale([]uint16{45, 46, 47, 48, 49}, []uint16{48, 49, 50, 51, 52}))
	// 投票比300少，说明之前节点估计得多了，最新的节点数要比之前的少，为4666
	fmt.Println(vote.EstimateScale([]uint16{260, 270, 280, 290, 300}, []uint16{4800, 4900, 5000, 5100, 5200}))
	// 投票比300多，说明之前节点估计得少了，最新的节点数要比之前的大，为5333
	fmt.Println(vote.EstimateScale([]uint16{300, 310, 320, 330, 340}, []uint16{4800, 4900, 5000, 5100, 5200}))
}

func TestBlockVerge(t *testing.T) {
	fmt.Println(vote.VoteVerge(257))
	fmt.Println(vote.BlockVerge(257))
}

func TestTicketPool(t *testing.T) {
	var signAndKey []SignAndKey
	signAndKeys := append(signAndKey, SignAndKey{chainhash.Hash64{1}, chainhash.Hash33{2}})
	ticketPool[chainhash.Hash{1}] = append(signAndKeys, SignAndKey{chainhash.Hash64{3}, chainhash.Hash33{4}})
	ticketPool[chainhash.Hash{2}] = append(signAndKeys, SignAndKey{chainhash.Hash64{5}, chainhash.Hash33{6}})

	for key, value := range ticketPool {
		fmt.Println("key:", key, ",value:", value)
	}
	fmt.Println(ticketPool[chainhash.Hash{2}])
	fmt.Println(chainhash.Hash{1, 2, 3, 5} == chainhash.Hash{1, 2, 3, 5})
}
