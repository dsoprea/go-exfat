package exfat

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/dsoprea/go-logging"
)

// func TestExfatNavigator_EnumerateDirectoryEntries(t *testing.T) {
// 	defer func() {
// 		if errRaw := recover(); errRaw != nil {
// 			err := errRaw.(error)

// 			log.PrintError(err)
// 			t.Fatalf("Test failed.")
// 		}
// 	}()

// 	f, er := getTestFileAndParser()

// 	defer f.Close()

// 	err := er.Parse()
// 	log.PanicIf(err)

// 	firstClusterNumber := er.FirstClusterOfRootDirectory()
// 	en := NewExfatNavigator(er, firstClusterNumber)

// 	cb := func(primaryEntry DirectoryEntry, secondaryEntries []DirectoryEntry) (err error) {
// 		fmt.Printf("[%s] %s\n", reflect.TypeOf(primaryEntry), primaryEntry)

// 		if len(secondaryEntries) > 0 {
// 			for i, de := range secondaryEntries {
// 				fmt.Printf("> (%d) %s\n", i, de)
// 			}

// 			fmt.Printf("\n")
// 		}

// 		if _, ok := primaryEntry.(*ExfatFileDirectoryEntry); ok == true {
// 			mf := MultipartFilename(secondaryEntries)
// 			filename := mf.Filename()

// 			fmt.Printf("Filename: [%s]\n", filename)
// 			fmt.Printf("\n")
// 		}

// 		return nil
// 	}

// 	err = en.EnumerateDirectoryEntries(cb)
// 	log.PanicIf(err)

// 	// primaryEntry = primaryEntry
// 	// secondaryEntries = secondaryEntries

// 	// er.bootRegion.bsh.Dump()
// 	// primaryEntry.Dump()

// 	// for i, sde := range secondaryEntries {
// 	// 	fmt.Printf("SE (%d): %s\n", i, sde)
// 	// }
// }

func TestExfatNavigator_IndexDirectoryEntries(t *testing.T) {
	defer func() {
		if errRaw := recover(); errRaw != nil {
			err := errRaw.(error)

			log.PrintError(err)
			t.Fatalf("Test failed.")
		}
	}()

	// Setup.

	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	firstClusterNumber := er.FirstClusterOfRootDirectory()
	en := NewExfatNavigator(er, firstClusterNumber)

	// Get index.

	index, err := en.IndexDirectoryEntries()
	log.PanicIf(err)

	if len(index) != 4 {
		t.Fatalf("Number of entries not correct: (%d)", len(index))
	}

	// Check types of directory-entries.

	typeNames := make([]string, len(index))
	i := 0
	for typeName, _ := range index {
		typeNames[i] = typeName
		i++
	}

	sort.StringSlice(typeNames).Sort()

	expectedTypeNames := []string{
		"AllocationBitmap",
		"File",
		"UpcaseTable",
		"VolumeLabel",
	}

	if reflect.DeepEqual(typeNames, expectedTypeNames) != true {
		t.Fatalf("Directory-entries not correct types: %v != %v", typeNames, expectedTypeNames)
	}

	// Check volume label.

	volumeLabel := index["VolumeLabel"][0].PrimaryEntry.(*ExfatVolumeLabelDirectoryEntry).Label()
	if volumeLabel != "testvolumelabel" {
		t.Fatalf("Volume label not correct: [%s]", volumeLabel)
	}

	// Check file entries.

	files := make([]string, len(index["File"]))

	for i, ide := range index["File"] {
		files[i] = ide.Extra["complete_filename"].(string)
	}

	expectedFilenames := []string{
		"79c6d31a-cca1-11e9-8325-9746d045e868",
		"2-delahaye-type-165-cabriolet-dsc_8025.jpg",
	}

	if reflect.DeepEqual(files, expectedFilenames) != true {
		for i, filename := range files {
			fmt.Printf("ACTUAL: (%d) (%d) [%s]\n", i, len(filename), filename)
		}

		for i, filename := range expectedFilenames {
			fmt.Printf("EXPECTED (%d): (%d) [%s]\n", i, len(filename), filename)
		}

		t.Fatalf("Root filenames not correct: %v != %v", files, expectedFilenames)
	}
}
