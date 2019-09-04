package exfat

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/dsoprea/go-logging"
	"github.com/go-restruct/restruct"
)

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

type ExfatPrimaryDirectoryEntry struct {
	// EntryType: This field is mandatory and Section 6.3.1 defines its contents.
	//
	// The EntryType field shall conform to the definition provided in the Generic DirectoryEntry template (see Section 6.2.1).
	EntryType EntryType

	// SecondaryCount: This field is mandatory and Section 6.3.2 defines its contents.
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
	SecondaryCount_ uint8

	// SetChecksum: This field is mandatory and Section 6.3.3 defines its contents.
	//
	// The SetChecksum field shall contain the checksum of all directory entries in the given directory entry set. However, the checksum excludes this field (see Figure 2). Implementations shall verify the contents of this field are valid prior to using any other directory entry in the given directory entry set.
	//
	// Critical primary directory entry structures which derive from this template may redefine both the SecondaryCount and SetChecksum fields.
	SetChecksum uint16

	// GeneralPrimaryFlags: This field is mandatory and Section 6.3.4 defines its contents.
	//
	// The GeneralPrimaryFlags field contains flags (see Table 17).
	//
	// Critical primary directory entry structures which derive from this template may redefine this field.
	GeneralPrimaryFlags uint16

	// CustomDefined: This field is mandatory and structures which derive from this template define its contents.
	CustomDefined [14]byte

	// FirstCluster: This field is mandatory and Section 6.3.5 defines its contents.
	//
	// The FirstCluster field shall conform to the definition provided in the Generic DirectoryEntry template (see Section 6.2.2).
	//
	// If the NoFatChain bit is 1 then FirstCluster must point to a valid cluster in the cluster heap.
	//
	// Critical primary directory entry structures which derive from this template may redefine the FirstCluster and DataLength fields. Other structures which derive from this template may redefine the FirstCluster and DataLength fields only if the AllocationPossible field contains the value 0.
	FirstCluster uint32

	// DataLength: This field is mandatory and Section 6.3.6 defines its contents.
	//
	// The DataLength field shall conform to the definition provided in the Generic DirectoryEntry template (see Section 6.2.3).
	//
	// If the NoFatChain bit is 1 then DataLength must not be zero. If the FirstCluster field is zero, then DataLength must also be zero.
	//
	// Critical primary directory entry structures which derive from this template may redefine the FirstCluster and DataLength fields. Other structures which derive from this template may redefine the FirstCluster and DataLength fields only if the AllocationPossible field contains the value 0.
	DataLength uint64
}

func (sde ExfatPrimaryDirectoryEntry) String() string {
	return fmt.Sprintf("PrimaryDirectoryEntry<TYPE=(%d) SECONDARY-COUNT=(%d) FIRST-CLUSTER=(%d) DATA-LENGTH=(%d)>", sde.EntryType, sde.SecondaryCount_, sde.FirstCluster, sde.DataLength)
}

func (sde ExfatPrimaryDirectoryEntry) Dump() {
	fmt.Printf("Primary Directory Entry\n")
	fmt.Printf("=======================\n")
	fmt.Printf("\n")

	fmt.Printf("EntryType: (%d) [%08b]\n", sde.EntryType, sde.EntryType)
	fmt.Printf("SecondaryCount: (%d)\n", sde.SecondaryCount_)
	fmt.Printf("SetChecksum: (%04x)\n", sde.SetChecksum)
	fmt.Printf("GeneralPrimaryFlags: (%04x)\n", sde.GeneralPrimaryFlags)
	fmt.Printf("FirstCluster: (%d)\n", sde.FirstCluster)
	fmt.Printf("DataLength: (%d)\n", sde.DataLength)

	fmt.Printf("\n")
}

func (sde ExfatPrimaryDirectoryEntry) SecondaryCount() uint8 {
	return sde.SecondaryCount_
}

func (ExfatPrimaryDirectoryEntry) TypeName() string {
	return "_Primary"
}

type ExfatSecondaryDirectoryEntry struct {
	// EntryType: This field is mandatory and Section 6.4.1 defines its contents.
	//
	// The EntryType field shall conform to the definition provided in the Generic DirectoryEntry template (see Section 6.2.1)
	EntryType EntryType

	// GeneralSecondaryFlags: This field is mandatory and Section 6.4.2 defines its contents.
	//
	// The GeneralSecondaryFlags field contains flags (see Table 19).
	GeneralSecondaryFlags uint8

	// CustomDefined: This field is mandatory and structures which derive from this template define its contents.
	CustomDefined [18]byte

	// FirstCluster: This field is mandatory and Section 6.4.3 defines its contents.
	//
	// The FirstCluster field shall conform to the definition provided in the Generic DirectoryEntry template (see Section 6.2.2).
	//
	// If the NoFatChain bit is 1 then FirstCluster must point to a valid cluster in the cluster heap.
	FirstCluster uint32

	// DataLength: This field is mandatory and Section 6.4.4 defines its contents.
	//
	// The DataLength field shall conform to the definition provided in the Generic DirectoryEntry template (see Section 6.2.3).
	//
	// If the NoFatChain bit is 1 then DataLength must not be zero. If the FirstCluster field is zero, then DataLength must also be zero.
	DataLength uint64
}

func (sde ExfatSecondaryDirectoryEntry) String() string {
	return fmt.Sprintf("SecondaryDirectoryEntry<TYPE=(%d) FIRST-CLUSTER=(%d) DATA-LENGTH=(%d)>", sde.EntryType, sde.FirstCluster, sde.DataLength)
}

