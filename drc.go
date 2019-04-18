package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/drcsuite/drc/btcec"
	"github.com/drcsuite/drc/chaincfg"
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"github.com/drcsuite/drc/drcutil"
	"github.com/drcsuite/drc/wire"
	"time"
)

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

//func main() {
//	//wifKey, address, _ := GenerateBTCTest() // 测试地址
//	wifKey, address, _ := GenerateBTC() // 正式地址
//	fmt.Println(address, wifKey)
//}

// github.com/btcsuite/btcd/btcec/privkey.go
// NewPrivateKey is a wrapper for ecdsa.GenerateKey that returns a PrivateKey
// instead of the normal ecdsa.PrivateKey.
func NewPrivateKey(curve elliptic.Curve) (*btcec.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, err
	}
	return (*btcec.PrivateKey)(key), nil
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
	bytes, _ := privateKey.Sign64(hash)

	// plan 1
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

func MapStruct() {
	var incrementBlock map[int32]struct{}
	incrementBlock = make(map[int32]struct{})
	incrementBlock[1] = struct{}{}
	incrementBlock[2] = struct{}{}
	incrementBlock[3] = struct{}{}
	_, i := incrementBlock[1]
	_, i2 := incrementBlock[4]
	fmt.Println(i)
	fmt.Println(i2)
}

func IOREADER() {
	a := []byte{17, 0, 0, 0}
	sigwit := wire.MsgSigwit{}
	err := sigwit.BtcDecode(bytes.NewReader(a), 0, 0)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(sigwit.Height)
}

var genesisMerkleRoot = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0x3b, 0xa3, 0xed, 0xfd, 0x7a, 0x7b, 0x12, 0xb2,
	0x7a, 0xc7, 0x2c, 0x3e, 0x67, 0x76, 0x8f, 0x61,
	0x7f, 0xc8, 0x1b, 0xc3, 0x88, 0x8a, 0x51, 0x32,
	0x3a, 0x9f, 0xb8, 0xaa, 0x4b, 0x1e, 0x5e, 0x4a,
})

var genesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{},         // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: genesisMerkleRoot,        // 4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b
		Timestamp:  time.Unix(0x495fab29, 0), // 2009-01-03 18:15:05 +0000 UTC
		Signature:  chainhash.Hash64{},       // 00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
		PublicKey:  chainhash.Hash{},         // 000000000000000000000000000000000000000000000000000000000000000000
		Scale:      1,
		Reserved:   0,
	},
	Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
}

var genesisCoinbaseTx = wire.MsgTx{
	Version: 1,
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{},
				Index: 0xffffffff,
			},
			SignatureScript: []byte{
				0x04, 0xff, 0xff, 0x00, 0x1d, 0x01, 0x04, 0x45, /* |.......E| */
				0x54, 0x68, 0x65, 0x20, 0x54, 0x69, 0x6d, 0x65, /* |The Time| */
				0x73, 0x20, 0x30, 0x33, 0x2f, 0x4a, 0x61, 0x6e, /* |s 03/Jan| */
				0x2f, 0x32, 0x30, 0x30, 0x39, 0x20, 0x43, 0x68, /* |/2009 Ch| */
				0x61, 0x6e, 0x63, 0x65, 0x6c, 0x6c, 0x6f, 0x72, /* |ancellor| */
				0x20, 0x6f, 0x6e, 0x20, 0x62, 0x72, 0x69, 0x6e, /* | on brin| */
				0x6b, 0x20, 0x6f, 0x66, 0x20, 0x73, 0x65, 0x63, /* |k of sec|*/
				0x6f, 0x6e, 0x64, 0x20, 0x62, 0x61, 0x69, 0x6c, /* |ond bail| */
				0x6f, 0x75, 0x74, 0x20, 0x66, 0x6f, 0x72, 0x20, /* |out for |*/
				0x62, 0x61, 0x6e, 0x6b, 0x73, /* |banks| */
			},
			Sequence: 0xffffffff,
		},
	},
	TxOut: []*wire.TxOut{
		{
			Value: 0x12a05f200,
			PkScript: []byte{
				0x41, 0x04, 0x67, 0x8a, 0xfd, 0xb0, 0xfe, 0x55, /* |A.g....U| */
				0x48, 0x27, 0x19, 0x67, 0xf1, 0xa6, 0x71, 0x30, /* |H'.g..q0| */
				0xb7, 0x10, 0x5c, 0xd6, 0xa8, 0x28, 0xe0, 0x39, /* |..\..(.9| */
				0x09, 0xa6, 0x79, 0x62, 0xe0, 0xea, 0x1f, 0x61, /* |..yb...a| */
				0xde, 0xb6, 0x49, 0xf6, 0xbc, 0x3f, 0x4c, 0xef, /* |..I..?L.| */
				0x38, 0xc4, 0xf3, 0x55, 0x04, 0xe5, 0x1e, 0xc1, /* |8..U....| */
				0x12, 0xde, 0x5c, 0x38, 0x4d, 0xf7, 0xba, 0x0b, /* |..\8M...| */
				0x8d, 0x57, 0x8a, 0x4c, 0x70, 0x2b, 0x6b, 0xf1, /* |.W.Lp+k.| */
				0x1d, 0x5f, 0xac, /* |._.| */
			},
		},
	},
	LockTime: 0,
}

// 生成创世区块
func genesisBLock() {
	genesisBlock := drcutil.NewBlock(&genesisBlock)
	genesisBlock.SetHeight(0)
	header := &genesisBlock.MsgBlock().Header
	hash := header.BlockHash()
	fmt.Println(hash)
	fmt.Println(hash.CloneBytes())
	for i := 0; i < len(hash); i++ {
		b := hash[i]
		fmt.Printf("%#x,", b)
	}
	//for i := len(hash) - 1; i >= 0; i-- {
	//	b := hash[i]
	//	fmt.Printf("%#x,", b)
	//}
}

//func main() {
//	//testCipherAndSign()
//	//ptr()
//	//ptrint()
//	//GenesisBlock()
//	genesisBLock()
//
//	//Array()
//	//MapStruct()
//	//IOREADER()
//	fmt.Println(" Hello, DRC!")
//}
