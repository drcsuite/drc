package votingsys

import (
	"fmt"
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"testing"
)

func TestEstimateScale(t *testing.T) {
	// 节点数少于300，上次收到的票数与实际节点数差不多，最新的节点数为47
	fmt.Println(EstimateScale(47, 50))
	// 投票比300少，说明之前节点估计得多了，最新的节点数要比之前的少，为4666
	fmt.Println(EstimateScale(280, 5000))
	// 投票比300多，说明之前节点估计得少了，最新的节点数要比之前的大，为5333
	fmt.Println(EstimateScale(320, 5000))
}

func TestBlockVerge(t *testing.T) {

	fmt.Println(VoteVerge(257))
	fmt.Println(BlockVerge(257))
}

func TestTicketPool(t *testing.T) {
	var signAndKey []SignAndKey
	signAndKeys := append(signAndKey, SignAndKey{chainhash.Hash64{1}, chainhash.Hash33{2}})
	TicketPool[chainhash.Hash{1}] = append(signAndKeys, SignAndKey{chainhash.Hash64{3}, chainhash.Hash33{4}})
	TicketPool[chainhash.Hash{2}] = append(signAndKeys, SignAndKey{chainhash.Hash64{5}, chainhash.Hash33{6}})

	for key, value := range TicketPool {
		fmt.Println("key:", key, ",value:", value)
	}
	fmt.Println(TicketPool[chainhash.Hash{2}])
	fmt.Println(chainhash.Hash{1, 2, 3, 5} == chainhash.Hash{1, 2, 3, 5})
}
