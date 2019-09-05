package exfat

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/dsoprea/go-logging"
	"github.com/go-restruct/restruct"
)

// TODO(dustin): Implement the timestamp timezones.

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

// DirectoryEntryParserKey describes the combination of attributes that uniquely
// identify an entry-type (`isCritical` corresponds directly to
// `TypeImportance` and `isPrimary` corresponds directly to `TypeCategory`):
//
// 	6.2.1.1 TypeCode Field
//
// 	The TypeCode field partially describes the specific type of the given directory entry. This field, plus the TypeImportance and TypeCategory fields (see Sections 6.2.1.2 and 6.2.1.3, respectively) uniquely identify the type of the given directory entry.
type DirectoryEntryParserKey struct {
	typeCode   int
	isCritical bool
	isPrimary  bool
}

func (depk DirectoryEntryParserKey) String() string {
	return fmt.Sprintf("DirectoryEntryParserKey<TYPE-CODE=(%d) IS-CRITICAL=[%v] IS-PRIMARY=[%v]>", depk.typeCode, depk.isCritical, depk.isPrimary)
}

var (
	// directoryEntryParsers expresses all entry-types describes in the exFAT
	// specification (and required *by* the specification).
	directoryEntryParsers = map[DirectoryEntryParserKey]reflect.Type{

		//// Critical primary

		// Allocation Bitmap (Section 7.1)
		DirectoryEntryParserKey{typeCode: 1, isCritical: true, isPrimary: true}: reflect.TypeOf(ExfatAllocationBitmapDirectoryEntry{}),

		// Up-case Table (Section 7.2)
		DirectoryEntryParserKey{typeCode: 2, isCritical: true, isPrimary: true}: reflect.TypeOf(ExfatUpcaseTableDirectoryEntry{}),

		// Volume Label (Section 7.3)
		DirectoryEntryParserKey{typeCode: 3, isCritical: true, isPrimary: true}: reflect.TypeOf(ExfatVolumeLabelDirectoryEntry{}),

		// File (Section 7.4)
		DirectoryEntryParserKey{typeCode: 5, isCritical: true, isPrimary: true}: reflect.TypeOf(ExfatFileDirectoryEntry{}),

		//// Benign primary

		// Volume GUID (Section 7.5)
		DirectoryEntryParserKey{typeCode: 0, isCritical: false, isPrimary: true}: reflect.TypeOf(ExfatVolumeGuidDirectoryEntry{}),

		// TexFAT Padding (Section 7.10)
		DirectoryEntryParserKey{typeCode: 1, isCritical: false, isPrimary: true}: reflect.TypeOf(ExfatTexFATDirectoryEntry{}),

		//// Critical secondary

		// Stream Extension (Section 7.6)
		DirectoryEntryParserKey{typeCode: 0, isCritical: true, isPrimary: false}: reflect.TypeOf(ExfatStreamExtensionDirectoryEntry{}),

		// File Name (Section 7.7)
		DirectoryEntryParserKey{typeCode: 1, isCritical: true, isPrimary: false}: reflect.TypeOf(ExfatFileNameDirectoryEntry{}),

		//// Benign secondary

		// Vendor Extension (Section 7.8)
		DirectoryEntryParserKey{typeCode: 0, isCritical: false, isPrimary: false}: reflect.TypeOf(ExfatVendorExtensionDirectoryEntry{}),

		// Vendor Allocation (Section 7.9)
		DirectoryEntryParserKey{typeCode: 1, isCritical: false, isPrimary: false}: reflect.TypeOf(ExfatVendorAllocationDirectoryEntry{}),
	}
)

type DirectoryEntry interface {
	TypeName() string
}

type PrimaryDirectoryEntry interface {
	SecondaryCount() uint8
}

type DumpableDirectoryEntry interface {
	Dump()
}

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

func (et ExfatTimestamp) TimestampWithOffset(offset int) time.Time {
	location := time.FixedZone(fmt.Sprintf("(off=%d)", offset), offset)

	return time.Date(et.Year(), time.Month(et.Month()), et.Day(), et.Hour(), et.Minute(), et.Second(), 0, location)
}

