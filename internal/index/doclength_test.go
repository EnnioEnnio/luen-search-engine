package index

import (
	"testing"

	"luen-search-engine/internal/model"
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
	docLengths := map[model.DocID]model.FieldDocLengths{
		1: {TitleLength: 10, BodyLength: 90},
		2: {TitleLength: 20, BodyLength: 180},
		3: {TitleLength: 30, BodyLength: 270},
	}

	avg := CalculateAvgDocLength(docLengths)
	expected := 200.0

	if avg != expected {
		t.Errorf("CalculateAvgDocLength() = %f, want %f", avg, expected)
	}
}

func TestCalculateAvgDocLengthEmpty(t *testing.T) {
	docLengths := map[model.DocID]model.FieldDocLengths{}

	avg := CalculateAvgDocLength(docLengths)

	if avg != 0 {
		t.Errorf("CalculateAvgDocLength(empty) = %f, want 0", avg)
	}
}

func TestCalculateAvgFieldLengths(t *testing.T) {
	docLengths := map[model.DocID]model.FieldDocLengths{
		1: {TitleLength: 10, BodyLength: 90},
		2: {TitleLength: 20, BodyLength: 180},
		3: {TitleLength: 30, BodyLength: 270},
	}

	avgTitle, avgBody := CalculateAvgFieldLengths(docLengths)
	expectedTitle := 20.0
	expectedBody := 180.0

	if avgTitle != expectedTitle {
		t.Errorf("CalculateAvgFieldLengths() title = %f, want %f", avgTitle, expectedTitle)
	}
	if avgBody != expectedBody {
		t.Errorf("CalculateAvgFieldLengths() body = %f, want %f", avgBody, expectedBody)
	}
}

func TestCalculateAvgFieldLengthsEmpty(t *testing.T) {
	docLengths := map[model.DocID]model.FieldDocLengths{}

	avgTitle, avgBody := CalculateAvgFieldLengths(docLengths)

	if avgTitle != 0 {
		t.Errorf("CalculateAvgFieldLengths(empty) title = %f, want 0", avgTitle)
	}
	if avgBody != 0 {
		t.Errorf("CalculateAvgFieldLengths(empty) body = %f, want 0", avgBody)
	}
}
