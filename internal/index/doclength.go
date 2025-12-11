package index

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"luen-search-engine/internal/model"
)

const docLengthFileName = "doclengths.bin"

// LoadDocLengths reads the document length map from disk.
func LoadDocLengths(dir string) (map[model.DocID]model.FieldDocLengths, error) {
	path := filepath.Join(dir, docLengthFileName)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open doclengths file: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	// Read count
	var count uint32
	if err := binary.Read(reader, binary.LittleEndian, &count); err != nil {
		return nil, fmt.Errorf("read count: %w", err)
	}

	docLengths := make(map[model.DocID]model.FieldDocLengths, count)

	// Read entries
	for i := uint32(0); i < count; i++ {
		var docID model.DocID
		if err := binary.Read(reader, binary.LittleEndian, &docID); err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("unexpected EOF at entry %d", i)
			}
			return nil, fmt.Errorf("read docID: %w", err)
		}
		var titleLength uint32
		if err := binary.Read(reader, binary.LittleEndian, &titleLength); err != nil {
			return nil, fmt.Errorf("read title length: %w", err)
		}
		var bodyLength uint32
		if err := binary.Read(reader, binary.LittleEndian, &bodyLength); err != nil {
			return nil, fmt.Errorf("read body length: %w", err)
		}
		docLengths[docID] = model.FieldDocLengths{
			TitleLength: titleLength,
			BodyLength:  bodyLength,
		}
	}

	return docLengths, nil
}

// CalculateAvgDocLength computes the average document length from the map.
func CalculateAvgDocLength(docLengths map[model.DocID]model.FieldDocLengths) float64 {
	if len(docLengths) == 0 {
		return 0
	}
	var total uint64
	for _, length := range docLengths {
		total += uint64(length.TitleLength + length.BodyLength)
	}
	return float64(total) / float64(len(docLengths))
}

// CalculateAvgFieldLengths computes the average title and body lengths separately.
func CalculateAvgFieldLengths(docLengths map[model.DocID]model.FieldDocLengths) (avgTitleLength, avgBodyLength float64) {
	if len(docLengths) == 0 {
		return 0, 0
	}
	var totalTitle, totalBody uint64
	for _, length := range docLengths {
		totalTitle += uint64(length.TitleLength)
		totalBody += uint64(length.BodyLength)
	}
	return float64(totalTitle) / float64(len(docLengths)), float64(totalBody) / float64(len(docLengths))
}
