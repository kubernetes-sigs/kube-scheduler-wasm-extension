package test

import "fmt"

// CapturePanic returns what was recovered from a panic as a string.
//
// Note: This was originally adapted from wazero require.CapturePanic
func CapturePanic(panics func()) (captured string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			captured = fmt.Sprintf("%v", recovered)
		}
	}()
	panics()
	return
}
