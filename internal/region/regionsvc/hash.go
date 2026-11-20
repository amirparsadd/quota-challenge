package regionsvc

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"quota-challenge/internal/capacity"
)

func HashReserve(product string, items []capacity.ResourceLine, ttl int64) string {
	type line struct {
		Resource string `json:"resource"`
		Amount   int64  `json:"amount"`
	}
	cp := make([]line, len(items))
	for i, it := range items {
		cp[i] = line{Resource: it.Resource, Amount: it.Amount}
	}
	sort.Slice(cp, func(i, j int) bool { return cp[i].Resource < cp[j].Resource })
	payload := struct {
		Product string `json:"product"`
		Items   []line `json:"items"`
		TTL     int64  `json:"ttl_seconds"`
	}{Product: product, Items: cp, TTL: ttl}
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}