package database

import (
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/fiatjaf/levelup"
	"github.com/fiatjaf/summadb/types"
	"github.com/mgutz/logxi/v1"
)

func bumpRev(rev string) string {
	spl := strings.Split(rev, "-")
	v, _ := strconv.Atoi(spl[0])
	v++
	return strconv.Itoa(v) + "-" + randomString(4)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func randomString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b)
}

func (db *SummaDB) checkRev(providedrev string, p types.Path) error {
	currentrev, err := db.Get(p.Child("_rev").Join())
	if err == levelup.NotFound && providedrev == "" {
		return nil
	}
	if err != nil {
		log.Error("failed to fetch rev for checking.",
			"path", p,
			"provided", providedrev)
		return err
	}
	if currentrev == providedrev {
		return nil
	}
	return errors.New(
		"mismatching revs at " + p.Join() + ". current: " + currentrev + "; provided: " + providedrev)
}
