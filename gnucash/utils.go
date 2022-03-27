package gnucash

import (
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

func NewGuid() string {
	return strings.Replace(uuid.NewString(), "-", "", -1)
}

func GetCurrentTimeString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func GncRationalToString(num int64, denom int64) string {
	rat := big.NewRat(num, denom)
	return rat.FloatString(2)
}
