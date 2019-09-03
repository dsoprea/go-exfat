package exfat

import (
	"fmt"
	"reflect"

	"github.com/dsoprea/go-logging"
	"github.com/go-restruct/restruct"
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

type DirectoryEntryVisitorFunc func() (err error)

type EntryType uint8

func (et EntryType) IsEndOfDirectory() bool {
	return et == 0
}

func (et EntryType) IsUnusedEntryMarker() bool {
	return et >= 0x01 && et <= 0x7f
}

func (et EntryType) IsRegular() bool {
	return et >= 0x81 && et <= 0xff
}

func (et EntryType) TypeCode() int {
	return int(et & 31)
}

func (et EntryType) TypeImportance() bool {
	return et&32 > 0
}

func (et EntryType) IsCritical() bool {
	return et.TypeImportance() == false
}

func (et EntryType) IsBenign() bool {
	return et.TypeImportance() == true
}

func (et EntryType) TypeCategory() bool {
	return et&64 > 0
}

func (et EntryType) IsPrimary() bool {
	return et.TypeCategory() == false
}

func (et EntryType) IsSecondary() bool {
	return et.TypeCategory() == true
}

func (et EntryType) IsInUse() bool {
	return et&128 > 0
}

func (et EntryType) Dump() {
	fmt.Printf("Entry Type\n")
	fmt.Printf("==========\n")
	fmt.Printf("\n")

	fmt.Printf("TypeCode: (%d)\n", et.TypeCode())
	fmt.Printf("\n")

	fmt.Printf("TypeImportance: [%v]\n", et.TypeImportance())
	fmt.Printf("- IsCritical: [%v]\n", et.IsCritical())
	fmt.Printf("- IsBenign: [%v]\n", et.IsBenign())
	fmt.Printf("\n")

	fmt.Printf("TypeCategory: [%v]\n", et.TypeCategory())
	fmt.Printf("- IsPrimary: [%v]\n", et.IsPrimary())
	fmt.Printf("- IsSecondary: [%v]\n", et.IsSecondary())
	fmt.Printf("\n")

	fmt.Printf("IsInUse: [%v]\n", et.IsInUse())
	fmt.Printf("\n")

	fmt.Printf("Entry-Type Classes\n")
	fmt.Printf("- IsEndOfDirectory: [%v]\n", et.IsEndOfDirectory())
	fmt.Printf("- IsUnusedEntryMarker: [%v]\n", et.IsUnusedEntryMarker())
	fmt.Printf("- IsRegular: [%v]\n", et.IsRegular())
	fmt.Printf("\n")
}

func (et EntryType) String() string {
	return fmt.Sprintf("EntryType<TYPE-CODE=(%d) IS-CRITICAL=[%v] IS-PRIMARY=[%v] IS-IN-USE=[%v] X-IS-REGULAR=[%v] X-IS-UNUSED=[%v] X-IS-END=[%v]>", et.TypeCode(), et.IsCritical(), et.IsPrimary(), et.IsInUse(), et.IsRegular(), et.IsUnusedEntryMarker(), et.IsEndOfDirectory())
}

type DirectoryEntryVisitorFunc func(primaryEntry ExfatPrimaryDirectoryEntry, secondaryEntries []ExfatSecondaryDirectoryEntry) (err error)

func (en *ExfatNavigator) EnumerateDirectoryEntries() (err error) {
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

	// Enumerate clusters.

	entryNumber := 0
	// entryCount := -1
	var collectedSecondaryEntries []ExfatSecondaryDirectoryEntry
	needSecondaryEntryCount := 0
	isDone := false

	// TODO(dustin): Add additional strictness? Should every secondary entry be collected for the nearest preceding primary entry? This means that we can/need to validate that the size of the sequence of secondary entries much match the second-entry count stored on the last primary entry.

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
				// fmt.Printf("    Directory entry (%d)\n", i)

				directoryEntryData := data[i*directoryEntryBytesCount : (i+1)*directoryEntryBytesCount]

				entryType := EntryType(directoryEntryData[0])

				fmt.Printf("(%d): (%d) (%08b) %s\n", entryNumber, entryType, entryType, entryType)

				// We've hit the terminal record.
				if entryType == 0 {
					isDone = true
					return false, nil
				}

				if entryType.TypeCode() == 5 && entryType.IsCritical() == true && entryType.IsPrimary() == true {
					fileDe := ExfatFileDirectoryEntry{}

					err := restruct.Unpack(directoryEntryData, defaultEncoding, &fileDe)
					log.PanicIf(err)

					fmt.Printf("FILE: %s\n", fileDe)
				}

				// TODO(dustin): !! Finish defining structs for all entry-types (which is also made mandatory by the spec), and then collect all primary entries (coupled with the exact number of subsequent secondary entries) and forward to a callback.

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

	return primaryEntry, secondaryEntries, nil
}
