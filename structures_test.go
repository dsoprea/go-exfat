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
	filepath := path.Join(AssetPath, "test.exfat")

	f, err := os.Open(filepath)
	log.PanicIf(err)

	er = NewExfatReader(f)
	return f, er
}

func TestExfatReader_readBootSectorHead(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	bsh, err := er.readBootSectorHead()
	log.PanicIf(err)

	if bsh.VolumeSerialNumber != 0x3d51a058 {
		t.Fatalf("Volume serial-number not correct: 0x%x", bsh.VolumeSerialNumber)
	} else if er.sectorSize != 512 {
		t.Fatalf("Sector-size not correct: (%d)", er.sectorSize)
	}
}

func TestExfatReader_readExtendedBootSector(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	_, err := er.readBootSectorHead()
	log.PanicIf(err)

	extendedBootCode, err := er.readExtendedBootSector()
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

	_, err := er.readBootSectorHead()
	log.PanicIf(err)

	extendedBootCodeList, err := er.readExtendedBootSectors()
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

	bsh, err := er.readBootSectorHead()
	log.PanicIf(err)

	bsh.Dump()
}

func TestBootSectorHeader_Parse(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	err := er.Parse()
	log.PanicIf(err)
}

func TestBootSectorHeader_readOemParameters(t *testing.T) {
	f, er := getTestFileAndParser()

	defer f.Close()

	_, err := er.readBootSectorHead()
	log.PanicIf(err)

	_, err = er.readExtendedBootSectors()
	log.PanicIf(err)

	oemParameters, err := er.readOemParameters()
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
