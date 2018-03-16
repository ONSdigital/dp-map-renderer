package health

import (
	"time"
	"fmt"
	"sort"
)

// TrackTime logs the time taken by the method. Usage - as the first line in a method: defer health.TrackTime(time.Now(), "methodName")
func TrackTime(start time.Time, name string) {
	elapsed := time.Since(start)
	fmt.Println(name, "took ", elapsed.Round(time.Millisecond), "ms")
}

// this is not going to be thread-safe. It assumes that all calls will be sequential (I can guarantee this working locally)
// If we need to keep this, I'd suggest replacing it with something thread-safe: https://github.com/cornelk/hashmap
var elapsedMap = make(map[string]int64)
var invocationMap = make(map[string]int64)

func RecordTime(start time.Time, name string) {
	elapsed := time.Since(start)
	elapsedMap[name] = elapsedMap[name] + elapsed.Nanoseconds()
	invocationMap[name] = invocationMap[name] + 1
}

func LogTime() {
	names := make([]string, len(invocationMap))
	i := 0
	for name, _ := range invocationMap {
		names[i] = name
		i++
	}
	sort.Strings(names)
	for _, name := range names {
		elapsed := elapsedMap[name]  / 1000000
		fmt.Println(name, "took ", elapsed, "ms", " over", invocationMap[name], "invocations")
	}
	elapsedMap = make(map[string]int64)
	invocationMap = make(map[string]int64)
}