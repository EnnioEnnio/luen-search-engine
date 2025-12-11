package index

import (
	"testing"
)

func TestLoadDocLengths(t *testing.T) {
	// This test requires the full indexer pipeline
	// For now, we just test that LoadDocLengths returns an error on missing file
	tmpDir := t.TempDir()

	_, err := LoadDocLengths(tmpDir)
	if err == nil {
		t.Error("expected error when loading from empty directory")
	}
}

func TestCalculateAvgDocLength(t *testing.T) {
	docLengths := map[DocID]DocLength{
		1: 100,
		2: 200,
		3: 300,
	}

	avg := CalculateAvgDocLength(docLengths)
	expected := 200.0

	if avg != expected {
		t.Errorf("CalculateAvgDocLength() = %f, want %f", avg, expected)
	}
}

func TestCalculateAvgDocLengthEmpty(t *testing.T) {
	docLengths := make(map[DocID]DocLength)
	avg := CalculateAvgDocLength(docLengths)

	if avg != 0 {
		t.Errorf("CalculateAvgDocLength(empty) = %f, want 0", avg)
	}
}
