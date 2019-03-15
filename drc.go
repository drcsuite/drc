package main

import (
	"fmt"
	"github.com/drcsuite/drc/btcec"
)

func main() {
	//testCipherAndSign()
	ptr()
	//ptrint()
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
