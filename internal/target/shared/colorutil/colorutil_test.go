package colorutil

import "testing"

func TestNormalizeHexLowerAndUpper(t *testing.T) {
	t.Parallel()

	gotLower, okLower := NormalizeHexLower(" #AaBbCc ")
	if !okLower || gotLower != "#aabbcc" {
		t.Fatalf("NormalizeHexLower mismatch: got=%q ok=%v", gotLower, okLower)
	}

	gotUpper, okUpper := NormalizeHexUpper("#aabbcc")
	if !okUpper || gotUpper != "#AABBCC" {
		t.Fatalf("NormalizeHexUpper mismatch: got=%q ok=%v", gotUpper, okUpper)
	}
}

func TestHexToRGBCSV(t *testing.T) {
	t.Parallel()

	compact, ok := HexToRGBCSV("#102030", false)
	if !ok || compact != "16,32,48" {
		t.Fatalf("HexToRGBCSV compact mismatch: got=%q ok=%v", compact, ok)
	}

	spaced, ok := HexToRGBCSV("#102030", true)
	if !ok || spaced != "16, 32, 48" {
		t.Fatalf("HexToRGBCSV spaced mismatch: got=%q ok=%v", spaced, ok)
	}
}
