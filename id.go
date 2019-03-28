package main

import "math/rand"
import "fmt"
import "time"
func GUID() string {
	return fmt.Sprintf("%016x%016x", time.Now().UnixNano(), rand.Uint64())
}
