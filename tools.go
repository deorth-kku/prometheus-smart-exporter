package main

import (
	"math"

	"github.com/anatol/smart.go"
)

const max_uint64 = float64(math.MaxUint64)

func Uint128toFloat64(in smart.Uint128) float64 {
	return float64(in.Val[0]) + max_uint64*float64(in.Val[1])
}
