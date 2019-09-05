// This package manages the low-level, on-disk storage structures.

package exfat

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"

	"encoding/binary"

	"github.com/dsoprea/go-logging"
	"github.com/go-restruct/restruct"
)

const (
	bootSectorHeaderSize        = 512
	oemParametersSize           = 48 * 10
	mainExtendedBootSectorCount = 8
)

var (
	requiredJumpBootSignature     = []byte{0xeb, 0x76, 0x90}
	requiredFileSystemName        = []byte("EXFAT   ")
	requiredBootSignature         = uint16(0xaa55)
	requiredExtendedBootSignature = uint32(0xaa550000)
)

type bootRegion struct {
	bsh        BootSectorHeader
	sectorSize uint32
}

type ExfatReader struct {
	rs io.ReadSeeker

	bootRegion bootRegion

	activeFat Fat
}

func NewExfatReader(rs io.ReadSeeker) *ExfatReader {
	return &ExfatReader{
		rs: rs,
	}
}

func (er *ExfatReader) parseN(byteCount int, x interface{}) (err error) {
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

	raw := make([]byte, byteCount)

	_, err = io.ReadFull(er.rs, raw)
	log.PanicIf(err)

	err = restruct.Unpack(raw, defaultEncoding, x)
	log.PanicIf(err)

	return nil
}

