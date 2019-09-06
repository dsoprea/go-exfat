package exfat

import (
	"testing"
)

func TestEntryType_Dump(t *testing.T) {
	EntryType(0xab).Dump()
}

func TestEntryType_String(t *testing.T) {
	s := EntryType(0xab).String()
	if s != "EntryType<TYPE-CODE=(11) IS-CRITICAL=[false] IS-PRIMARY=[true] IS-IN-USE=[true] X-IS-REGULAR=[true] X-IS-UNUSED=[false] X-IS-END=[false]>" {
		t.Fatalf("String not correct: [%s]", s)
	}
}

func TestExfatFileDirectoryEntry_Dump(t *testing.T) {
	fdf := ExfatFileDirectoryEntry{}
	fdf.Dump()
}

func TestExfatStreamExtensionDirectoryEntry_Dump(t *testing.T) {
	sede := ExfatStreamExtensionDirectoryEntry{}
	sede.Dump()
}

func TestDirectoryEntryParserKey_String(t *testing.T) {
	depk := DirectoryEntryParserKey{}
	s := depk.String()
	if s != "DirectoryEntryParserKey<TYPE-CODE=(0) IS-CRITICAL=[false] IS-PRIMARY=[false]>" {
		t.Fatalf("String not correct: [%s]", s)
	}
}

func TestFileAttributes_String(t *testing.T) {
	s := FileAttributes(0x1234).String()
	if s != "FileAttributes<IS-READONLY=[false] IS-HIDDEN=[false] IS-SYSTEM=[true] IS-DIRECTORY=[true] IS-ARCHIVE=[true]>" {
		t.Fatalf("String not correct: [%s]", s)
	}
}

func TestExfatVolumeGuidDirectoryEntry_String(t *testing.T) {
	vgde := ExfatVolumeGuidDirectoryEntry{}
	s := vgde.String()
	if s != "VolumeGuidDirectoryEntry<SECONDARY-COUNT=(0) SET-CHECKSUM=(0x0000) GENERAL-PRIMARY-FLAGS=(0x0000) GUID=[0x0000000000000000...]>" {
		t.Fatalf("String not correct: [%s]", s)
	}
}

func TestExfatVolumeGuidDirectoryEntry_SecondaryCount(t *testing.T) {
	vgde := ExfatVolumeGuidDirectoryEntry{
		SecondaryCountRaw: 99,
	}

	if vgde.SecondaryCount() != 99 {
		t.Fatalf("SecondaryCount not correct.")
	}
}

func TestExfatVolumeGuidDirectoryEntry_TypeName(t *testing.T) {
	vgde := ExfatVolumeGuidDirectoryEntry{}
	if vgde.TypeName() != "VolumeGuid" {
		t.Fatalf("TypeName not correct.")
	}
}

func TestExfatTexFATDirectoryEntry_String(t *testing.T) {
	tfde := ExfatTexFATDirectoryEntry{}
	s := tfde.String()
	if s != "TexFATDirectoryEntry<>" {
		t.Fatalf("String not correct: [%s]", s)
	}
}

func TestExfatTexFATDirectoryEntry_TypeName(t *testing.T) {
	tfde := ExfatTexFATDirectoryEntry{}
	if tfde.TypeName() != "TexFAT" {
		t.Fatalf("TypeName not correct.")
	}
}

func TestExfatVendorExtensionDirectoryEntry_String(t *testing.T) {
	vede := ExfatVendorExtensionDirectoryEntry{}
	s := vede.String()
	if s != "VendorExtensionDirectoryEntry<GENERAL-SECONDARY-FLAGS=(00000000) GUID=(0x00000000000000000000000000000000)>" {
		t.Fatalf("String not correct: [%s]", s)
	}
}

func TestExfatVendorExtensionDirectoryEntry_TypeName(t *testing.T) {
	vede := ExfatVendorExtensionDirectoryEntry{}
	if vede.TypeName() != "VendorExtension" {
		t.Fatalf("TypeName not correct.")
	}
}

func TestExfatVendorAllocationDirectoryEntry_String(t *testing.T) {
	vade := ExfatVendorAllocationDirectoryEntry{}
	s := vade.String()
	if s != "VendorAllocationDirectoryEntry<GENERAL-SECONDARY-FLAGS=(00000000) GUID=(0x00000000000000000000000000000000) VENDOR-DEFINED=(0x00000000) FIRST-CLUSTER=(0) DATA-LENGTH=(0)>" {
		t.Fatalf("String not correct: [%s]", s)
	}
}

func TestExfatVendorAllocationDirectoryEntry_TypeName(t *testing.T) {
	vade := ExfatVendorAllocationDirectoryEntry{}
	if vade.TypeName() != "VendorAllocation" {
		t.Fatalf("TypeName not correct.")
	}
}
