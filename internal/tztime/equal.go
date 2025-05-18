package tztime

import "time"

func EqualTime(a, b time.Time) bool {
	// aとbのタイムゾーンをUTCに変換してから比較
	return a.UTC().Truncate(time.Millisecond).Equal(b.UTC().Truncate(time.Millisecond))
}
