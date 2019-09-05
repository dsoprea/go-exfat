package exfat

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"crypto/sha1"

	"github.com/dsoprea/go-logging"
)

// TODO(dustin): !! Add static get-data test to ExfatReader for known file cluster number.

func TestExfatNavigator_Dump(t *testing.T) {
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

	index, _, _, err := en.IndexDirectoryEntries()
	log.PanicIf(err)

	index.Dump()
}

func TestExfatNavigator__GetFileData(t *testing.T) {
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

	index, _, _, err := en.IndexDirectoryEntries()
	log.PanicIf(err)

	// Get data.

	sede := index.FindIndexedFileStreamExtensionDirectoryEntry("2-delahaye-type-165-cabriolet-dsc_8025.jpg")

	h := sha1.New()

	useFat := sede.GeneralSecondaryFlags.NoFatChain() == false

	err = er.WriteFromClusterChain(sede.FirstCluster, sede.ValidDataLength, useFat, h)
	log.PanicIf(err)

	digest := h.Sum(nil)
	digestString := fmt.Sprintf("%040x", digest)

	expectedString := "a2219fa800ae2325003d8d4f5122b37f12f1e18e"
	if digestString != expectedString {
		t.Fatalf("Data not recovered correctly: [%s] != [%s]", digestString, expectedString)
	}
}

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

	index, _, _, err := en.IndexDirectoryEntries()
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
		"testdirectory",
		"8fd71ab132c59bf33cd7890c0acebf12.jpg",
		"testdirectory2",
		"064cbfd4-cec3-11e9-926d-c362c80fab7b",
		"testdirectory3",
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

func TestExfatNavigator__NavigateSubdirectory(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	firstClusterNumber := er.FirstClusterOfRootDirectory()
	en := NewExfatNavigator(er, firstClusterNumber)

	index, _, _, err := en.IndexDirectoryEntries()
	log.PanicIf(err)

	sede := index.FindIndexedFileStreamExtensionDirectoryEntry("testdirectory")
	subfolderEn := NewExfatNavigator(er, sede.FirstCluster)

	subfolderIndex, _, _, err := subfolderEn.IndexDirectoryEntries()
	log.PanicIf(err)

	expectedFilenames := map[string]bool{
		"300daec8-cec3-11e9-bfa2-0f240e41d1d8": false,
	}

	if reflect.DeepEqual(subfolderIndex.Filenames(), expectedFilenames) != true {
		t.Fatalf("Subdirectory filenames not correct: %v != %v", subfolderIndex.Filenames(), expectedFilenames)
	}
}

func TestDirectoryEntryIndex_Filenames(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	firstClusterNumber := er.FirstClusterOfRootDirectory()
	en := NewExfatNavigator(er, firstClusterNumber)

	index, _, _, err := en.IndexDirectoryEntries()
	log.PanicIf(err)

	filenames := index.Filenames()

	expectedFilenames := map[string]bool{
		"testdirectory":  true,
		"testdirectory2": true,
		"testdirectory3": true,
		"2-delahaye-type-165-cabriolet-dsc_8025.jpg": false,
		"8fd71ab132c59bf33cd7890c0acebf12.jpg":       false,
		"064cbfd4-cec3-11e9-926d-c362c80fab7b":       false,
		"79c6d31a-cca1-11e9-8325-9746d045e868":       false,
	}

	if reflect.DeepEqual(filenames, expectedFilenames) != true {
		t.Fatalf("Filenames not correct: %v != %v", filenames, expectedFilenames)
	}
}

func TestDirectoryEntryIndex_FileCount(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	firstClusterNumber := er.FirstClusterOfRootDirectory()
	en := NewExfatNavigator(er, firstClusterNumber)

	index, _, _, err := en.IndexDirectoryEntries()
	log.PanicIf(err)

	if index.FileCount() != 7 {
		t.Fatalf("File-count not correct: (%d)", index.FileCount())
	}
}

func TestDirectoryEntryIndex_GetFile(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	firstClusterNumber := er.FirstClusterOfRootDirectory()
	en := NewExfatNavigator(er, firstClusterNumber)

	index, _, _, err := en.IndexDirectoryEntries()
	log.PanicIf(err)

	files := make([]string, index.FileCount())
	for i := 0; i < index.FileCount(); i++ {
		files[i], _ = index.GetFile(i)
	}

	expectedFiles := []string{
		"79c6d31a-cca1-11e9-8325-9746d045e868",
		"2-delahaye-type-165-cabriolet-dsc_8025.jpg",
		"testdirectory",
		"8fd71ab132c59bf33cd7890c0acebf12.jpg",
		"testdirectory2",
		"064cbfd4-cec3-11e9-926d-c362c80fab7b",
		"testdirectory3",
	}

	if reflect.DeepEqual(files, expectedFiles) != true {
		t.Fatalf("Files not correct: %v != %v", files, expectedFiles)
	}
}

func TestDirectoryEntryIndex_FindIndexedFile(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	firstClusterNumber := er.FirstClusterOfRootDirectory()
	en := NewExfatNavigator(er, firstClusterNumber)

	index, _, _, err := en.IndexDirectoryEntries()
	log.PanicIf(err)

	for i := 0; i < index.FileCount(); i++ {
		filename, _ := index.GetFile(i)

		ide, found := index.FindIndexedFile(filename)
		if found != true {
			t.Fatalf("File not found: [%s]", filename)
		}

		foundFilename := ide.Extra["complete_filename"].(string)
		if foundFilename != filename {
			t.Fatalf("Found entry not correct: [%s] != [%s]", foundFilename, filename)
		}
	}
}

func TestDirectoryEntryIndex_FindIndexedFileFileDirectoryEntry(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	firstClusterNumber := er.FirstClusterOfRootDirectory()
	en := NewExfatNavigator(er, firstClusterNumber)

	index, _, _, err := en.IndexDirectoryEntries()
	log.PanicIf(err)

	for i := 0; i < index.FileCount(); i++ {
		filename, expectedFdf := index.GetFile(i)

		actualFdf := index.FindIndexedFileFileDirectoryEntry(filename)

		if actualFdf != expectedFdf {
			t.Fatalf("FDF for entry (%d) [%s] not correct.", i, filename)
		}
	}
}

func TestDirectoryEntryIndex_FindIndexedFileStreamExtensionDirectoryEntry(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	firstClusterNumber := er.FirstClusterOfRootDirectory()
	en := NewExfatNavigator(er, firstClusterNumber)

	index, _, _, err := en.IndexDirectoryEntries()
	log.PanicIf(err)

	sede := index.FindIndexedFileStreamExtensionDirectoryEntry("2-delahaye-type-165-cabriolet-dsc_8025.jpg")
	if sede.FirstCluster != 7 {
		t.Fatalf("Stream-extension entry-type not found: (%d)", sede.FirstCluster)
	}
}
