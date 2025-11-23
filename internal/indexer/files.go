package indexer

import "path/filepath"

// DictionaryPath returns the absolute path to dictionary.tsv in the index directory.
func DictionaryPath(dir string) string {
	return filepath.Join(dir, dictionaryFileName)
}

// PostingsPath returns the absolute path to postings.bin in the index directory.
func PostingsPath(dir string) string {
	return filepath.Join(dir, postingsFileName)
}

// ManifestPath returns the absolute path to manifest.json in the index directory.
func ManifestPath(dir string) string {
	return filepath.Join(dir, manifestFileName)
}