type BootSectorHeader struct {
	// This field is mandatory and Section 3.1.1 defines its contents.
	//
	// The JumpBoot field shall contain the jump instruction for CPUs common in personal computers, which, when executed, "jumps" the CPU to execute the boot-strapping instructions in the BootCode field.
	//
	// The valid value for this field is (in order of low-order byte to high-order byte) EBh 76h 90h.
	JumpBoot [3]byte

	// This field is mandatory and Section 3.1.2 defines its contents.
	//
	// The FileSystemName field shall contain the name of the file system on the volume.
	//
	// The valid value for this field is, in ASCII characters, "EXFAT   ", which includes three trailing white spaces.
	FileSystemName [8]byte

	// This field is mandatory and Section 3.1.3 defines its contents.
	//
	// The MustBeZero field shall directly correspond with the range of bytes the packed BIOS parameter block consumes on FAT12/16/32 volumes.
	//
	// The valid value for this field is 0, which helps to prevent FAT12/16/32 implementations from mistakenly mounting an exFAT volume.
	MustBeZero [53]byte

	// This field is mandatory and Section 3.1.4 defines its contents.
	//
	// The PartitionOffset field shall describe the media-relative sector offset of the partition which hosts the given exFAT volume. This field aids boot-strapping from the volume using extended INT 13h on personal computers.
	//
	// All possible values for this field are valid; however, the value 0 indicates implementations shall ignore this field.
	PartitionOffset uint64

	// This field is mandatory and Section 3.1.5 defines its contents.
	//
	// The VolumeLength field shall describe the size of the given exFAT volume in sectors.
	//
	// The valid range of values for this field shall be:
	//
	// At least 220/ 2BytesPerSectorShift, which ensures the smallest volume is no less than 1MB
	//
	// At most 264- 1, the largest value this field can describe
	//
	// However, if the size of the Excess Space sub-region is 0, then the value of this field is ClusterHeapOffset + (232- 11) * 2SectorsPerClusterShift.
	VolumeLength uint64

	// This field is mandatory and Section 3.1.6 defines its contents.
	//
	// The FatOffset field shall describe the volume-relative sector offset of the First FAT. This field enables implementations to align the First FAT to the characteristics of the underlying storage media.
	//
	// The valid range of values for this field shall be:
	//
	// At least 24, which accounts for the sectors the Main Boot and Backup Boot regions consume
	//
	// At most ClusterHeapOffset - (FatLength * NumberOfFats), which accounts for the sectors the Cluster Heap consumes
	FatOffset uint32

	// This field is mandatory and Section 3.1.7 defines its contents.
	//
	// The FatLength field shall describe the length, in sectors, of each FAT table (the volume may contain up to two FATs).
	//
	// The valid range of values for this field shall be:
	//
	// At least (ClusterCount + 2) * 22/ 2BytesPerSectorShiftrounded up to the nearest integer, which ensures each FAT has sufficient space for describing all the clusters in the Cluster Heap
	//
	// At most (ClusterHeapOffset - FatOffset) / NumberOfFats rounded down to the nearest integer, which ensures the FATs exist before the Cluster Heap
	//
	// This field may contain a value in excess of its lower bound (as described above) to enable the Second FAT, if present, to also be aligned to the characteristics of the underlying storage media. The contents of the space which exceeds what the FAT itself requires, if any, are undefined.
	FatLength uint32

	// This field is mandatory and Section 3.1.8 defines its contents.
	//
	// The ClusterHeapOffset field shall describe the volume-relative sector offset of the Cluster Heap. This field enables implementations to align the Cluster Heap to the characteristics of the underlying storage media.
	//
	// The valid range of values for this field shall be:
	//
	// At least FatOffset + FatLength * NumberOfFats, to account for the sectors all the preceding regions consume
	//
	// At most 232- 1 or VolumeLength - (ClusterCount * 2SectorsPerClusterShift), whichever calculation is less
	ClusterHeapOffset uint32

	// This field is mandatory and Section 3.1.9 defines its contents.
	//
	// The ClusterCount field shall describe the number of clusters the Cluster Heap contains.
	//
	// The valid value for this field shall be the lesser of the following:
	//
	// (VolumeLength - ClusterHeapOffset) / 2SectorsPerClusterShiftrounded down to the nearest integer, which is exactly the number of clusters which can fit between the beginning of the Cluster Heap and the end of the volume
	//
	// 232- 11, which is the maximum number of clusters a FAT can describe
	//
	// The value of the ClusterCount field determines the minimum size of a FAT. To avoid extremely large FATs, implementations can control the number of clusters in the Cluster Heap by increasing the cluster size (via the SectorsPerClusterShift field). This specification recommends no more than 224- 2 clusters in the Cluster Heap. However, implementations shall be able to handle volumes with up to 232- 11 clusters in the Cluster Heap.
	ClusterCount uint32

	// This field is mandatory and Section 3.1.10 defines its contents.
	//
	// The FirstClusterOfRootDirectory field shall contain the cluster index of the first cluster of the root directory. Implementations should make every effort to place the first cluster of the root directory in the first non-bad cluster after the clusters the Allocation Bitmap and Up-case Table consume.
	//
	// The valid range of values for this field shall be:
	//
	// At least 2, the index of the first cluster in the Cluster Heap
	//
	// At most ClusterCount + 1, the index of the last cluster in the Cluster Heap
	FirstClusterOfRootDirectory uint32

	// This field is mandatory and Section 3.1.11 defines its contents.
	//
	// The VolumeSerialNumber field shall contain a unique serial number. This assists implementations to distinguish among different exFAT volumes. Implementations should generate the serial number by combining the date and time of formatting the exFAT volume. The mechanism for combining date and time to form a serial number is implementation-specific.
	//
	// All possible values for this field are valid.
	VolumeSerialNumber uint32

	// This field is mandatory and Section 3.1.12 defines its contents.
	//
	// The FileSystemRevision field shall describe the major and minor revision numbers of the exFAT structures on the given volume.
	//
	// The high-order byte is the major revision number and the low-order byte is the minor revision number. For example, if the high-order byte contains the value 01h and if the low-order byte contains the value 05h, then the FileSystemRevision field describes the revision number 1.05. Likewise, if the high-order byte contains the value 0Ah and if the low-order byte contains the value 0Fh, then the FileSystemRevision field describes the revision number 10.15.
	//
	// The valid range of values for this field shall be:
	//
	// At least 0 for the low-order byte and 1 for the high-order byte
	//
	// At most 99 for the low-order byte and 99 for the high-order byte
	//
	// The revision number of exFAT this specification describes is 1.00. Implementations of this specification should mount any exFAT volume with major revision number 1 and shall not mount any exFAT volume with any other major revision number. Implementations shall honor the minor revision number and shall not perform operations or create any file system structures not described in the given minor revision number's corresponding specification.
	FileSystemRevision [2]uint8

	// This field is mandatory and Section 3.1.13 defines its contents.
	//
	// The VolumeFlags field shall contain flags which indicate the status of various file system structures on the exFAT volume (see Table 5).
	//
	// Implementations shall not include this field when computing its respective Main Boot or Backup Boot region checksum. When referring to the Backup Boot Sector, implementations shall treat this field as stale.
	VolumeFlags uint16

	// This field is mandatory and Section 3.1.14 defines its contents.
	//
	// The BytesPerSectorShift field shall describe the bytes per sector expressed as log~2~(N), where N is the number of bytes per sector. For example, for 512 bytes per sector, the value of this field is 9.
	//
	// The valid range of values for this field shall be:
	//
	// At least 9 (sector size of 512 bytes), which is the smallest sector possible for an exFAT volume
	//
	// At most 12 (sector size of 4096 bytes), which is the memory page size of CPUs common in personal computers
	BytesPerSectorShift uint8

	// This field is mandatory and Section 3.1.15 defines its contents.
	//
	// The SectorsPerClusterShift field shall describe the sectors per cluster expressed as log~2~(N), where N is number of sectors per cluster. For example, for 8 sectors per cluster, the value of this field is 3.
	//
	// The valid range of values for this field shall be:
	//
	// At least 0 (1 sector per cluster), which is the smallest cluster possible
	//
	// At most 25 - BytesPerSectorShift, which evaluates to a cluster size of 32MB
	SectorsPerClusterShift uint8

	// This field is mandatory and Section 3.1.16 defines its contents.
	//
	// The NumberOfFats field shall describe the number of FATs and Allocation Bitmaps the volume contains.
	//
	// The valid range of values for this field shall be:
	//
	// 1, which indicates the volume only contains the First FAT and First Allocation Bitmap
	//
	// 2, which indicates the volume contains the First FAT, Second FAT, First Allocation Bitmap, and Second Allocation Bitmap; this value is only valid for TexFAT volumes
	NumberOfFats uint8

	// This field is mandatory and Section 3.1.17 defines its contents.
	//
	// The DriveSelect field shall contain the extended INT 13h drive number, which aids boot-strapping from this volume using extended INT 13h on personal computers.
	//
	// All possible values for this field are valid. Similar fields in previous FAT-based file systems frequently contained the value 80h.
	DriveSelect uint8

	// This field is mandatory and Section 3.1.18 defines its contents.
	//
	// The PercentInUse field shall describe the percentage of clusters in the Cluster Heap which are allocated.
	//
	// The valid range of values for this field shall be:
	//
	// Between 0 and 100 inclusively, which is the percentage of allocated clusters in the Cluster Heap, rounded down to the nearest integer
	//
	// Exactly FFh, which indicates the percentage of allocated clusters in the Cluster Heap is not available
	//
	// Implementations shall change the value of this field to reflect changes in the allocation of clusters in the Cluster Heap or shall change it to FFh.
	//
	// Implementations shall not include this field when computing its respective Main Boot or Backup Boot region checksum. When referring to the Backup Boot Sector, implementations shall treat this field as stale.
	PercentInUse uint8

	// This field is mandatory and its contents are reserved.
	Reserved [7]byte

	// This field is mandatory and Section 3.1.19 defines its contents.
	//
	// The BootCode field shall contain boot-strapping instructions. Implementations may populate this field with the CPU instructions necessary for boot-strapping a computer system. Implementations which don't provide boot-strapping instructions shall initialize each byte in this field to F4h (the halt instruction for CPUs common in personal computers) as part of their format operation.
	BootCode [390]byte

	// This field is mandatory and Section 3.1.20 defines its contents.
	//
	// The BootSignature field shall describe whether the intent of a given sector is for it to be a Boot Sector or not.
	//
	// The valid value for this field is AA55h. Any other value in this field invalidates its respective Boot Sector. Implementations should verify the contents of this field prior to depending on any other field in its respective Boot Sector.
	BootSignature uint16
}