type FileAttributes uint16

func (fa FileAttributes) IsReadOnly() bool {
	return fa&1 > 0
}

func (fa FileAttributes) IsHidden() bool {
	return fa&2 > 0
}

func (fa FileAttributes) IsSystem() bool {
	return fa&4 > 0
}

func (fa FileAttributes) IsDirectory() bool {
	return fa&16 > 0
}

func (fa FileAttributes) IsArchive() bool {
	return fa&32 > 0
}

func (fa FileAttributes) String() string {
	return fmt.Sprintf("FileAttributes<IS-READONLY=[%v] IS-HIDDEN=[%v] IS-SYSTEM=[%v] IS-DIRECTORY=[%v] IS-ARCHIVE=[%v]>",
		fa.IsReadOnly(), fa.IsHidden(), fa.IsSystem(), fa.IsDirectory(), fa.IsArchive())
}

func (fa FileAttributes) DumpBareIndented(indent string) {
	fmt.Printf("%sRead Only? [%v]\n", indent, fa.IsReadOnly())
	fmt.Printf("%sHidden? [%v]\n", indent, fa.IsHidden())
	fmt.Printf("%sSystem? [%v]\n", indent, fa.IsSystem())
	fmt.Printf("%sDirectory? [%v]\n", indent, fa.IsDirectory())
	fmt.Printf("%sArchive? [%v]\n", indent, fa.IsArchive())
}

type ExfatFileDirectoryEntry struct {
	// EntryType: This field is mandatory and Section 7.4.1 defines its contents.
	EntryType EntryType

	// SecondaryCount: This field is mandatory and Section 7.4.2 defines its contents.
	SecondaryCount_ uint8

	// SetChecksum: This field is mandatory and Section 7.4.3 defines its contents.
	SetChecksum uint16

	// FileAttributes: This field is mandatory and Section 7.4.4 defines its contents.
	FileAttributes FileAttributes

	// Reserved1: This field is mandatory and its contents are reserved.
	Reserved1 uint16

	// CreateTimestamp: This field is mandatory and Section 7.4.5 defines its contents.
	CreateTimestamp_ ExfatTimestamp

	// LastModifiedTimestamp: This field is mandatory and Section 7.4.6 defines its contents.
	LastModifiedTimestamp_ ExfatTimestamp

	// LastAccessedTimestamp: This field is mandatory and Section 7.4.7 defines its contents.
	LastAccessedTimestamp_ ExfatTimestamp

	// Create10msIncrement: This field is mandatory and Section 7.4.5 defines its contents.
	Create10msIncrement uint8

	// LastModified10msIncrement: This field is mandatory and Section 7.4.6 defines its contents.
	LastModified10msIncrement uint8

	// CreateUtcOffset: This field is mandatory and Section 7.4.5 defines its contents.
	CreateUtcOffset uint8

	// LastModifiedUtcOffset: This field is mandatory and Section 7.4.6 defines its contents.
	LastModifiedUtcOffset uint8

	// LastAccessedUtcOffset: This field is mandatory and Section 7.4.7 defines its contents.
	LastAccessedUtcOffset uint8

	// Reserved2: This field is mandatory and its contents are reserved.
	Reserved2 [7]byte
}

func (fdf ExfatFileDirectoryEntry) String() string {
	return fmt.Sprintf("FileDirectoryEntry<SECONDARY-COUNT=(%d) CTIME=[%s] MTIME=[%s] ATIME=[%s]>",
		fdf.SecondaryCount_,
		fdf.CreateTimestamp(), fdf.LastModifiedTimestamp(), fdf.LastAccessedTimestamp())
}

func (fdf ExfatFileDirectoryEntry) SecondaryCount() uint8 {
	return fdf.SecondaryCount_
}

func (fdf ExfatFileDirectoryEntry) TypeName() string {
	return "File"
}

func (fdf ExfatFileDirectoryEntry) CreateTimestamp() time.Time {
	return fdf.CreateTimestamp_.TimestampWithOffset(int(fdf.CreateUtcOffset))
}

