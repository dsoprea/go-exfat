package exfat

import (
	"reflect"
	"sort"
	"strings"

	"github.com/dsoprea/go-logging"
)

// TODO(dustin): !! Refactor this to load lazily.

type TreeNode struct {
	name string

	isDirectory bool
	sede        *ExfatStreamExtensionDirectoryEntry

	childrenFolders sort.StringSlice
	childrenFiles   sort.StringSlice

	childrenMap map[string]*TreeNode
}

func NewTreeNode(name string, isDirectory bool, sede *ExfatStreamExtensionDirectoryEntry) (tn *TreeNode) {

	// TODO(dustin): !! Add tests.

	childrenList := make(sort.StringSlice, 0)
	childrenMap := make(map[string]*TreeNode)

	tn = &TreeNode{
		name:        name,
		isDirectory: isDirectory,
		sede:        sede,

		childrenFolders: childrenList,
		childrenFiles:   childrenList,

		childrenMap: childrenMap,
	}

	return tn
}

func (tn *TreeNode) Name() string {

	// TODO(dustin): !! Add tests.

	return tn.name
}

func (tn *TreeNode) StreamDirectoryEntry() *ExfatStreamExtensionDirectoryEntry {

	// TODO(dustin): !! Add tests.

	return tn.sede
}

func (tn *TreeNode) IsDirectory() bool {

	// TODO(dustin): !! Add tests.

	return tn.isDirectory
}

func (tn *TreeNode) ChildFolders() []string {

	// TODO(dustin): !! Add tests.

	return tn.childrenFolders
}

func (tn *TreeNode) ChildFiles() []string {

	// TODO(dustin): !! Add tests.

	return tn.childrenFiles
}

func (tn *TreeNode) GetChild(filename string) *TreeNode {

	// TODO(dustin): !! Add tests.

	return tn.childrenMap[filename]
}

func (tn *TreeNode) Lookup(pathParts []string) *TreeNode {

	// TODO(dustin): !! Add tests.

	if len(pathParts) == 0 {
		// We've reached and found the last part.
		return tn
	}

	childNode := tn.childrenMap[pathParts[0]]
	if childNode == nil {
		// An intermediate part was not found.
		return nil
	}

	foundNode := childNode.Lookup(pathParts[1:])
	return foundNode
}

func (tn *TreeNode) AddChild(name string, isDirectory bool, sede *ExfatStreamExtensionDirectoryEntry) *TreeNode {

	// TODO(dustin): !! Add tests.

	childNode := NewTreeNode(name, isDirectory, sede)

	// The adds are driven through a process based on a map, so the order will
	// always be random. Use insertion sort to order the children so their order
	// is deterministic.

	var list sort.StringSlice
	if isDirectory == true {
		list = tn.childrenFolders
	} else {
		list = tn.childrenFiles
	}

	insertOrEqualAt := list.Search(name)

	if insertOrEqualAt >= len(list) {
		list = append(list, name)
	} else if list[insertOrEqualAt] != name {
		leftHalf := list[:insertOrEqualAt]
		rightHalf := list[insertOrEqualAt:]
		list = append(leftHalf, append([]string{name}, rightHalf...)...)
	}

	if isDirectory == true {
		tn.childrenFolders = list
	} else {
		tn.childrenFiles = list
	}

	tn.childrenMap[name] = childNode

	return childNode
}

type Tree struct {
	er       *ExfatReader
	rootNode *TreeNode
}

func NewTree(er *ExfatReader) *Tree {
	rootNode := NewTreeNode("", true, nil)

	return &Tree{
		er:       er,
		rootNode: rootNode,
	}
}

func (tree *Tree) loadDirectory(clusterNumber uint32, node *TreeNode) (err error) {
	defer func() {
		if errRaw := recover(); errRaw != nil {
			var ok bool
			if err, ok = errRaw.(error); ok == true {
				err = log.Wrap(err)
			} else {
				err = log.Errorf("Error not an error: [%s] [%v]", reflect.TypeOf(err).Name(), err)
			}
		}
	}()

	// TODO(dustin): !! Add tests.

	en := NewExfatNavigator(tree.er, clusterNumber)

	index, err := en.IndexDirectoryEntries()
	log.PanicIf(err)

	filenames := index.Filenames()

	for filename, isDirectory := range filenames {
		sede := index.FindIndexedFileStreamExtensionDirectoryEntry(filename)
		childNode := node.AddChild(filename, isDirectory, sede)

		if isDirectory == true {
			err := tree.loadDirectory(sede.FirstCluster, childNode)
			log.PanicIf(err)
		}
	}

	return nil
}

