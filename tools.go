package main

import (
	"encoding/binary"
	"log/slog"
	"math"
	"math/big"
	"os"
	"os/signal"

	"github.com/anatol/smart.go"
)

const max_uint64 = float64(math.MaxUint64)

func Uint128toFloat64(in smart.Uint128) float64 {
	return float64(in.Val[0]) + max_uint64*float64(in.Val[1])
}

func bigFromInt128(int128 smart.Uint128) *big.Int {
	b := make([]byte, 16)
	binary.BigEndian.PutUint64(b[:8], int128.Val[1])
	binary.BigEndian.PutUint64(b[8:], int128.Val[0])

	result := new(big.Int).SetBytes(b)
	return result
}

func SignalsCallback(cb func(), once bool, sigs ...os.Signal) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, sigs...)
	go func() {
		for {
			sig := <-c
			slog.Debug("recived signal", "sig", sig)
			cb()
			if once {
				break
			}
		}
	}()
}
