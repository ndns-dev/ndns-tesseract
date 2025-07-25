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
		return "", fmt.Errorf("failed to fetch image: %v", err)
	}
	log.Printf("Image fetched. Size: %d bytes", len(imageBytes))

	// 2. 임시 파일로 저장
	tempFile, err := os.CreateTemp("", "ocr_image_*.jpg")
	if err != nil {
		log.Printf("ERROR: Failed to create temp file: %v", err)
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// 이미지 바이트를 임시 파일에 저장
	_, err = tempFile.Write(imageBytes)
	if err != nil {
		log.Printf("ERROR: Failed to write image to temp file: %v", err)
		return "", fmt.Errorf("failed to write image to temp file: %v", err)
	}
	tempFile.Close()

	// 3. 이미지 최적화 (크롭)
	log.Printf("Optimizing image for OCR...")
	optimizedImagePath, err := CropImageOptimal(tempFile.Name())
	if err != nil {
		log.Printf("WARNING: Failed to optimize image, using original: %v", err)
		optimizedImagePath = tempFile.Name()
	} else {
		log.Printf("Image optimized successfully: %s", optimizedImagePath)
	}

	// 4. 최적화된 이미지를 바이트로 다시 읽기
	optimizedImageBytes, err := os.ReadFile(optimizedImagePath)
	if err != nil {
		log.Printf("ERROR: Failed to read optimized image: %v", err)
		return "", fmt.Errorf("failed to read optimized image: %v", err)
	}

	// 5. Tesseract OCR 실행
	cmd := exec.Command(tesseractCmdPath, "-", "stdout", "-l", "kor", "--tessdata-dir", tessdataPath,
		"--psm", "6",
		"--oem", "3",
		"-c", "preserve_interword_spaces=1")

	cmd.Stdin = bytes.NewReader(optimizedImageBytes)
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

	// OCR 결과에서 줄바꿈 문자를 공백으로 변환
	cleanedOutput := strings.ReplaceAll(stdout.String(), "\n", " ")
	cleanedOutput = strings.ReplaceAll(cleanedOutput, "\r", " ")

	OcrResult := strings.TrimSpace(cleanedOutput)
	log.Printf("Tesseract stdout (cleaned): %s", OcrResult)

	return OcrResult, nil
}