type VolumeFlags int

const (
	// This field is mandatory and Section 3.1.13.1 defines its contents.
	//
	// The ActiveFat field shall describe which FAT and Allocation Bitmap are active (and implementations shall use), as follows:
	//
	// 0, which means the First FAT and First Allocation Bitmap are active
	//
	// 1, which means the Second FAT and Second Allocation Bitmap are active and is possible only when the NumberOfFats field contains the value 2
	//
	// Implementations shall consider the inactive FAT and Allocation Bitmap as stale. Only TexFAT-aware implementations shall switch the active FAT and Allocation Bitmaps (see Section 7.1).
	VolumeFlagActiveFat VolumeFlags = 1

	// This field is mandatory and Section 3.1.13.2 defines its contents.
	//
	// The VolumeDirty field shall describe whether the volume is dirty or not, as follows:
	//
	// 0, which means the volume is probably in a consistent state
	//
	// 1, which means the volume is probably in an inconsistent state
	//
	// Implementations should set the value of this field to 1 upon encountering file system metadata inconsistencies which they do not resolve. If, upon mounting a volume, the value of this field is 1, only implementations which resolve file system metadata inconsistencies may clear the value of this field to 0. Such implementations shall only clear the value of this field to 0 after ensuring the file system is in a consistent state.
	//
	// If, upon mounting a volume, the value of this field is 0, implementations should set this field to 1 before updating file system metadata and clear this field to 0 afterwards, similar to the recommended write ordering described in Section 8.1.
	VolumeFlagVolumeDirty = 2

	// This field is mandatory and Section 3.1.13.3 defines its contents.
	//
	// The MediaFailure field shall describe whether an implementation has discovered media failures or not, as follows:
	//
	// 0, which means the hosting media has not reported failures or any known failures are already recorded in the FAT as "bad" clusters
	//
	// 1, which means the hosting media has reported failures (i.e. has failed read or write operations)
	//
	// An implementation should set this field to 1 when:
	//
	// The hosting media fails access attempts to any region in the volume
	//
	// The implementation has exhausted access retry algorithms, if any
	//
	// If, upon mounting a volume, the value of this field is 1, implementations which scan the entire volume for media failures and record all failures as "bad" clusters in the FAT (or otherwise resolve media failures) may clear the value of this field to 0.
	VolumeFlagsMediaFailure = 4

	// This field is mandatory and Section 3.1.13.4 defines its contents.
	//
	// 3.1.13.4 ClearToZero Field
	// The ClearToZero field does not have significant meaning in this specification.
	//
	// The valid values for this field are:
	//
	// 0, which does not have any particular meaning
	//
	// 1, which means implementations shall clear this field to 0 prior to modifying any file system structures, directories, or files
	VolumeFlagClearToZero = 8
)

func (bsh BootSectorHeader) UseFirstFat() bool {
	return VolumeFlags(bsh.VolumeFlags)&VolumeFlagActiveFat == 0
}

