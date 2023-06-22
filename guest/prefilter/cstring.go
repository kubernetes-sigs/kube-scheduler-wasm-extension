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

package prefilter

func toNULTerminated(input []string) []byte {
	count := uint32(len(input))
	if count == 0 {
		return nil
	}

	size := count // NUL terminator count
	for _, s := range input {
		size += uint32(len(s))
	}

	// Write the NUL-terminated string to a byte slice.
	cStrings := make([]byte, size)
	pos := 0
	for _, s := range input {
		if len(s) == 0 {
			size--
			continue // skip empty
		}
		copy(cStrings[pos:], s)
		pos += len(s) + 1 // +1 for NUL-terminator
	}
	return cStrings[:size]
}
