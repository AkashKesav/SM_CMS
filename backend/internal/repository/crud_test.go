package repository

import "testing"

func TestConvertValueFormatsUUIDBytes(t *testing.T) {
	value := [16]byte{0xf2, 0x00, 0x50, 0xf9, 0xc2, 0x3d, 0x4f, 0x35, 0xa7, 0xf8, 0xe5, 0x27, 0xd8, 0x41, 0xb6, 0x8f}

	got := convertValue(value)
	want := "f20050f9-c23d-4f35-a7f8-e527d841b68f"

	if got != want {
		t.Fatalf("convertValue() = %v, want %v", got, want)
	}
}