func (ExfatSecondaryDirectoryEntry) TypeName() string {
	return "_Secondary"
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

func (et ExfatTimestamp) Timestamp() time.Time {

	// TODO(dustin): Implement the timezone.

	return time.Date(et.Year(), time.Month(et.Month()), et.Day(), et.Hour(), et.Minute(), et.Second(), 0, time.Local)
}

func (et ExfatTimestamp) String() string {
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", et.Year(), et.Month(), et.Day(), et.Hour(), et.Minute(), et.Second())
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
	CreateTimestamp ExfatTimestamp

	// LastModifiedTimestamp: This field is mandatory and Section 7.4.6 defines its contents.
	LastModifiedTimestamp ExfatTimestamp

	// LastAccessedTimestamp: This field is mandatory and Section 7.4.7 defines its contents.
	LastAccessedTimestamp ExfatTimestamp

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
		fdf.CreateTimestamp, fdf.LastModifiedTimestamp, fdf.LastAccessedTimestamp)
}

func (fdf ExfatFileDirectoryEntry) SecondaryCount() uint8 {
	return fdf.SecondaryCount_
}

func (fdf ExfatFileDirectoryEntry) TypeName() string {
	return "File"
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
	return fmt.Sprintf("UpcaseTableDirectoryEntry<TABLE-CHECKSUM=[%08x] FIRST-CLUSTER=(%d) DATA-LENGTH=(%d)>", utde.TableChecksum, utde.FirstCluster, utde.DataLength)
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
	return fmt.Sprintf("VolumeGuidDirectoryEntry<SECONDARY-COUNT=(%d) SET-CHECKSUM=(%04x) GENERAL-PRIMARY-FLAGS=(%016b) GUID=[%064x]>", vgde.SecondaryCount_, vgde.SetChecksum, vgde.GeneralPrimaryFlags, vgde.VolumeGuid)
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

type ExfatStreamExtensionDirectoryEntry struct {
	// EntryType: This field is mandatory and Section 7.6.1 defines its contents.
	EntryType EntryType

	// GeneralSecondaryFlags: This field is mandatory and Section 7.6.2 defines its contents.
	GeneralSecondaryFlags uint8

	// Reserved1: This field is mandatory and its contents are reserved.
	Reserved1 [1]byte

	// NameLength: This field is mandatory and Section 7.6.3 defines its contents.
	NameLength uint8

	// NameHash: This field is mandatory and Section 7.6.4 defines its contents.
	NameHash uint16

	// Reserved2: This field is mandatory and its contents are reserved.
	Reserved2 [2]byte

	// ValidDataLength: This field is mandatory and Section 7.6.5 defines its contents.
	ValidDataLength uint64

	// Reserved3: This field is mandatory and its contents are reserved.
	Reserved3 [4]byte

	// FirstCluster: This field is mandatory and Section 7.6.6 defines its contents.
	FirstCluster uint32

	// DataLength: This field is mandatory and Section 7.6.7 defines its contents.
	DataLength uint64
}

func (sede ExfatStreamExtensionDirectoryEntry) String() string {
	return fmt.Sprintf("StreamExtensionDirectoryEntry<GENERAL-SECONDARY-FLAGS=(%08b) NAME-LENGTH=(%d) NAME-HASH=(%04x) VALID-DATA-LENGTH=(%d) FIRST-CLUSTER=(%d) DATA-LENGTH=(%d)>",
		sede.GeneralSecondaryFlags, sede.NameLength, sede.NameHash, sede.ValidDataLength, sede.FirstCluster, sede.DataLength)
}

func (ExfatStreamExtensionDirectoryEntry) TypeName() string {
	return "StreamExtension"
}

type ExfatFileNameDirectoryEntry struct {
	// EntryType: This field is mandatory and Section 7.7.1 defines its contents.
	EntryType EntryType

	// GeneralSecondaryFlags: This field is mandatory and Section 7.7.2 defines its contents.
	GeneralSecondaryFlags uint8

	// FileName: This field is mandatory and Section 7.7.3 defines its contents.
	FileName [30]byte
}

func (fnde ExfatFileNameDirectoryEntry) String() string {
	return fmt.Sprintf("FileNameDirectoryEntry<GENERAL-SECONDARY-FLAGS=(%08b) FILENAME=[%s]>", fnde.GeneralSecondaryFlags, string(fnde.FileName[:]))
}

func (ExfatFileNameDirectoryEntry) TypeName() string {
	return "FileName"
}

type MultipartFilename []DirectoryEntry

func (mf MultipartFilename) Filename() string {
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
	GeneralSecondaryFlags uint8

	// VendorGuid: This field is mandatory and Section 7.8.3 defines its contents.
	VendorGuid [16]byte

	// VendorDefined: This field is mandatory and vendors may define its contents.
	VendorDefined [14]byte
}

func (vede ExfatVendorExtensionDirectoryEntry) String() string {
	return fmt.Sprintf("VendorExtensionDirectoryEntry<GENERAL-SECONDARY-FLAGS=(%08b) GUID=(%032x)>", vede.GeneralSecondaryFlags, vede.VendorGuid)
}

func (ExfatVendorExtensionDirectoryEntry) TypeName() string {
	return "VendorExtension"
}

type ExfatVendorAllocationDirectoryEntry struct {
	// EntryType: This field is mandatory and Section 7.9.1 defines its contents.
	EntryType EntryType

	// GeneralSecondaryFlags: This field is mandatory and Section 7.9.2 defines its contents.
	GeneralSecondaryFlags uint8

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
	return fmt.Sprintf("VendorAllocationDirectoryEntry<GENERAL-SECONDARY-FLAGS=(%08b) GUID=(%032x) VENDOR-DEFINED=(%08x) FIRST-CLUSTER=(%d) DATA-LENGTH=(%d)>", vade.GeneralSecondaryFlags, vade.VendorGuid, vade.VendorDefined, vade.FirstCluster, vade.DataLength)
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
