package exfat

import (
	"os"
	"path"
	"testing"
)

func TestNewBootSectorHeaderFromReader(t *testing.T) {
	filepath := path.Join(AssetPath, "test.exfat")
	f, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	bsh, err := NewBootSectorHeaderFromReader(f)
	if err != nil {
		panic(err)
	}

	if bsh.VolumeSerialNumber != 0x3d51a058 {
		t.Fatalf("volume serial-number not correct: 0x%x", bsh.VolumeSerialNumber)
	}
}

func TestBootSectorHeader_Dump(t *testing.T) {
	filepath := path.Join(AssetPath, "test.exfat")
	f, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	bsh, err := NewBootSectorHeaderFromReader(f)
	if err != nil {
		panic(err)
	}

	bsh.Dump()
}
