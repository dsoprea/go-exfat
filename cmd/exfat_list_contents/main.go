package main

import (
	"fmt"
	"os"

	"path/filepath"

	"github.com/dsoprea/go-logging"
	"github.com/dustin/go-humanize"
	"github.com/jessevdk/go-flags"

	"github.com/dsoprea/go-exfat"
)

type rootParameters struct {
	Filepath       string `short:"f" long:"filepath" description:"File-path of exFAT filesystem" required:"true"`
	FilenameFilter string `short:"p" long:"pattern" description:"Filename filter"`
	ShowDetail     bool   `short:"d" long:"detail" description:"Show additional entry detail"`
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

	f, err := os.Open(rootArguments.Filepath)
	log.PanicIf(err)

	defer f.Close()

	er := exfat.NewExfatReader(f)

	err = er.Parse()
	log.PanicIf(err)

	tree := exfat.NewTree(er)

	err = tree.Load()
	log.PanicIf(err)

	files, nodes, err := tree.List()
	log.PanicIf(err)

	for _, currentFilepath := range files {
		node := nodes[currentFilepath]

		if rootArguments.FilenameFilter != "" {
			// Since the filepaths are separated by Windows-standard backward-
			// slashes, they won't necessarily split correcty on all platforms.
			// Therefore, we'll just use the name from the node.
			filename := node.Name()

			isMatched, err := filepath.Match(rootArguments.FilenameFilter, filename)
			log.PanicIf(err)

			if isMatched != true {
				continue
			}
		}

		fde := node.FileDirectoryEntry()
		sde := node.StreamDirectoryEntry()

		if rootArguments.ShowDetail == true {
			fmt.Printf("## %s\n", currentFilepath)
			fmt.Printf("\n")

			ide := node.IndexedDirectoryEntry()

			fmt.Printf("[Primary Entry]\n")
			fmt.Printf("\n")

			fde.Dump()

			for _, de := range ide.SecondaryEntries {
				if dde, ok := de.(exfat.DumpableDirectoryEntry); ok == true {
					fmt.Printf("[Secondary Entry]\n")
					fmt.Printf("\n")

					dde.Dump()
				} else {
					fmt.Printf("[Secondary Entry] %s\n", de)
				}
			}

			fmt.Printf("\n")
		} else {
			fmt.Printf("%15s %30s %s\n", humanize.Comma(int64(sde.ValidDataLength)), fde.LastModifiedTimestamp(), currentFilepath)
		}
	}
}
