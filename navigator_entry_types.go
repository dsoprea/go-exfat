package exfat

import (
	"fmt"
	"time"
)

type ExfatPrimaryDirectoryEntry struct {
	// This field is mandatory and Section 6.3.1 defines its contents.
	//
	// The EntryType field shall conform to the definition provided in the Generic DirectoryEntry template (see Section 6.2.1).
	EntryType EntryType

	// This field is mandatory and Section 6.3.2 defines its contents.
	//
	// The SecondaryCount field shall describe the number of secondary directory entries which immediately follow the given primary directory entry. These secondary directory entries, along with the given primary directory entry, comprise the directory entry set.
	//
	// The valid range of values for this field shall be:
	//
	// At least 0, which means this primary directory entry is the only entry in the directory entry set
	//
	// At most 255, which means the next 255 directory entries and this primary directory entry comprise the directory entry set
	//
	// Critical primary directory entry structures which derive from this template may redefine both the SecondaryCount and SetChecksum fields.
	SecondaryCount uint8

	// This field is mandatory and Section 6.3.3 defines its contents.
	//
	// The SetChecksum field shall contain the checksum of all directory entries in the given directory entry set. However, the checksum excludes this field (see Figure 2). Implementations shall verify the contents of this field are valid prior to using any other directory entry in the given directory entry set.
	//
	// Critical primary directory entry structures which derive from this template may redefine both the SecondaryCount and SetChecksum fields.
	SetChecksum uint16

	// This field is mandatory and Section 6.3.4 defines its contents.
	//
	// The GeneralPrimaryFlags field contains flags (see Table 17).
	//
	// Critical primary directory entry structures which derive from this template may redefine this field.
	GeneralPrimaryFlags uint16

	// This field is mandatory and structures which derive from this template define its contents.
	CustomDefined [14]byte

	// This field is mandatory and Section 6.3.5 defines its contents.
	//
	// The FirstCluster field shall conform to the definition provided in the Generic DirectoryEntry template (see Section 6.2.2).
	//
	// If the NoFatChain bit is 1 then FirstCluster must point to a valid cluster in the cluster heap.
	//
	// Critical primary directory entry structures which derive from this template may redefine the FirstCluster and DataLength fields. Other structures which derive from this template may redefine the FirstCluster and DataLength fields only if the AllocationPossible field contains the value 0.
	FirstCluster uint32

	// This field is mandatory and Section 6.3.6 defines its contents.
	//
	// The DataLength field shall conform to the definition provided in the Generic DirectoryEntry template (see Section 6.2.3).
	//
	// If the NoFatChain bit is 1 then DataLength must not be zero. If the FirstCluster field is zero, then DataLength must also be zero.
	//
	// Critical primary directory entry structures which derive from this template may redefine the FirstCluster and DataLength fields. Other structures which derive from this template may redefine the FirstCluster and DataLength fields only if the AllocationPossible field contains the value 0.
	DataLength uint64
}

func (sde ExfatPrimaryDirectoryEntry) String() string {
	return fmt.Sprintf("PrimaryDirectoryEntry<TYPE=(%d) SECONDARY-COUNT=(%d) FIRST-CLUSTER=(%d) DATA-LENGTH=(%d)>", sde.EntryType, sde.SecondaryCount, sde.FirstCluster, sde.DataLength)
}

func (sde ExfatPrimaryDirectoryEntry) Dump() {
	fmt.Printf("Primary Directory Entry\n")
	fmt.Printf("=======================\n")
	fmt.Printf("\n")

	fmt.Printf("EntryType: (%d) [%08b]\n", sde.EntryType, sde.EntryType)
	fmt.Printf("SecondaryCount: (%d)\n", sde.SecondaryCount)
	fmt.Printf("SetChecksum: (%04x)\n", sde.SetChecksum)
	fmt.Printf("GeneralPrimaryFlags: (%04x)\n", sde.GeneralPrimaryFlags)
	fmt.Printf("FirstCluster: (%d)\n", sde.FirstCluster)
	fmt.Printf("DataLength: (%d)\n", sde.DataLength)

	fmt.Printf("\n")
}

// func (en *ExfatNavigator) parsePrimaryEntry(raw []byte) (primaryDe ExfatPrimaryDirectoryEntry, err error) {
// 	defer func() {
// 		if errRaw := recover(); errRaw != nil {
// 			var ok bool
// 			if err, ok = errRaw.(error); ok == true {
// 				err = log.Wrap(err)
// 			} else {
// 				err = log.Errorf("Error not an error: [%s] [%v]", reflect.TypeOf(err).Name(), err)
// 			}
// 		}
// 	}()

// 	err = restruct.Unpack(raw, defaultEncoding, &primaryDe)
// 	log.PanicIf(err)

// 	return primaryDe, nil
// }

type ExfatTimestamp uint32

func (et ExfatTimestamp) Second() int {
	return int(et & 31)
}

