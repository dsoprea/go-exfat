package main

import (
	"os"

	"github.com/dsoprea/go-logging"
	"github.com/jessevdk/go-flags"

	"github.com/dsoprea/go-exfat"
)

type rootParameters struct {
	Filepath string `short:"f" long:"filepath" description:"File-path of exFAT filesystem" required:"true"`
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

	er.ActiveBootRegion().Dump()
}