func (tree *Tree) Load() (err error) {
	defer func() {
		if errRaw := recover(); errRaw != nil {
			var ok bool
			if err, ok = errRaw.(error); ok == true {
				err = log.Wrap(err)
			} else {
				err = log.Errorf("Error not an error: [%s] [%v]", reflect.TypeOf(err).Name(), err)
			}
		}
	}()

	// TODO(dustin): !! Add tests.

	clusterNumber := tree.er.FirstClusterOfRootDirectory()

	err = tree.loadDirectory(clusterNumber, tree.rootNode)
	log.PanicIf(err)

	return nil
}

func (tree *Tree) Lookup(pathParts []string) (node *TreeNode) {
	return tree.rootNode.Lookup(pathParts)
}

type TreeVisitorFunc func(pathParts []string, node *TreeNode) (err error)

func (tree *Tree) Visit(cb TreeVisitorFunc) (err error) {
	defer func() {
		if errRaw := recover(); errRaw != nil {
			var ok bool
			if err, ok = errRaw.(error); ok == true {
				err = log.Wrap(err)
			} else {
				err = log.Errorf("Error not an error: [%s] [%v]", reflect.TypeOf(err).Name(), err)
			}
		}
	}()

	// TODO(dustin): !! Add tests.

	pathParts := make([]string, 0)

	err = tree.visit(pathParts, tree.rootNode, cb)
	log.PanicIf(err)

	return nil
}

func (tree *Tree) visit(pathParts []string, node *TreeNode, cb TreeVisitorFunc) (err error) {
	defer func() {
		if errRaw := recover(); errRaw != nil {
			var ok bool
			if err, ok = errRaw.(error); ok == true {
				err = log.Wrap(err)
			} else {
				err = log.Errorf("Error not an error: [%s] [%v]", reflect.TypeOf(err).Name(), err)
			}
		}
	}()

	// TODO(dustin): !! Add tests.

	err = cb(pathParts, node)
	log.PanicIf(err)

	files := make([]*TreeNode, 0)

	for _, childFolderName := range node.childrenFolders {
		childNode := node.childrenMap[childFolderName]

		childPathParts := make([]string, len(pathParts)+1)
		copy(childPathParts, pathParts)
		childPathParts[len(childPathParts)-1] = childNode.name

		if childNode.isDirectory == true {
			err := tree.visit(childPathParts, childNode, cb)
			log.PanicIf(err)
		} else {
			files = append(files, childNode)
		}
	}

	// Do the files all at once, at the bottom.
	for _, childFilename := range node.childrenFiles {
		childNode := node.childrenMap[childFilename]

		childPathParts := make([]string, len(pathParts)+1)
		copy(childPathParts, pathParts)
		childPathParts[len(childPathParts)-1] = childFilename

		err := cb(childPathParts, childNode)
		log.PanicIf(err)
	}

	return nil
}

func (tree *Tree) List() (files []string, nodes map[string]*TreeNode, err error) {
	defer func() {
		if errRaw := recover(); errRaw != nil {
			var ok bool
			if err, ok = errRaw.(error); ok == true {
				err = log.Wrap(err)
			} else {
				err = log.Errorf("Error not an error: [%s] [%v]", reflect.TypeOf(err).Name(), err)
			}
		}
	}()

	files = make([]string, 0)
	nodes = make(map[string]*TreeNode)

	cb := func(pathParts []string, node *TreeNode) (err error) {
		if len(pathParts) == 0 {
			return nil
		}

		nodePath := strings.Join(pathParts, `\`)

		files = append(files, nodePath)
		nodes[nodePath] = node

		return nil
	}

	err = tree.Visit(cb)
	log.PanicIf(err)

	return files, nodes, nil
}
