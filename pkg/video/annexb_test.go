package video

import (
	"bytes"
	"testing"
)

func TestSplitAnnexB(t *testing.T) {
	stream := []byte{
		0, 0, 0, 1, 0x67, 0xaa, 0xbb, // SPS, four-byte start code
		0, 0, 1, 0x68, 0xcc, // PPS, three-byte start code
		0, 0, 0, 1, 0x65, 0x00, 0x01, 0xdd, // IDR slice with an embedded zero
	}
	got := splitAnnexB(stream)
	want := [][]byte{{0x67, 0xaa, 0xbb}, {0x68, 0xcc}, {0x65, 0x00, 0x01, 0xdd}}
	if len(got) != len(want) {
		t.Fatalf("got %d NAL units, want %d", len(got), len(want))
	}
	for i := range want {
		if !bytes.Equal(got[i], want[i]) {
			t.Fatalf("NAL %d = % x, want % x", i, got[i], want[i])
		}
	}
	if nalType(got[0]) != nalTypeSPS || nalType(got[1]) != nalTypePPS || nalType(got[2]) != nalTypeIDR {
		t.Fatal("unexpected NAL unit types")
	}
}

func TestSplitAnnexBWithoutStartCode(t *testing.T) {
	if got := splitAnnexB([]byte{0x65, 0x01}); len(got) != 0 {
		t.Fatalf("expected no NAL units, got %d", len(got))
	}
}
