package exfat

import (
	"unicode/utf8"
)

func UnicodeFromAscii(raw []byte, unicodeCharCount int) string {
	// `VolumeLabel` is a Unicode-encoded string and the character-count
	// corresponds to the number of Unicode characters.

	decodedString := make([]rune, unicodeCharCount)
	realSize := 0
	for i := 0; i < unicodeCharCount; i++ {
		r, _ := utf8.DecodeRune(raw[i*2 : i*2+1])

		// NUL. We're probably in the extra space as the end of the last part.
		if r == 0 {
			continue
		}

		decodedString[i] = r
		realSize++
	}

	decodedString = decodedString[:realSize]

	return string(decodedString)
}
