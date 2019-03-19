package main

import (
	"encoding/hex"
	"fmt"
	"github.com/drcsuite/drc/btcec"
)

func main() {
	//testCipherAndSign()
	//ptr()
	//ptrint()
	//GenesisBlock()

	Array()
	fmt.Println(" Hello, DRC!")
}

func testCipherAndSign() {
	// cipher
	privKey, _ := btcec.NewPrivateKey(btcec.S256())
	fmt.Println(privKey)
	serialize := privKey.Serialize()
	fmt.Println(len(serialize))
	privateKey, publicKey := btcec.PrivKeyFromBytes(btcec.S256(), serialize)
	fmt.Println("公钥序列化前： ", publicKey)
	pkcomp := publicKey.SerializeCompressed()
	fmt.Println("公钥压缩后长度： ", len(pkcomp))
	pubKey, _ := btcec.ParsePubKey(pkcomp, btcec.S256())
	fmt.Println("公钥反序列化后： ", pubKey)

	// signature
	hash := []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9}
	signature, _ := privateKey.Sign(hash)

	// plan 1
	bytes := signature.GenSignBytes()
	fmt.Println("长度： ", len(bytes), "签名bytes： ", bytes)
	derSignature := btcec.GetSignature(bytes)

	// plan 2
	//ss := signature.Serialize()
	//fmt.Println("序列化签名为： ", signature, "长度： ", len(ss))
	//derSignature, _ := btcec.ParseDERSignature(ss, btcec.S256())

	verify := derSignature.Verify(hash, publicKey)
	fmt.Println(verify)

}

func ptr() {
	a := people{
		name: "asdf",
		age:  100,
	}
	p := &a
	fmt.Println(&p)
	p.age = 50
	fmt.Println(&p)
	fmt.Println(a)

	(*p).age = 150
	fmt.Println(p)
	fmt.Println(a)

}

func ptrint() {
	//a := people{
	//	name: "asdf",
	//	age:  100,
	//}
	a := "adsf"
	fmt.Println(&a)
	//a.age = 200
	a = "fdsa"
	fmt.Println(&a)

}

type people struct {
	name string
	age  int
}

func GenesisBlock() {
	PkScript := []byte{
		0x41, 0x04, 0x67, 0x8a, 0xfd, 0xb0, 0xfe, 0x55, /* |A.g....U| */
		0x48, 0x27, 0x19, 0x67, 0xf1, 0xa6, 0x71, 0x30, /* |H'.g..q0| */
		0xb7, 0x10, 0x5c, 0xd6, 0xa8, 0x28, 0xe0, 0x39, /* |..\..(.9| */
		0x09, 0xa6, 0x79, 0x62, 0xe0, 0xea, 0x1f, 0x61, /* |..yb...a| */
		0xde, 0xb6, 0x49, 0xf6, 0xbc, 0x3f, 0x4c, 0xef, /* |..I..?L.| */
		0x38, 0xc4, 0xf3, 0x55, 0x04, 0xe5, 0x1e, 0xc1, /* |8..U....| */
		0x12, 0xde, 0x5c, 0x38, 0x4d, 0xf7, 0xba, 0x0b, /* |..\8M...| */
		0x8d, 0x57, 0x8a, 0x4c, 0x70, 0x2b, 0x6b, 0xf1, /* |.W.Lp+k.| */
		0x1d, 0x5f, 0xac, /* |._.| */
	}
	s := hex.EncodeToString(PkScript)
	fmt.Println(s)
}

func Array() {
	bytes := make([]byte, 0)
	i := append(bytes, 1, 2, 3)
	fmt.Println(len(i))
}
