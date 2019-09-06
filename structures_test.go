package exfat

import (
	"bytes"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/dsoprea/go-logging"
)

func getTestFileAndParser() (f *os.File, er *ExfatReader) {
	filepath := path.Join(assetPath, "test.exfat")

	f, err := os.Open(filepath)
	log.PanicIf(err)

	er = NewExfatReader(f)
	return f, er
}

func TestExfatReader_readBootSectorHead(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	bsh, sectorSize, err := er.readBootSectorHead()
	log.PanicIf(err)

	if bsh.VolumeSerialNumber != 0x3d51a058 {
		t.Fatalf("Volume serial-number not correct: 0x%x", bsh.VolumeSerialNumber)
	} else if sectorSize != 512 {
		t.Fatalf("Sector-size not correct: (%d)", sectorSize)
	} else if bsh.ClusterCount != 239 {
		t.Fatalf("ClusterCount not correct: (%d)", bsh.ClusterCount)
	} else if bsh.NumberOfFats != 1 {
		t.Fatalf("NumberOfFats not correct: (%d)", bsh.NumberOfFats)
	}
}

func TestExfatReader_readExtendedBootSector(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	_, sectorSize, err := er.readBootSectorHead()
	log.PanicIf(err)

	extendedBootCode, err := er.readExtendedBootSector(sectorSize)
	log.PanicIf(err)

	nullExtendedBootCode := make(ExtendedBootCode, 508)
	if bytes.Equal(extendedBootCode, nullExtendedBootCode) != true {
		t.Fatalf("Extended boot-code not correct.")
	}
}

func TestExfatReader_readExtendedBootSectors(t *testing.T) {
	defer func() {
		if errRaw := recover(); errRaw != nil {
			err := errRaw.(error)

			log.PrintError(err)
			t.Fatalf("Test failed.")
		}
	}()

	f, er := getTestFileAndParser()

	defer f.Close()

	_, sectorSize, err := er.readBootSectorHead()
	log.PanicIf(err)

	extendedBootCodeList, err := er.readExtendedBootSectors(sectorSize)
	log.PanicIf(err)

	var expectedExtendedBootCodeList [mainExtendedBootSectorCount]ExtendedBootCode

	for i := 0; i < mainExtendedBootSectorCount; i++ {
		nullExtendedBootCode := make(ExtendedBootCode, 508)
		expectedExtendedBootCodeList[i] = nullExtendedBootCode
	}

	if reflect.DeepEqual(extendedBootCodeList, expectedExtendedBootCodeList) != true {
		t.Fatalf("readExtendedBootSectors did not return correct data.")
	}
}

func TestBootSectorHeader_Dump(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	bsh, _, err := er.readBootSectorHead()
	log.PanicIf(err)

	bsh.Dump()
}

func TestExfatReader_readOemParameters(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	_, sectorSize, err := er.readBootSectorHead()
	log.PanicIf(err)

	_, err = er.readExtendedBootSectors(sectorSize)
	log.PanicIf(err)

	oemParameters, err := er.readOemParameters(sectorSize)
	log.PanicIf(err)

	if len(oemParameters.Parameters) != 10 {
		t.Fatalf("Expected 10 OEM-parameter members: (%d)", len(oemParameters.Parameters))
	}

	for i, oemParameter := range oemParameters.Parameters {
		if len(oemParameter.Parameter) != 48 {
			t.Fatalf("OEM-parameter (%d) not correct size: (%d)", i, len(oemParameter.Parameter))
		}

		for j, c := range oemParameter.Parameter {
			if c != 0 {
				t.Fatalf("OEM-parameter not full of NULs as expected: (%d) (%d)", i, j)
			}
		}
	}
}

func TestExfatReader_parseBootRegion(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	br, err := er.parseBootRegion()
	log.PanicIf(err)

	if br.bsh.SectorSize() != 512 {
		t.Fatalf("Sector-size not correct: (%d)", br.bsh.SectorSize())
	}

	description := br.bsh.String()
	if description != "BootSector<SN=(0x3d51a058) REVISION=(0x00)-(0x01)>" {
		t.Fatalf("Boot-sector description not correct: %s", description)
	}
}

func TestExfatReader_parseFats(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	bootRegionMain, err := er.parseBootRegion()
	log.PanicIf(err)

	_, err = er.parseBootRegion()
	log.PanicIf(err)

	er.bootRegion = bootRegionMain

	_, err = er.parseFats()
	log.PanicIf(err)

	// TODO(dustin): Add additional validation on FAT structures.
}

func TestExfatReader_parseFats__NotLoaded(t *testing.T) {
	defer func() {
		errRaw := recover()
		if errRaw == nil {
			t.Fatalf("Expected error when BSH not yet loaded.")
		}

		err := errRaw.(error)
		if err.Error() != "boot-sectors not loaded yet" {
			t.Fatalf("Expected not-loaded error.")
		}
	}()

	f, er := getTestFileAndParser()

	defer f.Close()

	_, err := er.parseFats()
	log.PanicIf(err)
}

func TestExfatReader_Parse(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)
}

func TestExfatReader_getCurrentSector(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	sector, offset := er.getCurrentSector()
	if sector == 0 || offset != 0 {
		t.Fatalf("Current sector not correct: (%d)", sector)
	}
}

func TestExfatReader_printCurrentSector(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	er.printCurrentSector()
}

func TestExfatReader_assertAlignedToSector__ok(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	er.assertAlignedToSector()
}

func TestExfatReader_assertAlignedToSector__fail(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Fatalf("Expected failure when misaligned.")
		}
	}()

	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	_, err = f.Seek(1, os.SEEK_CUR)
	log.PanicIf(err)

	er.assertAlignedToSector()
}

func TestExfatReader_ActiveBootSectorHeader(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)

	if er.ActiveBootSectorHeader() != er.bootRegion.bsh {
		t.Fatalf("ActiveBootSectorHeader not correct.")
	}
}

func TestMappedCluster_IsBad__true(t *testing.T) {
	if MappedCluster(0).IsBad() != false {
		t.Fatalf("Expected MC to not be bad.")
	}
}

func TestMappedCluster_IsBad__false(t *testing.T) {
	if MappedCluster(0xfffffff7).IsBad() != true {
		t.Fatalf("Expected MC to be bad.")
	}
}
