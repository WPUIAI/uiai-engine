package captcha

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"math/big"
)

func secureIntn(max int) int {
	if max <= 0 {
		return 0
	}
	n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

func secureFloat64(max float64) float64 {
	if max <= 0 {
		return 0
	}
	var buf [8]byte
	if _, err := cryptorand.Read(buf[:]); err != nil {
		return 0
	}
	unit := float64(binary.BigEndian.Uint64(buf[:])) / float64(^uint64(0))
	return unit * max
}