func (fdf ExfatFileDirectoryEntry) LastModifiedTimestamp() time.Time {
	return fdf.LastModifiedTimestamp_.TimestampWithOffset(int(fdf.LastModifiedUtcOffset))
}

func (fdf ExfatFileDirectoryEntry) LastAccessedTimestamp() time.Time {
	return fdf.LastAccessedTimestamp_.TimestampWithOffset(int(fdf.LastAccessedUtcOffset))
}

func (fdf ExfatFileDirectoryEntry) Dump() {
	fmt.Printf("File Directory Entry\n")
	fmt.Printf("====================\n")
	fmt.Printf("\n")

	fmt.Printf("SecondaryCount: (%d)\n", fdf.SecondaryCount())
	fmt.Printf("SetChecksum: (0x%04x)\n", fdf.SetChecksum)
	fmt.Printf("CreateTimestamp: [%s]\n", fdf.CreateTimestamp())
	fmt.Printf("LastModifiedTimestamp: [%s]\n", fdf.LastModifiedTimestamp())
	fmt.Printf("LastAccessedTimestamp: [%s]\n", fdf.LastAccessedTimestamp())
	fmt.Printf("\n")

	fmt.Printf("Attributes:\n")

	fdf.FileAttributes.DumpBareIndented("  ")

	fmt.Printf("\n")
}

type ExfatAllocationBitmapDirectoryEntry struct {
	// EntryType: This field is mandatory and Section 7.1.1 defines its contents.
	EntryType EntryType

	// BitmapFlags: This field is mandatory and Section 7.1.2 defines its contents.
	BitmapFlags uint8

	// Reserved: This field is mandatory and its contents are reserved.
	Reserved [18]byte

	// FirstCluster: This field is mandatory and Section 7.1.3 defines its contents.
	FirstCluster uint32

	// DataLength: This field is mandatory and Section 7.1.4 defines its contents.
	DataLength uint64
}

func (abde ExfatAllocationBitmapDirectoryEntry) String() string {
	return fmt.Sprintf("AllocationBitmapDirectoryEntry<BITMAP-FLAGS=[%08b] FIRST-CLUSTER=(%d) DATA-LENGTH=(%d)>", abde.BitmapFlags, abde.FirstCluster, abde.DataLength)
}

func (ExfatAllocationBitmapDirectoryEntry) TypeName() string {
	return "AllocationBitmap"
}

type ExfatUpcaseTableDirectoryEntry struct {
	// EntryType: This field is mandatory and Section 7.2.1 defines its contents.
	EntryType EntryType

	// Reserved1: This field is mandatory and its contents are reserved.
	Reserved1 [3]byte

	// TableChecksum: This field is mandatory and Section 7.2.2 defines its contents.
	TableChecksum uint32

	// Reserved2: This field is mandatory and its contents are reserved.
	Reserved2 [12]byte

	// FirstCluster: This field is mandatory and Section 7.2.3 defines its contents.
	FirstCluster uint32

	// DataLength: This field is mandatory and Section 7.2.4 defines its contents.
	DataLength uint64
}

func (utde ExfatUpcaseTableDirectoryEntry) String() string {
	return fmt.Sprintf("UpcaseTableDirectoryEntry<TABLE-CHECKSUM=[0x%08x] FIRST-CLUSTER=(%d) DATA-LENGTH=(%d)>", utde.TableChecksum, utde.FirstCluster, utde.DataLength)
}

func (ExfatUpcaseTableDirectoryEntry) TypeName() string {
	return "UpcaseTable"
}

type ExfatVolumeLabelDirectoryEntry struct {
	// EntryType: This field is mandatory and Section 7.3.1 defines its contents.
	EntryType EntryType

	// CharacterCount: This field is mandatory and Section 7.3.2 defines its contents.
	CharacterCount uint8

	// VolumeLabel: This field is mandatory and Section 7.3.3 defines its contents.
	//
	// NOTES
	//
	// - The specification states that this is Unicode (naturally):
	//
	// 		The VolumeLabel field shall contain a Unicode string, which is the
	// 		user-friendly name of the volume. The VolumeLabel field has the same
	// 		set of invalid characters as the FileName field of the File Name
	// 		directory entry (see Section 7.7.3).
	//
	// - In practice, tools will combine both the `VolumeLabel` and `Reserved`
	//   fields. So, we combine them here.
	VolumeLabel [30]byte

	// // VolumeLabel: This field is mandatory and Section 7.3.3 defines its contents.
	// VolumeLabel [22]byte

	// // Reserved: This field is mandatory and its contents are reserved.
	// Reserved [8]byte
}

