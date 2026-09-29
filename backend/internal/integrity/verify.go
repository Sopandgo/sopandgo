package integrity

import "os"

func VerifyFile(path string, expectedHash string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	actual := HashBytes(data)
	return actual == expectedHash, nil
}
