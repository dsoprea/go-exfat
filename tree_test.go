package exfat

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/dsoprea/go-logging"
)

func TestTree_List(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	files, nodes, err := tree.List()
	log.PanicIf(err)

	// Check filenames.

	expectedFiles := []string{
		"testdirectory",
		"testdirectory\\300daec8-cec3-11e9-bfa2-0f240e41d1d8",
		"testdirectory2",
		"testdirectory2\\00c57ab0-cec3-11e9-b750-bbed8d2244c8",
		"testdirectory2\\ff7b94be-cec2-11e9-b7b1-6b2e61bd775c",
		"testdirectory2\\file1",
		"testdirectory2\\file2",
		"testdirectory3",
		"testdirectory3\\10422c86-cec3-11e9-953f-4f501efd2640",
		"064cbfd4-cec3-11e9-926d-c362c80fab7b",
		"2-delahaye-type-165-cabriolet-dsc_8025.jpg",
		"79c6d31a-cca1-11e9-8325-9746d045e868",
		"8fd71ab132c59bf33cd7890c0acebf12.jpg",
	}

	if reflect.DeepEqual(files, expectedFiles) != true {
		for i, filePath := range files {
			fmt.Printf("ACTUAL: (%d) [%s]\n", i, filePath)
		}

		for i, filePath := range expectedFiles {
			fmt.Printf("EXPECTED: (%d) [%s]\n", i, filePath)
		}

		t.Fatalf("Files not correct.")
	}

	// Check nodes.

	actualTypes := make(map[string]bool)

	for path, node := range nodes {
		actualTypes[path] = node.IsDirectory()
	}

	expectedTypes := map[string]bool{
		"testdirectory": true,
		"testdirectory\\300daec8-cec3-11e9-bfa2-0f240e41d1d8": false,
		"testdirectory2":        true,
		"testdirectory2\\file1": false,
		"testdirectory2\\file2": false,
		"testdirectory2\\ff7b94be-cec2-11e9-b7b1-6b2e61bd775c": false,
		"testdirectory2\\00c57ab0-cec3-11e9-b750-bbed8d2244c8": false,
		"testdirectory3": true,
		"testdirectory3\\10422c86-cec3-11e9-953f-4f501efd2640": false,
		"8fd71ab132c59bf33cd7890c0acebf12.jpg":                 false,
		"064cbfd4-cec3-11e9-926d-c362c80fab7b":                 false,
		"79c6d31a-cca1-11e9-8325-9746d045e868":                 false,
		"2-delahaye-type-165-cabriolet-dsc_8025.jpg":           false,
	}

	if reflect.DeepEqual(actualTypes, expectedTypes) != true {
		t.Fatalf("File-entry types not correct.")
	}
}

func TestTree_Lookup__Hit(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node := tree.Lookup([]string{"testdirectory2", "ff7b94be-cec2-11e9-b7b1-6b2e61bd775c"})
	if node.Name() != "ff7b94be-cec2-11e9-b7b1-6b2e61bd775c" {
		t.Fatalf("Found node not correct (hit).")
	}
}

func TestTree_Lookup__Miss(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node := tree.Lookup([]string{"invalid", "path"})
	if node != nil {
		t.Fatalf("Found node not correct (miss).")
	}
}