func (vlde ExfatVolumeLabelDirectoryEntry) Label() string {
	// `VolumeLabel` is a Unicode-encoded string and the character-count
	// corresponds to the number of Unicode characters.

	decodedString := UnicodeFromAscii(vlde.VolumeLabel[:], int(vlde.CharacterCount))
	return decodedString
}

func (vlde ExfatVolumeLabelDirectoryEntry) String() string {
	return fmt.Sprintf("VolumeLabelDirectoryEntry<CHARACTER-COUNT=(%d) LABEL=[%s]>", vlde.CharacterCount, vlde.Label())
}

func (ExfatVolumeLabelDirectoryEntry) TypeName() string {
	return "VolumeLabel"
}

type ExfatVolumeGuidDirectoryEntry struct {
	// EntryType: This field is mandatory and Section 7.5.1 defines its contents.
	EntryType EntryType

	// SecondaryCount: This field is mandatory and Section 7.5.2 defines its contents.
	SecondaryCount_ uint8

	// SetChecksum: This field is mandatory and Section 7.5.3 defines its contents.
	SetChecksum uint16

	// GeneralPrimaryFlags: This field is mandatory and Section 7.5.4 defines its contents.
	GeneralPrimaryFlags uint16

	// VolumeGuid: This field is mandatory and Section 7.5.5 defines its contents.
	VolumeGuid [16]byte

	// Reserved: This field is mandatory and its contents are reserved.
	Reserved [10]byte
}

func (vgde ExfatVolumeGuidDirectoryEntry) String() string {
	return fmt.Sprintf("VolumeGuidDirectoryEntry<SECONDARY-COUNT=(%d) SET-CHECKSUM=(0x%04x) GENERAL-PRIMARY-FLAGS=(%016b) GUID=[0x%064x]>", vgde.SecondaryCount_, vgde.SetChecksum, vgde.GeneralPrimaryFlags, vgde.VolumeGuid)
}

func (vgde ExfatVolumeGuidDirectoryEntry) SecondaryCount() uint8 {
	return vgde.SecondaryCount_
}

func (ExfatVolumeGuidDirectoryEntry) TypeName() string {
	return "VolumeGuid"
}

type ExfatTexFATDirectoryEntry struct {
	// Reserved: Not covered by the exFAT specification. Just mask the whole thing as reserved.
	Reserved [32]byte
}

func (ExfatTexFATDirectoryEntry) String() string {
	return "TexFATDirectoryEntry<>"
}

func (ExfatTexFATDirectoryEntry) TypeName() string {
	return "TexFAT"
}

type GeneralSecondaryFlags uint8

func (gsf GeneralSecondaryFlags) IsAllocationPossible() bool {
	return gsf&1 > 0
}

func (gsf GeneralSecondaryFlags) NoFatChain() bool {
	return gsf&2 > 0
}

func (gsf GeneralSecondaryFlags) String() string {
	return fmt.Sprintf("GeneralSecondaryFlags<IsAllocationPossible=[%v] NoFatChain=[%v]>",
		gsf.IsAllocationPossible(), gsf.NoFatChain())
}

func (gsf GeneralSecondaryFlags) DumpBareIndented(indent string) {
	fmt.Printf("%sRaw Value: (%08b)\n", indent, gsf)
	fmt.Printf("%sIsAllocationPossible: [%v]\n", indent, gsf.IsAllocationPossible())
	fmt.Printf("%sNoFatChain: [%v]\n", indent, gsf.NoFatChain())
}