func (bsh BootSectorHeader) UseSecondFat() bool {
	return VolumeFlags(bsh.VolumeFlags)&VolumeFlagActiveFat > 0
}

func (bsh BootSectorHeader) SectorSize() uint32 {
	return uint32(math.Pow(2, float64(bsh.BytesPerSectorShift)))
}

func (bsh BootSectorHeader) SectorsPerCluster() uint32 {
	return uint32(math.Pow(float64(2), float64(bsh.SectorsPerClusterShift)))
}

func (bsh BootSectorHeader) Dump() {
	fmt.Printf("Boot Sector Header\n")
	fmt.Printf("==================\n")
	fmt.Printf("\n")

	fmt.Printf("PartitionOffset: (%d)\n", bsh.PartitionOffset)
	fmt.Printf("VolumeLength: (%d)\n", bsh.VolumeLength)
	fmt.Printf("FatOffset: (%d)\n", bsh.FatOffset)
	fmt.Printf("FatLength: (%d)\n", bsh.FatLength)
	fmt.Printf("ClusterHeapOffset: (%d)\n", bsh.ClusterHeapOffset)
	fmt.Printf("ClusterCount: (%d)\n", bsh.ClusterCount)
	fmt.Printf("FirstClusterOfRootDirectory: (%d)\n", bsh.FirstClusterOfRootDirectory)
	fmt.Printf("VolumeSerialNumber: (0x%08x)\n", bsh.VolumeSerialNumber)
	fmt.Printf("FileSystemRevision: (0x%02x) (0x%02x)\n", bsh.FileSystemRevision[0], bsh.FileSystemRevision[1])
	fmt.Printf("BytesPerSectorShift: (%d)\n", bsh.BytesPerSectorShift)
	fmt.Printf("-> Sector-size: 2^(%d) -> %d\n", bsh.BytesPerSectorShift, bsh.SectorSize())
	fmt.Printf("SectorsPerClusterShift: (%d)\n", bsh.SectorsPerClusterShift)
	fmt.Printf("-> Sectors-per-cluster: 2^(%d) -> %d\n", bsh.SectorsPerClusterShift, bsh.SectorsPerCluster())
	fmt.Printf("NumberOfFats: (%d)\n", bsh.NumberOfFats)
	fmt.Printf("DriveSelect: (%d)\n", bsh.DriveSelect)
	fmt.Printf("PercentInUse: (%d)\n", bsh.PercentInUse)
	fmt.Printf("\n")

	fmt.Printf("VolumeFlags: (%d)\n", bsh.VolumeFlags)
	fmt.Printf("- VolumeFlagActiveFat: [%v]\n", VolumeFlags(bsh.VolumeFlags)&VolumeFlagActiveFat > 0)
	fmt.Printf("- VolumeFlagVolumeDirty: [%v]\n", VolumeFlags(bsh.VolumeFlags)&VolumeFlagVolumeDirty > 0)
	fmt.Printf("- VolumeFlagsMediaFailure: [%v]\n", VolumeFlags(bsh.VolumeFlags)&VolumeFlagsMediaFailure > 0)
	fmt.Printf("- VolumeFlagClearToZero: [%v]\n", VolumeFlags(bsh.VolumeFlags)&VolumeFlagClearToZero > 0)

	fmt.Printf("\n")
}

// func (bsh BootSectorHeader) UseFirstFat() bool {
// 	return VolumeFlags(bsh.VolumeFlags)&VolumeFlagActiveFat == 0
// }

// func (bsh BootSectorHeader) UseSecondFat() bool {
// 	return VolumeFlags(bsh.VolumeFlags)&VolumeFlagActiveFat > 0

func (bsh BootSectorHeader) String() string {
	return fmt.Sprintf("BootSector<SN=(0x%08x) REVISION=(0x%02x)-(0x%02x)>", bsh.VolumeSerialNumber, bsh.FileSystemRevision[0], bsh.FileSystemRevision[1])
}

func (er *ExfatReader) readBootSectorHead() (bsh BootSectorHeader, sectorSize uint32, err error) {
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

	err = er.parseN(bootSectorHeaderSize, &bsh)
	log.PanicIf(err)

	if bytes.Equal(bsh.JumpBoot[:], requiredJumpBootSignature) != true {
		log.Panicf("jump-boot value not correct: %x", bsh.JumpBoot[:])
	} else if bytes.Equal(bsh.FileSystemName[:], requiredFileSystemName) != true {
		log.Panicf("filesystem name not correct: %x [%s]", bsh.FileSystemName, string(bsh.FileSystemName[:]))
	} else if bsh.BootSignature != requiredBootSignature {
		log.Panicf("boot-signature not correct: %x", bsh.BootSignature)
	}

	for _, c := range bsh.MustBeZero {
		if c != 0 {
			log.Panicf("must-be-zero field not all zeros")
		}
	}

	// Forward through the excess bytes.
	sectorSize = bsh.SectorSize()
	excessByteCount := sectorSize - 512

	if excessByteCount != 0 {
		_, err := er.rs.Seek(int64(excessByteCount), os.SEEK_CUR)
		log.PanicIf(err)
	}

	return bsh, sectorSize, nil
}

