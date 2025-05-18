package tztime

import "time"

func Now() time.Time {
	return time.Now().In(Zone())
}