type ExfatStreamExtensionDirectoryEntry struct {

	// TODO(dustin): It's unclear where the names for the one or more streams under each file are stored.

	// EntryType: This field is mandatory and Section 7.6.1 defines its contents.
	EntryType EntryType

	// GeneralSecondaryFlags: This field is mandatory and Section 7.6.2 defines its contents.
	GeneralSecondaryFlags GeneralSecondaryFlags

	// Reserved1: This field is mandatory and its contents are reserved.
	Reserved1 [1]byte

	// NameLength: This field is mandatory and Section 7.6.3 defines its contents.
	NameLength uint8

	// NameHash: This field is mandatory and Section 7.6.4 defines its contents.
	NameHash uint16

	// Reserved2: This field is mandatory and its contents are reserved.
	Reserved2 [2]byte

	// ValidDataLength: This field is mandatory and Section 7.6.5 defines its contents.
	//
	// NOTES
	//
	// - For files, `ValidDataLength` is the real amount of data. Ostensibly,
	//   subsequent updates to a file don't necessarily have to occupy as much
	//   space as is already allocated and this describes the actual data size.
	//   For directories, only `DataLength` should be considered.
	//
	//   From the spec (7.6.5 ValidDataLength Field):
	//
	//   	The ValidDataLength field shall describe how far into the data
	//   	stream user data has been written. Implementations shall update this
	//   	field as they write data further out into the data stream. On the
	//   	storage media, the data between the valid data length and the data
	//   	length of the data stream is undefined. Implementations shall return
	//   	zeroes for read operations beyond the valid data length.
	//
	//   	If the corresponding File directory entry describes a directory,
	//   	then the only valid value for this field is equal to the value of
	//   	the DataLength field.
	//
	ValidDataLength uint64

	// Reserved3: This field is mandatory and its contents are reserved.
	Reserved3 [4]byte

	// FirstCluster: This field is mandatory and Section 7.6.6 defines its contents.
	//
	// NOTES
	//
	// - If a directory, this cluster has all of the subdirectories and files
	//   for that directory (formatted the same as the root directory located at
	//   cluster FirstClusterOfRootDirectory).
	//
	FirstCluster uint32

	// DataLength: This field is mandatory and Section 7.6.7 defines its contents.
	DataLength uint64
}

func (sede ExfatStreamExtensionDirectoryEntry) String() string {
	return fmt.Sprintf("StreamExtensionDirectoryEntry<GENERAL-SECONDARY-FLAGS=(%08b) NAME-LENGTH=(%d) NAME-HASH=(0x%04x) VALID-DATA-LENGTH=(%d) FIRST-CLUSTER=(%d) DATA-LENGTH=(%d)>",
		sede.GeneralSecondaryFlags, sede.NameLength, sede.NameHash, sede.ValidDataLength, sede.FirstCluster, sede.DataLength)
}

func (sede ExfatStreamExtensionDirectoryEntry) Dump() {
	fmt.Printf("Stream Extension Directory Entry\n")
	fmt.Printf("================================\n")
	fmt.Printf("\n")

	fmt.Printf("NameLength: (%d)\n", sede.NameLength)
	fmt.Printf("NameHash: (0x%04x)\n", sede.NameHash)
	fmt.Printf("ValidDataLength: (%d)\n", sede.ValidDataLength)
	fmt.Printf("FirstCluster: (%d)\n", sede.FirstCluster)
	fmt.Printf("DataLength: (%d)\n", sede.DataLength)
	fmt.Printf("\n")

	fmt.Printf("General secondary flags:\n")
	sede.GeneralSecondaryFlags.DumpBareIndented("  ")

	fmt.Printf("\n")
}

func (ExfatStreamExtensionDirectoryEntry) TypeName() string {
	return "StreamExtension"
}

type ExfatFileNameDirectoryEntry struct {
	// EntryType: This field is mandatory and Section 7.7.1 defines its contents.
	EntryType EntryType

	// GeneralSecondaryFlags: This field is mandatory and Section 7.7.2 defines its contents.
	GeneralSecondaryFlags GeneralSecondaryFlags

	// FileName: This field is mandatory and Section 7.7.3 defines its contents.
	FileName [30]byte
}

