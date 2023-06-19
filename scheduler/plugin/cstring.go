package wasm

func fromNULTerminated(b []byte) (entries []string) {
	if len(b) == 0 {
		return
	}
	entry, entryPos := 0, 0
	for size := len(b); size > 0; size-- {
		if b[entryPos] == 0 { // then, we reached the end of the field.
			if entry != entryPos { // read non-empty
				entries = append(entries, string(b[entry:entryPos]))
			}
			entryPos++
			entry = entryPos
		} else {
			entryPos++
		}
	}
	return
}
