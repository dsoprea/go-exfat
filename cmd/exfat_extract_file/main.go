package main

import (
	"fmt"
	"os"

	"github.com/dsoprea/go-logging"
	"github.com/jessevdk/go-flags"

	"github.com/dsoprea/go-exfat"
)

type rootParameters struct {
	FilesystemFilepath string `short:"f" long:"filesystem-filepath" description:"File-path of exFAT filesystem" required:"true"`
	ExtractFilepath    string `short:"e" long:"extract-filepath" description:"File-path to extract (use forward slashes)" required:"true"`
	OutputFilepath     string `short:"o" long:"output-filepath" description:"File-path to write to ('-' for STDOUT)" required:"true"`
}

var (
	rootArguments = new(rootParameters)
)

func main() {
	defer func() {
		if state := recover(); state != nil {
			err := log.Wrap(state.(error))
			log.PrintError(err)
			os.Exit(-1)
		}
	}()

	p := flags.NewParser(rootArguments, flags.Default)

	_, err := p.Parse()
	if err != nil {
		os.Exit(1)
	}

	f, err := os.Open(rootArguments.FilesystemFilepath)
	log.PanicIf(err)

	defer f.Close()

	er := exfat.NewExfatReader(f)

	err = er.Parse()
	log.PanicIf(err)

	tree := exfat.NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	// We use the List() call because it provides a simple lookup for the
	// complete path strings, which a) simplifies the process for us, and
	// b) eliminates any unnecessary interpretation/construction of the path-
	// names on our end so that we can avoid issues with problematic preexisting
	// slashes in the filepath that prevents us from finding the file-path that
	// the user provides.
	_, nodes, err := tree.List()
	log.PanicIf(err)

	node, found := nodes[rootArguments.ExtractFilepath]
	if found != true {
		fmt.Printf("File not found.\n")
		os.Exit(2)
	}

	var g *os.File

	if rootArguments.OutputFilepath == "-" {
		g = os.Stdout
	} else {
		var err error

		g, err = os.Create(rootArguments.OutputFilepath)
		log.PanicIf(err)

		defer func() {
			g.Close()
		}()
	}

	sde := node.StreamDirectoryEntry()

	useFat := sde.GeneralSecondaryFlags.NoFatChain() == false

	err = er.WriteFromClusterChain(sde.FirstCluster, sde.ValidDataLength, useFat, g)
	log.PanicIf(err)

	if rootArguments.OutputFilepath != "-" {
		fmt.Printf("(%d) bytes written.\n", sde.ValidDataLength)
	}
}
