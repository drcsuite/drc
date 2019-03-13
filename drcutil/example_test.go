package drcutil_test

import (
	"fmt"
	"math"

	"github.com/drcsuite/drc/drcutil"
)

func ExampleAmount() {

	a := drcutil.Amount(0)
	fmt.Println("Zero Satoshi:", a)

	a = drcutil.Amount(1e8)
	fmt.Println("100,000,000 Satoshis:", a)

	a = drcutil.Amount(1e5)
	fmt.Println("100,000 Satoshis:", a)
	// Output:
	// Zero Satoshi: 0 BTC
	// 100,000,000 Satoshis: 1 BTC
	// 100,000 Satoshis: 0.001 BTC
}

func ExampleNewAmount() {
	amountOne, err := drcutil.NewAmount(1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountOne) //Output 1

	amountFraction, err := drcutil.NewAmount(0.01234567)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountFraction) //Output 2

	amountZero, err := drcutil.NewAmount(0)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountZero) //Output 3

	amountNaN, err := drcutil.NewAmount(math.NaN())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountNaN) //Output 4

	// Output: 1 BTC
	// 0.01234567 BTC
	// 0 BTC
	// invalid bitcoin amount
}

func ExampleAmount_unitConversions() {
	amount := drcutil.Amount(44433322211100)

	fmt.Println("Satoshi to kBTC:", amount.Format(drcutil.AmountKiloBTC))
	fmt.Println("Satoshi to BTC:", amount)
	fmt.Println("Satoshi to MilliBTC:", amount.Format(drcutil.AmountMilliBTC))
	fmt.Println("Satoshi to MicroBTC:", amount.Format(drcutil.AmountMicroBTC))
	fmt.Println("Satoshi to Satoshi:", amount.Format(drcutil.AmountSatoshi))

	// Output:
	// Satoshi to kBTC: 444.333222111 kBTC
	// Satoshi to BTC: 444333.222111 BTC
	// Satoshi to MilliBTC: 444333222.111 mBTC
	// Satoshi to MicroBTC: 444333222111 μBTC
	// Satoshi to Satoshi: 44433322211100 Satoshi
}
