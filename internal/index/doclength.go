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
func LoadDocLengths(dir string) (map[model.DocID]model.DocLength, error) {
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

	docLengths := make(map[model.DocID]model.DocLength, count)

	// Read entries
	for i := uint32(0); i < count; i++ {
		var docID model.DocID
		if err := binary.Read(reader, binary.LittleEndian, &docID); err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("unexpected EOF at entry %d", i)
			}
			return nil, fmt.Errorf("read docID: %w", err)
		}
		var length uint32
		if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
			return nil, fmt.Errorf("read length: %w", err)
		}
		docLengths[docID] = model.DocLength(length)
	}

	return docLengths, nil
}

// CalculateAvgDocLength computes the average document length from the map.
func CalculateAvgDocLength(docLengths map[model.DocID]model.DocLength) float64 {
	if len(docLengths) == 0 {
		return 0
	}
	var total uint64
	for _, length := range docLengths {
		total += uint64(length)
	}
	return float64(total) / float64(len(docLengths))
}
