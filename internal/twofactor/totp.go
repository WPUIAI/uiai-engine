package twofactor

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"fmt"
	"hash"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Code struct {
	Code      string `json:"code"`
	ExpiresIn int64  `json:"expires_in"`
	Period    int    `json:"period"`
	Digits    int    `json:"digits"`
	Algorithm string `json:"algorithm"`
}

func Generate(secret, algorithm string, digits, period int, now time.Time) (Code, error) {
	if strings.TrimSpace(secret) == "" {
		return Code{}, fmt.Errorf("totp secret required")
	}
	if digits == 0 {
		digits = 6
	}
	if period == 0 {
		period = 30
	}
	if algorithm == "" {
		algorithm = "SHA1"
	}
	key, err := decodeBase32Secret(secret)
	if err != nil {
		return Code{}, err
	}
	counter := uint64(now.Unix() / int64(period))
	otp, err := hotp(key, counter, strings.ToUpper(algorithm), digits)
	if err != nil {
		return Code{}, err
	}
	return Code{Code: otp, ExpiresIn: int64(period) - (now.Unix() % int64(period)), Period: period, Digits: digits, Algorithm: strings.ToUpper(algorithm)}, nil
}

func FromOTPAuth(raw string, now time.Time) (Code, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Code{}, fmt.Errorf("parse otpauth URL: %w", err)
	}
	if u.Scheme != "otpauth" || u.Host != "totp" {
		return Code{}, fmt.Errorf("unsupported otpauth URL; only otpauth://totp is supported")
	}
	q := u.Query()
	digits, _ := strconv.Atoi(q.Get("digits"))
	period, _ := strconv.Atoi(q.Get("period"))
	return Generate(q.Get("secret"), q.Get("algorithm"), digits, period, now)
}

func decodeBase32Secret(secret string) ([]byte, error) {
	clean := strings.ToUpper(strings.Map(func(r rune) rune {
		switch r {
		case ' ', '-', '_':
			return -1
		default:
			return r
		}
	}, secret))
	if rem := len(clean) % 8; rem != 0 {
		clean += strings.Repeat("=", 8-rem)
	}
	key, err := base32.StdEncoding.DecodeString(clean)
	if err != nil {
		return nil, fmt.Errorf("decode totp secret: %w", err)
	}
	return key, nil
}

func hotp(key []byte, counter uint64, algorithm string, digits int) (string, error) {
	var newHash func() hash.Hash
	switch strings.ToUpper(algorithm) {
	case "", "SHA1":
		newHash = sha1.New
	case "SHA256":
		newHash = sha256.New
	case "SHA512":
		newHash = sha512.New
	default:
		return "", fmt.Errorf("unsupported totp algorithm %q", algorithm)
	}
	msg := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		msg[i] = byte(counter)
		counter >>= 8
	}
	mac := hmac.New(newHash, key)
	_, _ = mac.Write(msg)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	bin := (uint32(sum[offset])&0x7f)<<24 | (uint32(sum[offset+1])&0xff)<<16 | (uint32(sum[offset+2])&0xff)<<8 | (uint32(sum[offset+3]) & 0xff)
	mod := uint32(math.Pow10(digits))
	return fmt.Sprintf("%0*d", digits, bin%mod), nil
}
