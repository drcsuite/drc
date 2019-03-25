// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package drcutil

const (
	// SatoshiPerBitcent is the number of satoshi in one bitcoin cent.
	SatoshiPerBitcent = 1e6

	// SatoshiPerBitcoin is the number of satoshi in one bitcoin (1 BTC).
	// SatoshiPerBitcoin是一个比特币(1 BTC)中satoshi的数量。
	SatoshiPerBitcoin = 1e8

	// MaxSatoshi is the maximum transaction amount allowed in satoshi.
	// MaxSatoshi是satoshi中允许的最大交易金额。2100万
	MaxSatoshi = 21e6 * SatoshiPerBitcoin
)
