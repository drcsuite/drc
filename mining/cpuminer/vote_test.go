package cpuminer

import (
	"fmt"
	"github.com/drcsuite/drc/btcec"
	"github.com/drcsuite/drc/chaincfg"
	"github.com/drcsuite/drc/drcutil"
	"github.com/drcsuite/drc/vote"
	"testing"
	"time"
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
	fmt.Println(vote.VotesVerge(257))
	fmt.Println(vote.BlockVerge(257))
	fmt.Println(10*time.Second*time.Duration(6) + 20*time.Second)
}

func GenerateBTC() (string, string, error) {
	privKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return "", "", err
	}

	privKeyWif, err := drcutil.NewWIF(privKey, &chaincfg.MainNetParams, false)
	if err != nil {
		return "", "", err
	}
	pubKeySerial := privKey.PubKey().SerializeUncompressed()

	pubKeyAddress, err := drcutil.NewAddressPubKey(pubKeySerial, &chaincfg.MainNetParams)
	if err != nil {
		return "", "", err
	}

	return privKeyWif.String(), pubKeyAddress.EncodeAddress(), nil
}

func TestDi(t *testing.T) {
	wifKey, address, _ := GenerateBTC()
	fmt.Println(address, wifKey)
}

func TestIsEnough(t *testing.T) {
	fmt.Println(IsEnough(1, 1))
	fmt.Println(IsEnough(2, 2))
	fmt.Println(IsEnough(2, 3))
	fmt.Println(IsEnough(3, 4))
	fmt.Println(IsEnough(4, 5))
	fmt.Println(IsEnough(133, 200))
	fmt.Println(IsEnough(134, 200))

	fmt.Println("_____________________________________")
	fmt.Println(IsEnough(199, 500))
	fmt.Println(IsEnough(200, 500))
	fmt.Println(IsEnough(201, 500))

}
