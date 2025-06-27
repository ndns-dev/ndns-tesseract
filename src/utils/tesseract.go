package utils

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

const tessdataPath = "/opt/share/tessdata"
const tesseractCmdPath = "/opt/bin/tesseract"

// PerformOCR 함수는 이제 API Gateway 등으로부터 받은 전체 요청 바디를 받습니다.
// 요청 바디 안에는 이미지 URL이 JSON 형태로 포함될 것이라고 가정합니다.
func PerformOCR(imageUrl string) (string, error) {
	// 1. 이미지 바이트를 메모리로 가져오기
	log.Printf("Fetching image bytes from URL: %s", imageUrl)
	imageBytes, err := FetchImageBytes(imageUrl)
	if err != nil {
		log.Printf("ERROR: Failed to fetch image bytes from URL %s: %v", imageUrl, err)
		return "", fmt.Errorf("Failed to fetch image: %v", err)
	}
	log.Printf("Image fetched. Size: %d bytes", len(imageBytes))

	cmd := exec.Command(tesseractCmdPath, "-", "stdout", "-l", "kor", "--tessdata-dir", tessdataPath,
		"--psm", "6",
		"--oem", "3",
		"-c", "preserve_interword_spaces=1")

	cmd.Stdin = bytes.NewReader(imageBytes)
	env := os.Environ()
	cmd.Env = env

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Printf("Executing Tesseract command: %s", cmd.Args)
	err = cmd.Run()

	if err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		errMsg := fmt.Sprintf("Tesseract 실행 실패: %v", err)
		if stderrStr != "" {
			errMsg = fmt.Sprintf("%s - %s", errMsg, stderrStr)
		}
		log.Printf("ERROR: %s", errMsg)
		return "", errors.New(errMsg)
	}

	OcrResult := strings.TrimSpace(stdout.String())
	log.Printf("Tesseract stdout: %s", OcrResult)

	return OcrResult, nil
}
