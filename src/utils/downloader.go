// utils/utils.go (또는 DownloadImage가 있던 파일)

package utils

import (
	"fmt"
	"io"
	"net/http"
)

// FetchImageBytes: URL에서 이미지를 다운로드하여 바이트 배열로 반환
func FetchImageBytes(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d %s", resp.StatusCode, resp.Status)
	}

	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image content from response body: %w", err)
	}

	return imageBytes, nil
}
