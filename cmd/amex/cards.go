package main

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/saucesteals/amex"
)

type CardWriter struct {
	f    *os.File
	path string
}

func cleanName(name string) string {
	var result strings.Builder
	for _, r := range name {
		if r <= 'z' && r >= 'a' || r <= '9' && r >= '0' {
			result.WriteRune(r)
		} else if r >= 'A' && r <= 'Z' {
			result.WriteRune(r + 32)
		} else if r == ' ' || r == '_' || r == '-' {
			result.WriteRune('_')
		}
	}

	return result.String()
}

func NewCardWriter(profile *Profile, card amex.EligibleCard, prefix string) (*CardWriter, error) {
	dir, err := profile.GetDirectory(
		"cards",
		cleanName(card.Product.ProductName),
	)
	if err != nil {
		return nil, err
	}

	t := time.Now().Format("2006_01_02_15_04_05")
	fileName := path.Join(dir, fmt.Sprintf("%s_%s.csv", cleanName(prefix), t))
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}

	return &CardWriter{f: f, path: fileName}, nil
}

func (w *CardWriter) GetPath() string {
	return w.path
}

func (w *CardWriter) Write(vcc amex.VirtualCard) error {
	expirationParts := strings.Split(vcc.ExpiryYearMonth, "-")
	if len(expirationParts) != 2 {
		return fmt.Errorf("invalid expiration date: %s", vcc.ExpiryYearMonth)
	}

	expYear, err := strconv.Atoi(expirationParts[0])
	if err != nil {
		return err
	}

	expMonth, err := strconv.Atoi(strings.TrimPrefix(expirationParts[1], "0"))
	if err != nil {
		return err
	}

	if expYear < 100 {
		expYear += 2000
	}

	line := fmt.Sprintf("%s,%d,%d,%s", vcc.VirtualCardNumber, expMonth, expYear, vcc.SecurityCode)
	_, err = w.f.Write([]byte(line + "\n"))
	return err
}

func (w *CardWriter) Close() error {
	return w.f.Close()
}