func (fnde ExfatFileNameDirectoryEntry) String() string {
	return fmt.Sprintf("FileNameDirectoryEntry<GENERAL-SECONDARY-FLAGS=(%08b) FILENAME=%v>", fnde.GeneralSecondaryFlags, fnde.FileName[:])
}

func (ExfatFileNameDirectoryEntry) TypeName() string {
	return "FileName"
}

type MultipartFilename []DirectoryEntry

func (mf MultipartFilename) Filename() string {

	// NOTE(dustin): The total filename length is specified in the "Stream
	// Extension" directory entry that occurs after the primary file entry and
	// before these file-name directory-entries, but we don't implement/
	// validate that count, here.

	parts := make([]string, 0)

	for _, deRaw := range mf {
		if fnde, ok := deRaw.(*ExfatFileNameDirectoryEntry); ok == true {
			part := UnicodeFromAscii(fnde.FileName[:], 15)
			parts = append(parts, part)
		}
	}

	filename := strings.Join(parts, "")

	return filename
}

type ExfatVendorExtensionDirectoryEntry struct {
	// EntryType: This field is mandatory and Section 7.8.1 defines its contents.
	EntryType EntryType

	// GeneralSecondaryFlags: This field is mandatory and Section 7.8.2 defines its contents.
	GeneralSecondaryFlags GeneralSecondaryFlags

	// VendorGuid: This field is mandatory and Section 7.8.3 defines its contents.
	VendorGuid [16]byte

	// VendorDefined: This field is mandatory and vendors may define its contents.
	VendorDefined [14]byte
}

func (vede ExfatVendorExtensionDirectoryEntry) String() string {
	return fmt.Sprintf("VendorExtensionDirectoryEntry<GENERAL-SECONDARY-FLAGS=(%08b) GUID=(0x%032x)>", vede.GeneralSecondaryFlags, vede.VendorGuid)
}

func (ExfatVendorExtensionDirectoryEntry) TypeName() string {
	return "VendorExtension"
}

type ExfatVendorAllocationDirectoryEntry struct {
	// EntryType: This field is mandatory and Section 7.9.1 defines its contents.
	EntryType EntryType

	// GeneralSecondaryFlags: This field is mandatory and Section 7.9.2 defines its contents.
	GeneralSecondaryFlags GeneralSecondaryFlags

	// VendorGuid: This field is mandatory and Section 7.9.3 defines its contents.
	VendorGuid [16]byte

	// VendorDefined: This field is mandatory and vendors may define its contents.
	VendorDefined [2]byte

	// FirstCluster: This field is mandatory and Section 7.9.4 defines its contents.
	FirstCluster uint32

	// DataLength: This field is mandatory and Section 7.9.5 defines its contents.
	DataLength uint64
}

func (vade ExfatVendorAllocationDirectoryEntry) String() string {
	return fmt.Sprintf("VendorAllocationDirectoryEntry<GENERAL-SECONDARY-FLAGS=(%08b) GUID=(0x%032x) VENDOR-DEFINED=(0x%08x) FIRST-CLUSTER=(%d) DATA-LENGTH=(%d)>", vade.GeneralSecondaryFlags, vade.VendorGuid, vade.VendorDefined, vade.FirstCluster, vade.DataLength)
}

func (ExfatVendorAllocationDirectoryEntry) TypeName() string {
	return "VendorAllocation"
}

func parseDirectoryEntry(entryType EntryType, directoryEntryData []byte) (parsed DirectoryEntry, err error) {
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

	depk := DirectoryEntryParserKey{
		typeCode:   entryType.TypeCode(),
		isCritical: entryType.IsCritical(),
		isPrimary:  entryType.IsPrimary(),
	}

	structType, found := directoryEntryParsers[depk]
	if found == false {
		log.Panicf("no struct-type recorded for entry-type: %s", depk)
	}

	s := reflect.New(structType)
	x := s.Interface()

	err = restruct.Unpack(directoryEntryData, defaultEncoding, x)
	log.PanicIf(err)

	return x.(DirectoryEntry), nil
}
