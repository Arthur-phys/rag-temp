package chunks

import (
	"os"
)

func NewFromMarkdown(filePath string, chunkSize uint32, overlap uint32) ([]string, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return []string{}, err
	}
	defer file.Close()

	data := make([]byte, 10000)
	n, err := file.Read(data)
	if err != nil {
		return []string{}, err
	}

	chunks := []string{}
	var start uint32 = 0
	var end uint32
	for start < uint32(n)-overlap {
		end = start + chunkSize
		if end > uint32(n) {
			end = uint32(n)
		}
		chunks = append(chunks, string(data[start:end]))
		start = end - overlap
	}

	return chunks, nil

}
