package exfat

import (
	"testing"

	"github.com/dsoprea/go-logging"
)

func TestExfatNavigator_EnumerateDirectoryEntries(t *testing.T) {
	defer func() {
		if errRaw := recover(); errRaw != nil {
			err := errRaw.(error)

			log.PrintError(err)
			t.Fatalf("Test failed.")
		}
	}()

	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	firstClusterNumber := er.FirstClusterOfRootDirectory()
	en := NewExfatNavigator(er, firstClusterNumber)

	err = en.EnumerateDirectoryEntries()
	log.PanicIf(err)

	// primaryEntry = primaryEntry
	// secondaryEntries = secondaryEntries

	// er.bootRegion.bsh.Dump()
	// primaryEntry.Dump()

	// for i, sde := range secondaryEntries {
	// 	fmt.Printf("SE (%d): %s\n", i, sde)
	// }
}
