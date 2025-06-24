// src/ocr/ocr.go

package ocr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/ndns-dev/ndns-tesseract/src/types"
	"github.com/ndns-dev/ndns-tesseract/src/utils"
)

const tessdataPath = "/opt/share/tessdata"
const tesseractCmdPath = "/opt/bin/tesseract"

// PerformOCR 함수는 이제 API Gateway 등으로부터 받은 전체 요청 바디를 받습니다.
// 요청 바디 안에는 이미지 URL이 JSON 형태로 포함될 것이라고 가정합니다.
func PerformOCR(requestBody []byte) (types.OCRJobDetails, error) {
	var req types.OCRRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		log.Printf("ERROR: Invalid request body: %v", err)
		return types.OCRJobDetails{Status: types.JobStatusFailed, Error: "Invalid request body"}, fmt.Errorf("invalid request body: %w", err)
	}

	imageUrl := req.ImageUrl
	if imageUrl == "" {
		log.Print("ERROR: Image URL is missing in the request.")
		return types.OCRJobDetails{Status: types.JobStatusFailed, Error: "Image URL is missing."}, fmt.Errorf("image URL is missing")
	}

	// 1. 이미지 바이트를 메모리로 가져오기
	log.Printf("Fetching image bytes from URL: %s", imageUrl)
	imageBytes, err := utils.FetchImageBytes(imageUrl)
	if err != nil {
		log.Printf("ERROR: Failed to fetch image bytes from URL %s: %v", imageUrl, err)
		return types.OCRJobDetails{Status: types.JobStatusFailed, Error: fmt.Sprintf("Failed to fetch image: %v", err)}, err
	}
	log.Printf("Image fetched. Size: %d bytes", len(imageBytes))

	// 🚨🚨🚨 Tesseract 바이너리 실행 권한 확인 및 디버그 로깅 🚨🚨🚨
	fileInfo, err := os.Stat(tesseractCmdPath)
	if err != nil {
		log.Printf("ERROR: Failed to stat Tesseract binary at %s: %v", tesseractCmdPath, err)
		// 디렉토리 내용 출력
		if dirEntries, err := os.ReadDir("/opt/bin"); err == nil {
			log.Printf("Contents of /opt/bin:")
			for _, entry := range dirEntries {
				info, _ := entry.Info()
				if info != nil {
					log.Printf("- %s (mode: %s)", entry.Name(), info.Mode().String())
				} else {
					log.Printf("- %s", entry.Name())
				}
			}
		}
		return types.OCRJobDetails{Status: types.JobStatusFailed, Error: fmt.Sprintf("Tesseract binary not found at %s", tesseractCmdPath)}, fmt.Errorf("tesseract binary not found")
	}
	log.Printf("Tesseract binary found at %s with mode: %s", tesseractCmdPath, fileInfo.Mode().String())

	cmd := exec.Command(tesseractCmdPath, "-", "stdout", "-l", "kor", "--tessdata-dir", tessdataPath)

	cmd.Stdin = bytes.NewReader(imageBytes)

	// 🚨🚨🚨 Tesseract 바이너리 경로 탐색 및 확인 (새로운 진단 코드) 🚨🚨🚨
	var actualTesseractPath string
	possibleTesseractPaths := []string{
		"/opt/bin/tesseract",       // 가장 일반적인 경로
		"/opt/tesseract",           // 가끔 Layer ZIP의 루트에 tesseract가 있을 수도
		"/usr/local/bin/tesseract", // 일부 빌드에서 사용될 수 있는 경로
		// 필요한 경우 다른 예상 경로 추가
	}

	log.Println("--- Searching for Tesseract binary ---")
	for _, p := range possibleTesseractPaths {
		fileInfo, err := os.Stat(p)
		if err == nil {
			// 파일이 존재하고, 실행 권한이 있는지 확인
			if fileInfo.Mode().IsRegular() && (fileInfo.Mode()&0111 != 0) {
				actualTesseractPath = p
				log.Printf("INFO: Found executable Tesseract binary at: %s", actualTesseractPath)
				break
			} else {
				log.Printf("WARN: Found file at %s but it's not a regular executable. Mode: %s", p, fileInfo.Mode().String())
			}
		} else if os.IsNotExist(err) {
			log.Printf("INFO: Tesseract not found at: %s", p)
		} else {
			log.Printf("ERROR: Error checking path %s: %v", p, err)
		}
	}
	log.Println("--- End Tesseract binary search ---")

	if actualTesseractPath == "" {
		errMsg := "ERROR: No executable Tesseract binary found in expected paths after search."
		log.Print(errMsg)
		return types.OCRJobDetails{Status: types.JobStatusFailed, Error: errMsg}, errors.New(errMsg)
	}
	// 🚨🚨🚨 진단 코드 끝 🚨🚨🚨
	env := os.Environ() // 현재 환경 변수 복사

	// LD_LIBRARY_PATH 추가 또는 업데이트
	foundLibPath := false
	for i, e := range env {
		if strings.HasPrefix(e, "LD_LIBRARY_PATH=") {
			env[i] = "LD_LIBRARY_PATH=/opt/lib:" + strings.TrimPrefix(e, "LD_LIBRARY_PATH=")
			foundLibPath = true
			break
		}
	}
	if !foundLibPath {
		env = append(env, "LD_LIBRARY_PATH=/opt/lib")
	}

	// TESSDATA_PREFIX 추가 또는 업데이트 (tessdataPath는 이미 /opt/share/tessdata)
	foundTessDataPrefix := false
	for i, e := range env {
		if strings.HasPrefix(e, "TESSDATA_PREFIX=") {
			env[i] = "TESSDATA_PREFIX=" + tessdataPath
			foundTessDataPrefix = true
			break
		}
	}
	if !foundTessDataPrefix {
		env = append(env, "TESSDATA_PREFIX="+tessdataPath)
	}

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
		return types.OCRJobDetails{
			Status: types.JobStatusFailed,
			Error:  errMsg,
		}, errors.New(errMsg)
	}

	ocrResult := strings.TrimSpace(stdout.String())
	log.Printf("Tesseract stdout: %s", ocrResult)

	return types.OCRJobDetails{Status: types.JobStatusCompleted, OCRText: ocrResult}, nil
}

func RunOCR(imagePath string) (string, error) {
	cmd := exec.Command("tesseract", imagePath, "stdout", "-l", "kor")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("tesseract 실행 실패: %v\nTesseract stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
