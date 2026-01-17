package compression

import (
	"bytes"
	"testing"
)

func TestCompression(t *testing.T) {
	config := DefaultConfig()
	config.MinSize = 0 // Compress everything for testing

	testData := []byte("this is some test data that should be compressed and decompressed correctly. it needs to be long enough to actually see some compression if possible, but here we just care about correctness.")

	algorithms := []Algorithm{
		AlgorithmGzip,
		AlgorithmLZ4,
		AlgorithmSnappy,
		AlgorithmZstd,
	}

	for _, algo := range algorithms {
		t.Run(algo.String(), func(t *testing.T) {
			config.Algorithm = algo
			compressor := NewCompressor(config)

			compressed, err := compressor.Compress(testData)
			if err != nil {
				t.Fatalf("failed to compress with %s: %v", algo, err)
			}

			// For some small data or specific algos, it might not actually be smaller, that's fine for this test

			decompressed, err := compressor.Decompress(compressed, algo)
			if err != nil {
				t.Fatalf("failed to decompress with %s: %v", algo, err)
			}

			if !bytes.Equal(testData, decompressed) {
				t.Errorf("decompressed data does not match original for %s", algo)
			}
		})
	}
}

func TestBatchCompression(t *testing.T) {
	config := DefaultConfig()
	config.MinSize = 0
	config.Algorithm = AlgorithmZstd

	batchCompressor := NewBatchCompressor(config)

	entries := [][]byte{
		[]byte("entry 1"),
		[]byte("entry 2"),
		[]byte("entry 3 - a bit longer than others"),
	}

	for _, entry := range entries {
		batchCompressor.Add(entry)
	}

	compressed, err := batchCompressor.Flush()
	if err != nil {
		t.Fatalf("failed to flush batch: %v", err)
	}

	decompressedEntries, err := batchCompressor.DecompressBatch(compressed, config.Algorithm)
	if err != nil {
		t.Fatalf("failed to decompress batch: %v", err)
	}

	if len(decompressedEntries) != len(entries) {
		t.Fatalf("expected %d entries, got %d", len(entries), len(decompressedEntries))
	}

	for i, entry := range entries {
		if !bytes.Equal(entry, decompressedEntries[i]) {
			t.Errorf("entry %d does not match", i)
		}
	}
}