func (et ExfatTimestamp) Minute() int {
	return int(et&2016) >> 5
}

func (et ExfatTimestamp) Hour() int {
	return int(et&63488) >> 11
}

func (et ExfatTimestamp) Day() int {
	return int(et&2031616) >> 16
}

func (et ExfatTimestamp) Month() int {
	return int(et&31457280) >> 21
}

func (et ExfatTimestamp) Year() int {
	return 1980 + int(et&4261412864)>>25
}

func (et ExfatTimestamp) Timestamp() time.Time {

	// TODO(dustin): Implement the timezone.

	return time.Date(et.Year(), time.Month(et.Month()), et.Day(), et.Hour(), et.Minute(), et.Second(), 0, time.Local)
}

func (et ExfatTimestamp) String() string {
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", et.Year(), et.Month(), et.Day(), et.Hour(), et.Minute(), et.Second())
}

type ExfatFileDirectoryEntry struct {
	// This field is mandatory and Section 7.4.1 defines its contents.
	EntryType EntryType

	// This field is mandatory and Section 7.4.2 defines its contents.
	SecondaryCount uint8

	// This field is mandatory and Section 7.4.3 defines its contents.
	SetChecksum uint16

	// This field is mandatory and Section 7.4.4 defines its contents.
	FileAttributes uint16

	// This field is mandatory and its contents are reserved.
	Reserved1 uint16

	// This field is mandatory and Section 7.4.5 defines its contents.
	CreateTimestamp ExfatTimestamp

	// This field is mandatory and Section 7.4.6 defines its contents.
	LastModifiedTimestamp ExfatTimestamp

	// This field is mandatory and Section 7.4.7 defines its contents.
	LastAccessedTimestamp ExfatTimestamp

	// This field is mandatory and Section 7.4.5 defines its contents.
	Create10msIncrement uint8

	// This field is mandatory and Section 7.4.6 defines its contents.
	LastModified10msIncrement uint8

	// This field is mandatory and Section 7.4.5 defines its contents.
	CreateUtcOffset uint8

	// This field is mandatory and Section 7.4.6 defines its contents.
	LastModifiedUtcOffset uint8

	// This field is mandatory and Section 7.4.7 defines its contents.
	LastAccessedUtcOffset uint8

	// This field is mandatory and its contents are reserved.
	Reserved2 [7]byte
}

func (fdf ExfatFileDirectoryEntry) String() string {
	return fmt.Sprintf("FileDirectoryEntry<SECONDARY-COUNT=(%d) CTIME=[%s] MTIME=[%s] ATIME=[%s]>",
		fdf.SecondaryCount,
		fdf.CreateTimestamp, fdf.LastModifiedTimestamp, fdf.LastAccessedTimestamp)
}

type ExfatSecondaryDirectoryEntry struct {
	// This field is mandatory and Section 6.4.1 defines its contents.
	//
	// The EntryType field shall conform to the definition provided in the Generic DirectoryEntry template (see Section 6.2.1)
	EntryType EntryType

	// This field is mandatory and Section 6.4.2 defines its contents.
	//
	// The GeneralSecondaryFlags field contains flags (see Table 19).
	GeneralSecondaryFlags uint8

	// This field is mandatory and structures which derive from this template define its contents.
	CustomDefined [18]byte

	// This field is mandatory and Section 6.4.3 defines its contents.
	//
	// The FirstCluster field shall conform to the definition provided in the Generic DirectoryEntry template (see Section 6.2.2).
	//
	// If the NoFatChain bit is 1 then FirstCluster must point to a valid cluster in the cluster heap.
	FirstCluster uint32

	// This field is mandatory and Section 6.4.4 defines its contents.
	//
	// The DataLength field shall conform to the definition provided in the Generic DirectoryEntry template (see Section 6.2.3).
	//
	// If the NoFatChain bit is 1 then DataLength must not be zero. If the FirstCluster field is zero, then DataLength must also be zero.
	DataLength uint64
}

func (sde ExfatSecondaryDirectoryEntry) String() string {
	return fmt.Sprintf("SecondaryDirectoryEntry<TYPE=(%d) FIRST-CLUSTER=(%d) DATA-LENGTH=(%d)>", sde.EntryType, sde.FirstCluster, sde.DataLength)
}

// func (en *ExfatNavigator) parseSecondaryEntry(raw []byte) (secondaryDe ExfatSecondaryDirectoryEntry, err error) {
// 	defer func() {
// 		if errRaw := recover(); errRaw != nil {
// 			var ok bool
// 			if err, ok = errRaw.(error); ok == true {
// 				err = log.Wrap(err)
// 			} else {
// 				err = log.Errorf("Error not an error: [%s] [%v]", reflect.TypeOf(err).Name(), err)
// 			}
// 		}
// 	}()

// 	err = restruct.Unpack(raw, defaultEncoding, &secondaryDe)
// 	log.PanicIf(err)

// 	return secondaryDe, nil
// }