type ExtendedBootCode []byte

func (er *ExfatReader) readExtendedBootSector(sectorSize uint32) (extendedBootCode ExtendedBootCode, err error) {
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

	// This field is mandatory and Section 3.2.1 defines its contents.
	//
	// Note: the Main and Backup Boot Sectors both contain the BytesPerSectorShift field.
	//
	// The ExtendedBootCode field shall contain boot-strapping instructions. Implementations may populate this field with the CPU instructions necessary for boot-strapping a computer system. Implementations which don't provide boot-strapping instructions shall initialize each byte in this field to 00h as part of their format operation.

	extendedBootCodeSize := sectorSize - 4
	extendedBootCode = make(ExtendedBootCode, extendedBootCodeSize)

	_, err = io.ReadFull(er.rs, extendedBootCode)
	log.PanicIf(err)

	// This field is mandatory and Section 3.2.2 defines its contents.
	//
	// Note: the Main and Backup Boot Sectors both contain the BytesPerSectorShift field.
	//
	// The ExtendedBootSignature field shall describe whether the intent of given sector is for it to be an Extended Boot Sector or not.
	//
	// The valid value for this field is AA550000h. Any other value in this field invalidates its respective Main or Backup Extended Boot Sector. Implementations should verify the contents of this field prior to depending on any other field in its respective Extended Boot Sector.

	extendedBootSignature := uint32(0)
	err = binary.Read(er.rs, defaultEncoding, &extendedBootSignature)
	log.PanicIf(err)

	if extendedBootSignature != requiredExtendedBootSignature {
		panic(fmt.Errorf("extended boot-signature not correct: %x", extendedBootSignature))
	}

	return extendedBootCode, nil
}

func (er *ExfatReader) readExtendedBootSectors(sectorSize uint32) (extendedBootCodeList [mainExtendedBootSectorCount]ExtendedBootCode, err error) {
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

	for i := 0; i < mainExtendedBootSectorCount; i++ {
		extendedBootCode, err := er.readExtendedBootSector(sectorSize)
		log.PanicIf(err)

		extendedBootCodeList[i] = extendedBootCode
	}

	return extendedBootCodeList, nil
}

type OemParameter struct {
	Parameter [48]byte
}

type OemParameters struct {
	Parameters [10]OemParameter
}

func (er *ExfatReader) readOemParameters(sectorSize uint32) (oemParameters OemParameters, err error) {
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

	err = er.parseN(oemParametersSize, &oemParameters)
	log.PanicIf(err)

	// Rad the remaining unused data of the sector.

	remainder := sectorSize - 480
	buffer := make([]byte, remainder)

	_, err = io.ReadFull(er.rs, buffer)
	log.PanicIf(err)

	return oemParameters, nil
}

func (er *ExfatReader) readMainReserved(sectorSize uint32) (err error) {
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

	// TODO(dustin): !! Add test.

	// This sub-region is mandatory and its contents are reserved.

	buffer := make([]byte, sectorSize)

	_, err = io.ReadFull(er.rs, buffer)
	log.PanicIf(err)

	return nil
}

func (er *ExfatReader) readMainBootChecksum(sectorSize uint32) (err error) {
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

	// TODO(dustin): !! Add test.

	// This sub-region is mandatory and Section 3.4 defines its contents.

	buffer := make([]byte, sectorSize)

	_, err = io.ReadFull(er.rs, buffer)
	log.PanicIf(err)

	// TODO(dustin): Implement the checksum validation.

	return nil
}

func (er *ExfatReader) getCurrentSector() (sector uint32, offset uint32) {

	// TODO(dustin): Add test.

	sectorSize := er.SectorSize()

	currentOffsetRaw, err := er.rs.Seek(0, os.SEEK_CUR)
	log.PanicIf(err)

	currentOffset := uint32(currentOffsetRaw)

	return currentOffset / sectorSize, currentOffset % sectorSize
}

func (er *ExfatReader) printCurrentSector() {

	// TODO(dustin): Add test.

	sectorSize := er.SectorSize()

	currentOffsetRaw, err := er.rs.Seek(0, os.SEEK_CUR)
	log.PanicIf(err)

	currentOffset := uint32(currentOffsetRaw)

	fmt.Printf("CURRENT SECTOR: (%d) (%d)\n", currentOffset/sectorSize, currentOffset%sectorSize)
}

func (er *ExfatReader) assertAlignedToSector() {

	// TODO(dustin): Add test.

	sectorSize := er.SectorSize()

	currentOffsetRaw, err := er.rs.Seek(0, os.SEEK_CUR)
	log.PanicIf(err)

	currentOffset := uint32(currentOffsetRaw)

	if currentOffset%sectorSize != 0 {
		log.Panicf("not currently aligned to a sector: (%d) (%d)", currentOffset/sectorSize, currentOffset%sectorSize)
	}
}

