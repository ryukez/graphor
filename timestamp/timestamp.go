package timestamp

import (
	"time"
)

func Now() int {
	return ToTimestamp(time.Now())
}

func Empty() int {
	return 0
}

func ToTimestamp(t time.Time) int {
	return int(t.UnixNano() / 10e5)
}
