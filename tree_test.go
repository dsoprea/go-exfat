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

	for _, filepath := range files {
		fmt.Printf("%s\n", filepath)
		fmt.Printf("%s\n", nodes[filepath].sede.GeneralSecondaryFlags)
		fmt.Printf("\n")
	}

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
		for i, filepath := range files {
			fmt.Printf("ACTUAL: (%d) [%s]\n", i, filepath)
		}

		for i, filepath := range expectedFiles {
			fmt.Printf("EXPECTED: (%d) [%s]\n", i, filepath)
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

func TestTree_Lookup__File__Hit(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node, err := tree.Lookup([]string{"testdirectory2", "ff7b94be-cec2-11e9-b7b1-6b2e61bd775c"})
	log.PanicIf(err)

	if node == nil {
		t.Fatalf("Did not find the node.")
	}

	if node.Name() != "ff7b94be-cec2-11e9-b7b1-6b2e61bd775c" {
		t.Fatalf("Found node not correct (hit).")
	}
}

func TestTree_Lookup__File__Miss(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node, err := tree.Lookup([]string{"testdirectory2", "invalid_file"})
	log.PanicIf(err)

	if node != nil {
		t.Fatalf("Found node not correct (miss).")
	}
}

func TestTree_Lookup__Folder__Hit(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node, err := tree.Lookup([]string{"testdirectory2"})
	log.PanicIf(err)

	if node == nil {
		t.Fatalf("Did not find the node.")
	}

	if node.Name() != "testdirectory2" {
		t.Fatalf("Found node not correct (hit).")
	}
}

func TestTree_Lookup__Folder__Miss(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node, err := tree.Lookup([]string{"testdirectory2", "invalid_path", "invalid_file"})
	log.PanicIf(err)

	if node != nil {
		t.Fatalf("Expected to not find any nodes.")
	}
}

func TestTree_Lookup__Root__Hit(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node, err := tree.Lookup([]string{})
	log.PanicIf(err)

	if node != tree.rootNode {
		t.Fatalf("Expected root node to be returned.")
	}
}

func TestTree_Lookup__Root__EntryMiss(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node, err := tree.Lookup([]string{"invalid_file"})
	log.PanicIf(err)

	if node != nil {
		t.Fatalf("Expected no node to be found.")
	}
}

func TestTree_IndexedDirectoryEntry(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node, err := tree.Lookup([]string{"2-delahaye-type-165-cabriolet-dsc_8025.jpg"})
	log.PanicIf(err)

	ide := node.IndexedDirectoryEntry()
	if reflect.DeepEqual(ide, node.ide) != true {
		t.Fatalf("IndexedDirectoryEntry did not return IDE.")
	}
}

func TestTree_loadDirectory(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	// Load our directory.

	node, err := tree.Lookup([]string{"testdirectory"})
	log.PanicIf(err)

	err = tree.loadDirectory(node.sede.FirstCluster, node)
	log.PanicIf(err)

	// Do the test.

	rootNode, err := tree.Lookup([]string{})
	log.PanicIf(err)

	_, _, foundNode := rootNode.Lookup([]string{"testdirectory", "300daec8-cec3-11e9-bfa2-0f240e41d1d8"})
	log.PanicIf(err)

	if foundNode.Name() != "300daec8-cec3-11e9-bfa2-0f240e41d1d8" {
		t.Fatalf("Found node not correct.")
	}
}

func TestNewTreeNode(t *testing.T) {
	fde := new(ExfatFileDirectoryEntry)
	sede := new(ExfatStreamExtensionDirectoryEntry)

	tn := NewTreeNode("some name", true, IndexedDirectoryEntry{}, fde, sede)

	if tn.name != "some name" {
		t.Fatalf("name not set correctly.")
	} else if tn.IsDirectory() != true {
		t.Fatalf("IsDirectory not set correctly.")
	}

	if tn.fde != fde {
		t.Fatalf("ExfatFileDirectoryEntry not set correctly.")
	} else if tn.sede != sede {
		t.Fatalf("ExfatStreamExtensionDirectoryEntry not set correctly.")
	}
}

func TestTreeNode_AddChild(t *testing.T) {
	rootNode := NewTreeNode("root", true, IndexedDirectoryEntry{}, nil, nil)
	childNode := rootNode.AddChild("child name", false, nil, nil, IndexedDirectoryEntry{})

	if reflect.DeepEqual(rootNode.ChildFiles(), []string{"child name"}) != true {
		t.Fatalf("New child not registered in parent.")
	}

	recoveredChild := rootNode.GetChild("child name")
	if recoveredChild != childNode {
		t.Fatalf("Recovered child node not correct.")
	}

	if childNode.Name() != "child name" {
		t.Fatalf("New child does not have the right name.")
	}
}

func TestTreeNode_Name(t *testing.T) {
	tn := NewTreeNode("some name", true, IndexedDirectoryEntry{}, nil, nil)

	if tn.Name() != "some name" {
		t.Fatalf("Name not correct.")
	}
}

func TestTreeNode_FileDirectoryEntry(t *testing.T) {
	fde := new(ExfatFileDirectoryEntry)

	tn := NewTreeNode("some name", true, IndexedDirectoryEntry{}, fde, nil)

	if tn.FileDirectoryEntry() != fde {
		t.Fatalf("FileDirectoryEntry not correct.")
	}
}

func TestTreeNode_StreamDirectoryEntry(t *testing.T) {
	sede := new(ExfatStreamExtensionDirectoryEntry)

	tn := NewTreeNode("some name", true, IndexedDirectoryEntry{}, nil, sede)

	if tn.StreamDirectoryEntry() != sede {
		t.Fatalf("StreamDirectoryEntry not correct.")
	}
}

func TestTreeNode_IsDirectory__true(t *testing.T) {
	tn := NewTreeNode("some name", true, IndexedDirectoryEntry{}, nil, nil)

	if tn.IsDirectory() != true {
		t.Fatalf("IsDirectory not correct.")
	}
}

func TestTreeNode_IsDirectory__false(t *testing.T) {
	tn := NewTreeNode("some name", false, IndexedDirectoryEntry{}, nil, nil)

	if tn.IsDirectory() != false {
		t.Fatalf("IsDirectory not correct.")
	}
}

func TestTreeNode_ChildFolders__Root(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	rootNode, err := tree.Lookup([]string{})
	log.PanicIf(err)

	expectedFolders := []string{
		"testdirectory",
		"testdirectory2",
		"testdirectory3",
	}

	if reflect.DeepEqual(rootNode.ChildFolders(), expectedFolders) != true {
		t.Fatalf("Child folders not correct: %v", rootNode.ChildFolders())
	}
}

func TestTreeNode_ChildFolders__Subfolder(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node, err := tree.Lookup([]string{"testdirectory"})
	log.PanicIf(err)

	expectedFolders := []string{}

	if reflect.DeepEqual(node.ChildFolders(), expectedFolders) != true {
		t.Fatalf("Child folders not correct: %v", node.ChildFolders())
	}
}

func TestTreeNode_ChildFiles__Root(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	rootNode, err := tree.Lookup([]string{})
	log.PanicIf(err)

	expectedFiles := []string{
		"064cbfd4-cec3-11e9-926d-c362c80fab7b",
		"2-delahaye-type-165-cabriolet-dsc_8025.jpg",
		"79c6d31a-cca1-11e9-8325-9746d045e868",
		"8fd71ab132c59bf33cd7890c0acebf12.jpg",
	}

	if reflect.DeepEqual(rootNode.ChildFiles(), expectedFiles) != true {
		t.Fatalf("Child files not correct: %v", rootNode.ChildFiles())
	}
}

func TestTreeNode_ChildFiles__Subfolder(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node, err := tree.Lookup([]string{"testdirectory"})
	log.PanicIf(err)

	expectedFiles := []string{
		"300daec8-cec3-11e9-bfa2-0f240e41d1d8",
	}

	if reflect.DeepEqual(node.ChildFiles(), expectedFiles) != true {
		t.Fatalf("Child files not correct: %v", node.ChildFiles())
	}
}

func TestTreeNode_GetChild(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node, err := tree.Lookup([]string{"testdirectory"})
	log.PanicIf(err)

	childNode := node.GetChild("300daec8-cec3-11e9-bfa2-0f240e41d1d8")

	if childNode != node.childrenMap["300daec8-cec3-11e9-bfa2-0f240e41d1d8"] {
		t.Fatalf("Child not correct.")
	}
}

func TestTreeNode_Lookup__Folder__Hit(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node, err := tree.Lookup([]string{"testdirectory"})
	log.PanicIf(err)

	_, _, foundNode := node.Lookup([]string{"300daec8-cec3-11e9-bfa2-0f240e41d1d8"})
	log.PanicIf(err)

	if foundNode.Name() != "300daec8-cec3-11e9-bfa2-0f240e41d1d8" {
		t.Fatalf("Found node not correct.")
	}
}

func TestTreeNode_Lookup__Folder__Miss(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node, err := tree.Lookup([]string{"testdirectory"})
	log.PanicIf(err)

	lastPathParts, lastNode, foundNode := node.Lookup([]string{"invalid_path", "invalid_file"})
	log.PanicIf(err)

	if foundNode != nil {
		t.Fatalf("Expected no node to be returned for miss.")
	} else if reflect.DeepEqual(lastPathParts, []string{"invalid_path", "invalid_file"}) != true {
		t.Fatalf("Expected missing file to still be in the path-parts.")
	} else if lastNode != node {
		t.Fatalf("Last-node not correct.")
	}
}

func TestTreeNode_Lookup__File__Hit(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	// Load our directory.

	node, err := tree.Lookup([]string{"testdirectory"})
	log.PanicIf(err)

	err = tree.loadDirectory(node.sede.FirstCluster, node)
	log.PanicIf(err)

	// Do the test.

	rootNode, err := tree.Lookup([]string{})
	log.PanicIf(err)

	_, _, foundNode := rootNode.Lookup([]string{"testdirectory", "300daec8-cec3-11e9-bfa2-0f240e41d1d8"})
	log.PanicIf(err)

	if foundNode.Name() != "300daec8-cec3-11e9-bfa2-0f240e41d1d8" {
		t.Fatalf("Found node not correct.")
	}
}

func TestTreeNode_Lookup__File__Miss(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	node, err := tree.Lookup([]string{"testdirectory"})
	log.PanicIf(err)

	lastPathParts, lastNode, foundNode := node.Lookup([]string{"invalid_file"})
	log.PanicIf(err)

	if foundNode != nil {
		t.Fatalf("Expected no node to be returned for miss.")
	} else if reflect.DeepEqual(lastPathParts, []string{"invalid_file"}) != true {
		t.Fatalf("Expected missing file to still be in the path-parts.")
	} else if lastNode != node {
		t.Fatalf("Last-node not correct.")
	}
}

func TestTree_Load(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	rootNode, err := tree.Lookup([]string{})
	log.PanicIf(err)

	expectedFolders := []string{
		"testdirectory",
		"testdirectory2",
		"testdirectory3",
	}

	if reflect.DeepEqual(rootNode.ChildFolders(), expectedFolders) != true {
		t.Fatalf("Child folders not correct: %v", rootNode.ChildFolders())
	}

	expectedFiles := []string{
		"064cbfd4-cec3-11e9-926d-c362c80fab7b",
		"2-delahaye-type-165-cabriolet-dsc_8025.jpg",
		"79c6d31a-cca1-11e9-8325-9746d045e868",
		"8fd71ab132c59bf33cd7890c0acebf12.jpg",
	}

	if reflect.DeepEqual(rootNode.ChildFiles(), expectedFiles) != true {
		t.Fatalf("Child files not correct: %v", rootNode.ChildFiles())
	}
}

func TestTree_Visit(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	collected := make([][]string, 0)

	cb := func(pathParts []string, node *TreeNode) (err error) {
		collected = append(collected, pathParts)
		return nil
	}

	err = tree.Visit(cb)
	log.PanicIf(err)

	expectedCollected := [][]string{
		[]string{},
		[]string{"testdirectory"},
		[]string{"testdirectory", "300daec8-cec3-11e9-bfa2-0f240e41d1d8"},
		[]string{"testdirectory2"},
		[]string{"testdirectory2", "00c57ab0-cec3-11e9-b750-bbed8d2244c8"},
		[]string{"testdirectory2", "ff7b94be-cec2-11e9-b7b1-6b2e61bd775c"},
		[]string{"testdirectory2", "file1"},
		[]string{"testdirectory2", "file2"},
		[]string{"testdirectory3"},
		[]string{"testdirectory3", "10422c86-cec3-11e9-953f-4f501efd2640"},
		[]string{"064cbfd4-cec3-11e9-926d-c362c80fab7b"},
		[]string{"2-delahaye-type-165-cabriolet-dsc_8025.jpg"},
		[]string{"79c6d31a-cca1-11e9-8325-9746d045e868"},
		[]string{"8fd71ab132c59bf33cd7890c0acebf12.jpg"},
	}

	if reflect.DeepEqual(collected, expectedCollected) != true {
		for i, pathParts := range collected {
			fmt.Printf("ACTUAL (%d): %v\n", i, pathParts)
		}

		for i, pathParts := range expectedCollected {
			fmt.Printf("EXPECTED (%d): %v\n", i, pathParts)
		}

		t.Fatalf("Collected paths not correct.")
	}
}

func TestTree_visit(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	tree := NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	collected := make([][]string, 0)

	cb := func(pathParts []string, node *TreeNode) (err error) {
		collected = append(collected, pathParts)
		return nil
	}

	pathParts := make([]string, 0)

	err = tree.visit(pathParts, tree.rootNode, cb)
	log.PanicIf(err)

	expectedCollected := [][]string{
		[]string{},
		[]string{"testdirectory"},
		[]string{"testdirectory", "300daec8-cec3-11e9-bfa2-0f240e41d1d8"},
		[]string{"testdirectory2"},
		[]string{"testdirectory2", "00c57ab0-cec3-11e9-b750-bbed8d2244c8"},
		[]string{"testdirectory2", "ff7b94be-cec2-11e9-b7b1-6b2e61bd775c"},
		[]string{"testdirectory2", "file1"},
		[]string{"testdirectory2", "file2"},
		[]string{"testdirectory3"},
		[]string{"testdirectory3", "10422c86-cec3-11e9-953f-4f501efd2640"},
		[]string{"064cbfd4-cec3-11e9-926d-c362c80fab7b"},
		[]string{"2-delahaye-type-165-cabriolet-dsc_8025.jpg"},
		[]string{"79c6d31a-cca1-11e9-8325-9746d045e868"},
		[]string{"8fd71ab132c59bf33cd7890c0acebf12.jpg"},
	}

	if reflect.DeepEqual(collected, expectedCollected) != true {
		for i, pathParts := range collected {
			fmt.Printf("ACTUAL (%d): %v\n", i, pathParts)
		}

		for i, pathParts := range expectedCollected {
			fmt.Printf("EXPECTED (%d): %v\n", i, pathParts)
		}

		t.Fatalf("Collected paths not correct.")
	}
}