func (er *ExfatReader) parseBootRegion() (br bootRegion, err error) {
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

	// TODO(dustin): !! Add test.

	bsh, sectorSize, err := er.readBootSectorHead()
	log.PanicIf(err)

	// We don't care about these (for now, at least).
	_, err = er.readExtendedBootSectors(sectorSize)
	log.PanicIf(err)

	// We don't care about these (for now, at least).
	_, err = er.readOemParameters(sectorSize)
	log.PanicIf(err)

	err = er.readMainReserved(sectorSize)
	log.PanicIf(err)

	err = er.readMainBootChecksum(sectorSize)
	log.PanicIf(err)

	br = bootRegion{
		bsh:        bsh,
		sectorSize: sectorSize,
	}

	return br, nil
}

func (er *ExfatReader) selectBootRegion(bootRegionMain, bootRegionBackup bootRegion) (err error) {
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

	// TODO(dustin): !! Add test.

	// We currently always elect the main region.
	er.bootRegion = bootRegionMain

	// TODO(dustin): Add validation logic to select the backup region if the main region is no good.

	return nil
}

type MappedCluster uint32

func (mc MappedCluster) IsBad() bool {
	return mc == 0xfffffff7
}

func (mc MappedCluster) IsLast() bool {
	return mc == 0xffffffff
}

type Fat []MappedCluster

func (er *ExfatReader) parseFat() (fat Fat, err error) {
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

	// TODO(dustin): !! Add test

	er.assertAlignedToSector()

	sectorSize := er.SectorSize()

	// This field is mandatory and Section 4.1.1 defines its contents.
	//
	// The FatEntry[0] field shall describe the media type in the first byte (the lowest order byte) and shall contain FFh in the remaining three bytes.
	//
	// The media type (the first byte) should be F8h.

	mediaTypeRaw := uint32(0)
	err = binary.Read(er.rs, defaultEncoding, &mediaTypeRaw)
	log.PanicIf(err)

	mediaType := mediaTypeRaw & 0xff

	if mediaType != 0xf8 {
		log.Panicf("media-type not correct: (0x%08x) -> (0x%02x)", mediaTypeRaw, mediaType)
	}

	// This field is mandatory and Section 4.1.2 defines its contents.
	//
	// The FatEntry[1] field only exists due to historical precedence and does not describe anything of interest.
	//
	// The valid value for this field is FFFFFFFFh. Implementations shall initialize this field to its prescribed value and should not use this field for any purpose. Implementations should not interpret this field and shall preserve its contents across operations which modify surrounding fields.

	value := uint32(0)
	err = binary.Read(er.rs, defaultEncoding, &value)
	log.PanicIf(err)

	if value != 0xffffffff {
		log.Panicf("second fat-entry has unexpected value: (0x%08x)", value)
	}

	totalFatSize := er.bootRegion.bsh.FatLength * sectorSize

	// Includes the two uint32s above.
	actualFatSize := ((er.bootRegion.bsh.ClusterCount + 1) * 4)

	excessSize := totalFatSize - actualFatSize

	// This field is mandatory and Section 4.1.3 defines its contents.
	//
	// ClusterCount + 1 can never exceed FFFFFFF6h.
	//
	// Note: the Main and Backup Boot Sectors both contain the ClusterCount field.
	//
	// Each FatEntry field in this array shall represent a cluster in the Cluster Heap. FatEntry[2] represents the first cluster in the Cluster Heap and FatEntry[ClusterCount+1] represents the last cluster in the Cluster Heap.
	//
	// The valid range of values for these fields shall be:
	//
	// Between 2 and ClusterCount + 1, inclusively, which points to the next FatEntry in the given cluster chain; the given FatEntry shall not point to any FatEntry which precedes it in the given cluster chain
	//
	// Exactly FFFFFFF7h, which marks the given FatEntry's corresponding cluster as "bad"
	//
	// Exactly FFFFFFFFh, which marks the given FatEntry's corresponding cluster as the last cluster of a cluster chain; this is the only valid value for the last FatEntry of any given cluster chain

	entryCount := er.bootRegion.bsh.ClusterCount - 1

	fat = make(Fat, entryCount)
	for i := uint32(0); i < entryCount; i++ {
		err := binary.Read(er.rs, defaultEncoding, &fat[i])
		log.PanicIf(err)
	}

	excess := make([]byte, excessSize)

	_, err = io.ReadFull(er.rs, excess)
	log.PanicIf(err)

	return fat, nil
}

func (er *ExfatReader) parseFats() (fats []Fat, err error) {
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

	sectorSize := er.SectorSize()

	emptyBootRegion := bootRegion{}
	if er.bootRegion == emptyBootRegion {
		log.Panicf("boot-sectors not loaded yet")
	}

	// This sub-region is mandatory and its contents, if any, are undefined.
	//
	// Note: the Main and Backup Boot Sectors both contain the FatOffset field.

	fatAlignment := make([]byte, (uint32(er.bootRegion.bsh.FatOffset)-24)*sectorSize)

	_, err = io.ReadFull(er.rs, fatAlignment)
	log.PanicIf(err)

	// This sub-region is mandatory and Section 4.1 defines its contents.
	//
	// Note: the Main and Backup Boot Sectors both contain the FatOffset and FatLength fields.

	fats = make([]Fat, er.bootRegion.bsh.NumberOfFats)
	for i := 0; i < int(er.bootRegion.bsh.NumberOfFats); i++ {
		fat, err := er.parseFat()
		log.PanicIf(err)

		fats[i] = fat
	}

	return fats, nil
}

