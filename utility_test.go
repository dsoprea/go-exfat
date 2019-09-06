package exfat

import (
	"testing"
)

func TestUnicodeFromAscii(t *testing.T) {
	b := []byte{'a', 0, 'b', 0, 'c', 0, 'd', 0, 'e', 0}
	s := UnicodeFromAscii(b, 3)

	if s != "abc" {
		t.Fatalf("Ascii not decoded to Unicode correctly.")
	}
}
