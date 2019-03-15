package votingsys

import (
	"fmt"
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