func (er *ExfatReader) SectorSize() uint32 {

	// TODO(dustin): !! Add test.

	return uint32(er.bootRegion.sectorSize)
}

func (er *ExfatReader) SectorsPerCluster() uint32 {

	// TODO(dustin): !! Add test.

	return er.bootRegion.bsh.SectorsPerCluster()
}

func (er *ExfatReader) ActiveBootRegion() BootSectorHeader {

	// TODO(dustin): !! Add test.

	return er.bootRegion.bsh
}

func (er *ExfatReader) FirstClusterOfRootDirectory() uint32 {

	// TODO(dustin): !! Add test.

	return er.bootRegion.bsh.FirstClusterOfRootDirectory
}

func (er *ExfatReader) GetCluster(clusterNumber uint32) *ExfatCluster {
	ec, err := newExfatCluster(er, clusterNumber)
	log.PanicIf(err)

	return ec
}

type ClusterVisitorFunc func(ec *ExfatCluster) (doContinue bool, err error)

// EnumerateClusters calls the given callback for each cluster in the chain
// starting from the given cluster.
func (er *ExfatReader) EnumerateClusters(startingClusterNumber uint32, cb ClusterVisitorFunc, useFat bool) (err error) {
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

	if startingClusterNumber < 2 {
		log.Panicf("cluster can not be less than (2): (%d)", startingClusterNumber)
	}

	currentClusterNumber := startingClusterNumber
	for {
		if currentClusterNumber < 2 {
			log.Panicf("cluster-number too low: (%d)", currentClusterNumber)
		}

		ec := er.GetCluster(currentClusterNumber)

		doContinue, err := cb(ec)
		log.PanicIf(err)

		if doContinue == false {
			break
		}

		if useFat == true {
			if currentClusterNumber >= uint32(len(er.activeFat)) {
				log.Panicf("cluster exceeds FAT bounds: (%d) >= (%d)", currentClusterNumber, len(er.activeFat))
			}

			nextMappedCluster := er.activeFat[currentClusterNumber-2]
			if nextMappedCluster.IsLast() == true {
				break
			}

			currentClusterNumber = uint32(nextMappedCluster)
		} else {
			// If not using fat, just move to the next, adjacent cluster.
			//
			// The specification implies that "no fat" means that the data
			// could be allocated in adjacent clusters on disk:
			//
			//  6.3.4.2 NoFatChain Field:
			//
			// 	"...the associated allocation is one contiguous series of
			// 	clusters; the corresponding FAT entries for the clusters are
			// 	invalid and implementations shall not interpret them"
			//
			// However, in practice this is only used when only one cluster is
			// needed. So, this measure is just a theoretical exercise (since we
			// should never even reach the increment if the callback is properly
			// consuming the correct amount of data and stopping when that is
			// reached).

			currentClusterNumber++
		}
	}

	return nil
}

func (er *ExfatReader) checkClusterHeapOffset() (err error) {
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

	// TODO(dustin): !! Add test.

	sectorSize := er.SectorSize()

	alignmentSectors := er.bootRegion.bsh.ClusterHeapOffset - (er.bootRegion.bsh.FatOffset + er.bootRegion.bsh.FatLength*uint32(er.bootRegion.bsh.NumberOfFats))
	alignmentByteCount := alignmentSectors * sectorSize

	alignmentBytes := make([]byte, alignmentByteCount)

	_, err = io.ReadFull(er.rs, alignmentBytes)
	log.PanicIf(err)

	currentOffsetRaw, err := er.rs.Seek(0, os.SEEK_CUR)
	log.PanicIf(err)

	clusterHeapOffset := uint32(currentOffsetRaw)

	currentSectorNumber := clusterHeapOffset / sectorSize
	remainder := clusterHeapOffset % sectorSize

	if uint32(currentSectorNumber) != er.bootRegion.bsh.ClusterHeapOffset || remainder != 0 {
		log.Panicf("calculated cluster offset does not match expected cluster offset: (%d) (%d) != (%d)", currentSectorNumber, remainder, er.bootRegion.bsh.ClusterHeapOffset)
	}

	return nil
}

