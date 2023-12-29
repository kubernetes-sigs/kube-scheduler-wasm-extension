/*
   Copyright 2023 The Kubernetes Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package clusterevent

import "sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"

// sizeEncodedClusterEvent is the size in bytes to encode
// framework.ClusterEvent with 32-bit little endian gvk and ActionType
const sizeEncodedClusterEvent = 4 + 4

func EncodeClusterEvents(input []api.ClusterEvent) []byte {
	size := len(input) * sizeEncodedClusterEvent
	if size == 0 {
		return nil // don't allocate an empty slice
	}

	// Encode the events to a byte slice.
	encoded := make([]byte, size)
	pos := 0
	for i := range input {
		e := input[i]
		// Write Resource in little endian encoding.
		encoded[pos+0] = byte(e.Resource)
		encoded[pos+1] = byte(e.Resource >> 8)
		encoded[pos+2] = byte(e.Resource >> 16)
		encoded[pos+3] = byte(e.Resource >> 24)

		// Write ActionType in little endian encoding.
		encoded[pos+4] = byte(e.ActionType)
		encoded[pos+5] = byte(e.ActionType >> 8)
		encoded[pos+6] = byte(e.ActionType >> 16)
		encoded[pos+7] = byte(e.ActionType >> 24)
		pos += sizeEncodedClusterEvent
	}
	return encoded
}
