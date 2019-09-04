package exfat

import (
	"fmt"
	"reflect"

	"github.com/dsoprea/go-logging"
)

const (
	// This field is mandatory and Section 6.1 defines its contents.
	directoryEntryBytesCount = 32
)

type ExfatNavigator struct {
	er                 *ExfatReader
	firstClusterNumber uint32
}

func NewExfatNavigator(er *ExfatReader, firstClusterNumber uint32) (en *ExfatNavigator) {
	return &ExfatNavigator{
		er:                 er,
		firstClusterNumber: firstClusterNumber,
	}
}

type DirectoryEntryVisitorFunc func(primaryEntry DirectoryEntry, secondaryEntries []DirectoryEntry) (err error)

func (en *ExfatNavigator) EnumerateDirectoryEntries(cb DirectoryEntryVisitorFunc) (err error) {
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

	// TODO(dustin): Add test.

	// Enumerate clusters.

	entryNumber := 0
	isDone := false

	var primaryEntry DirectoryEntry
	var secondaryEntries []DirectoryEntry

	cvf := func(ec *ExfatCluster) (doContinue bool, err error) {
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

		// Enumerate sectors.

		svf := func(sectorNumber uint32, data []byte) (doContinue bool, err error) {
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

			sectorSize := en.er.SectorSize()

			i := 0
			for {
				directoryEntryData := data[i*directoryEntryBytesCount : (i+1)*directoryEntryBytesCount]

				entryType := EntryType(directoryEntryData[0])

				// We've hit the terminal record.
				if entryType.IsEndOfDirectory() == true {
					isDone = true
					return false, nil
				}

				de, err := parseDirectoryEntry(entryType, directoryEntryData)
				log.PanicIf(err)

				if entryType.IsPrimary() == true {
					primaryEntry = de

					// We'll always overwrite the primary as part of our
					// process. Note that any secordary entries that we
					// encounter will be appended to `secondaryEntries` but
					// unless the last primary entry indicate that it wanted any
					// of those secondary entries, they'll be forgotten.
					secondaryEntries = make([]DirectoryEntry, 0)
				} else {
					secondaryEntries = append(secondaryEntries, de)
				}

				// If the primary entry did not have a secondary entry
				// requirement, or it did and we've met it, call the callback.
				if pde, ok := primaryEntry.(PrimaryDirectoryEntry); ok == true {
					if len(secondaryEntries) == int(pde.SecondaryCount()) {
						err := cb(primaryEntry, secondaryEntries)
						log.PanicIf(err)
					}
				} else if entryType.IsPrimary() == true {
					// We're conceding the presence of primary entry-types that
					// don't necessarily have a SecondaryCount field (which is
					// the qualification to be considered a
					// `PrimaryDirectoryEntry`). Therefore, if our primary was
					// not a `PrimaryDirectoryEntry` *but* it's still
					// purportedly a primary entry, call the callback with an
					// empty list for the secondary entries (the
					// `secondaryEntries` entry list will always be empty here
					// due to above).

					err := cb(primaryEntry, secondaryEntries)
					log.PanicIf(err)
				}

				entryNumber++

				i++

				if uint32(i*directoryEntryBytesCount) >= sectorSize {
					break
				}
			}

			return true, nil
		}

		err = ec.EnumerateSectors(svf)
		log.PanicIf(err)

		if isDone == true {
			return false, nil
		}

		return true, nil
	}

	err = en.er.EnumerateClusters(en.firstClusterNumber, cvf)
	log.PanicIf(err)

	return nil
}

type IndexedDirectoryEntry struct {
	PrimaryEntry     DirectoryEntry
	SecondaryEntries []DirectoryEntry
	Extra            map[string]interface{}
}

type DirectoryEntryIndex map[string][]IndexedDirectoryEntry

func (dei DirectoryEntryIndex) Dump() {
	fmt.Printf("Directory Entry Index\n")
	fmt.Printf("=====================\n")
	fmt.Printf("\n")

	for typeName, ideList := range dei {
		for _, ide := range ideList {
			fmt.Printf("%s\n", typeName)
			fmt.Printf("--------------------\n")
			fmt.Printf("Primary: %s\n", ide.PrimaryEntry)

			for i, secondaryEntry := range ide.SecondaryEntries {
				fmt.Printf("Secondary (%d): %s\n", i, secondaryEntry)
			}

			fmt.Printf("\n")

			if len(ide.Extra) > 0 {
				fmt.Printf("Extra:\n")

				for k, v := range ide.Extra {
					fmt.Printf("> %s: %s\n", k, v)
				}

				fmt.Printf("\n")
			}
		}
	}
}

func (en *ExfatNavigator) IndexDirectoryEntries() (index DirectoryEntryIndex, err error) {
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

	index = make(DirectoryEntryIndex)

	cb := func(primaryEntry DirectoryEntry, secondaryEntries []DirectoryEntry) (err error) {
		extra := make(map[string]interface{})

		ide := IndexedDirectoryEntry{
			PrimaryEntry:     primaryEntry,
			SecondaryEntries: secondaryEntries,
			Extra:            extra,
		}

		if _, ok := primaryEntry.(*ExfatFileDirectoryEntry); ok == true {
			mf := MultipartFilename(secondaryEntries)
			complete_filename := mf.Filename()

			extra["complete_filename"] = complete_filename
		}

		typeName := primaryEntry.TypeName()
		if list_, found := index[typeName]; found == true {
			index[typeName] = append(list_, ide)
		} else {
			index[typeName] = []IndexedDirectoryEntry{ide}
		}

		return nil
	}

	err = en.EnumerateDirectoryEntries(cb)
	log.PanicIf(err)

	return index, nil
}