func (er *ExfatReader) Parse() (err error) {
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

	bootRegionMain, err := er.parseBootRegion()
	log.PanicIf(err)

	bootRegionBackup, err := er.parseBootRegion()
	log.PanicIf(err)

	er.selectBootRegion(bootRegionMain, bootRegionBackup)

	fats, err := er.parseFats()
	log.PanicIf(err)

	// Technically, the spec says that only the active-fat flag in the main
	// boot-sector should be used (not the backup):
	//
	// 	The ActiveFat field of the VolumeFlags field describes which FAT is
	// 	active. Only the VolumeFlags field in the Main Boot Sector is current.
	// 	Implementations shall treat the FAT which is not active as stale. Use
	// 	of the inactive FAT and switching between FATs is implementation
	// 	specific.
	//
	// Obviously, the backup boot sector is there for a reason and, in the event
	// that the main boot-sector is garbage, we want to be consistent with the
	// boot-sector that we're supposed to be using.

	if er.bootRegion.bsh.UseFirstFat() == true {
		er.activeFat = fats[0]
	} else if er.bootRegion.bsh.UseSecondFat() == true {
		er.activeFat = fats[1]
	} else {
		log.Panicf("no fat selected")
	}

	err = er.checkClusterHeapOffset()
	log.PanicIf(err)

	return nil
}

// WriteFromClusterChain enumerates all sectors from all clusters starting
// from the given one.
func (er *ExfatReader) WriteFromClusterChain(firstClusterNumber uint32, dataSize uint64, useFat bool, w io.Writer) (err error) {
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

	// TODO(dustin): !! Add test

	sectorSize := er.SectorSize()
	tailFragmentSize := dataSize % uint64(sectorSize)

	written := uint64(0)
	sectorCount := uint32(0)
	doContinue := true

	clusterCb := func(ec *ExfatCluster) (doContinueInner bool, err error) {
		sectorCb := func(sectorNumber uint32, data []byte) (bool, error) {
			// If we're in the last sector.
			if uint64((sectorCount+1)*sectorSize) > dataSize {
				// If we're in the last sector and the file-size is not an exact
				// multiple of sectors.
				if tailFragmentSize > 0 {
					data = data[:tailFragmentSize]
				}

				doContinue = false
			}

			_, err = w.Write(data)
			log.PanicIf(err)

			written += uint64(len(data))
			sectorCount++

			return doContinue, nil
		}

		err = ec.EnumerateSectors(sectorCb)
		log.PanicIf(err)

		return doContinue, nil
	}

	err = er.EnumerateClusters(firstClusterNumber, clusterCb, useFat)
	log.PanicIf(err)

	if written != dataSize {
		log.Panicf("written bytes do not equal data-size: (%d) != (%d)", written, dataSize)
	}

	return nil
}

// Cluster manages reads on the sectors in a cluster and checks that the
// requested sectors are within bounds.
type ExfatCluster struct {
	er *ExfatReader

	clusterNumber     uint32
	clusterSize       uint32
	sectorsPerCluster uint32
	clusterOffset     uint32
}

func newExfatCluster(er *ExfatReader, clusterNumber uint32) (ec *ExfatCluster, err error) {

	// TODO(dustin): !! Add test.

	if clusterNumber < 2 {
		log.Panicf("cluster-number can not be less than two: (%d)", clusterNumber)
	}

	sectorsPerCluster := er.SectorsPerCluster()
	sectorSize := er.SectorSize()

	clusterSize := sectorsPerCluster * sectorSize
	clusterHeapOffset := er.bootRegion.bsh.ClusterHeapOffset * er.SectorSize()

	// Only clusters numbering (2) and above are stored on disk.
	clusterOffset := clusterHeapOffset + clusterSize*(clusterNumber-2)

	ec = &ExfatCluster{
		er: er,

		clusterNumber:     clusterNumber,
		clusterSize:       clusterSize,
		sectorsPerCluster: sectorsPerCluster,
		clusterOffset:     clusterOffset,
	}

	return ec, nil
}

func (ec *ExfatCluster) ClusterNumber() uint32 {
	return ec.clusterNumber
}

func (ec *ExfatCluster) GetSectorByIndex(sectorIndex uint32) (data []byte, err error) {
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

	// TODO(dustin): !! Add test.

	if sectorIndex >= ec.sectorsPerCluster {
		log.Panicf("sector-index exceeds the number of sectors per cluster: (%d) >= (%d)", sectorIndex, ec.sectorsPerCluster)
	}

	sectorSize := ec.er.SectorSize()

	offset := ec.clusterOffset + sectorSize*sectorIndex

	_, err = ec.er.rs.Seek(int64(offset), os.SEEK_SET)
	log.PanicIf(err)

	data = make([]byte, sectorSize)

	_, err = io.ReadFull(ec.er.rs, data)
	log.PanicIf(err)

	return data, nil
}

type SectorVisitorFunc func(sectorNumber uint32, data []byte) (bool, error)

func (ec *ExfatCluster) EnumerateSectors(cb SectorVisitorFunc) (err error) {
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

	for i := uint32(0); i < ec.sectorsPerCluster; i++ {
		sectorData, err := ec.GetSectorByIndex(i)
		log.PanicIf(err)

		sectorNumber := ec.er.bootRegion.bsh.ClusterHeapOffset + ec.clusterNumber + i

		doContinue, err := cb(sectorNumber, sectorData)
		log.PanicIf(err)

		if doContinue == false {
			break
		}
	}

	return nil
}
