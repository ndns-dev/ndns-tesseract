package utils

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"

	types "github.com/ndns-dev/ndns-tesseract/src/types"
)

// ImageDimensions는 이미지의 가로/세로 크기 정보를 담고 있습니다
type ImageDimensions struct {
	Width  int
	Height int
}

// GetImageDimensions는 이미지 파일의 가로/세로 크기를 반환합니다
func GetImageDimensions(filePath string) (*ImageDimensions, error) {
	// 파일 열기
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("파일 열기 실패: %v", err)
	}
	defer file.Close()

	// 표준 image 패키지를 사용하여 이미지 디코딩
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("이미지 디코딩 실패: %v", err)
	}

	// 이미지 크기 반환
	bounds := img.Bounds()
	return &ImageDimensions{
		Width:  bounds.Max.X - bounds.Min.X,
		Height: bounds.Max.Y - bounds.Min.Y,
	}, nil
}

// CropImageTop은 큰 이미지를 상단 일부만 잘라서 새 파일로 저장합니다
func CropImageTop(sourcePath string, maxHeight int) (string, error) {
	// 원본 파일 열기
	file, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("원본 파일 열기 실패: %v", err)
	}
	defer file.Close()

	// 이미지 디코딩
	sourceImg, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("이미지 디코딩 실패: %v", err)
	}

	// 원본 이미지 크기 확인
	bounds := sourceImg.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	// 이미지가 이미 maxHeight 이하면 원본 반환
	if height <= maxHeight {
		return sourcePath, nil
	}

	// 잘라낼 영역 계산 (상단 maxHeight 픽셀)
	cropRect := image.Rect(0, 0, width, maxHeight)

	// 새 이미지 생성
	croppedImg := image.NewRGBA(cropRect)

	// 원본 이미지에서 상단 부분만 새 이미지로 복사
	draw.Draw(croppedImg, cropRect, sourceImg, bounds.Min, draw.Src)

	// 새 파일 이름 생성 (원본 파일명_cropped)
	croppedPath := sourcePath + "_cropped.jpg"

	// 새 파일 생성
	outFile, err := os.Create(croppedPath)
	if err != nil {
		return "", fmt.Errorf("출력 파일 생성 실패: %v", err)
	}
	defer outFile.Close()

	// 이미지를 JPEG 형식으로 저장 (간단함을 위해 JPEG만 지원)
	err = jpeg.Encode(outFile, croppedImg, &jpeg.Options{Quality: 90})
	if err != nil {
		return "", fmt.Errorf("이미지 인코딩 실패: %v", err)
	}

	return croppedPath, nil
}

// CropImageCenter는 가로가 긴 이미지의 가운데 부분만 잘라서 새 파일로 저장합니다
func CropImageCenter(sourcePath string, cropWidth int) (string, error) {
	// 원본 파일 열기
	file, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("원본 파일 열기 실패: %v", err)
	}
	defer file.Close()

	// 이미지 디코딩
	sourceImg, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("이미지 디코딩 실패: %v", err)
	}

	// 원본 이미지 크기 확인
	bounds := sourceImg.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	// 가로 길이가 충분히 길지 않으면 원본 반환
	targetWidth := width - (cropWidth * 2)
	if targetWidth <= 0 {
		return sourcePath, nil
	}

	// 잘라낼 영역 계산 (가운데 부분)
	cropRect := image.Rect(cropWidth, 0, width-cropWidth, height)

	// 새 이미지 생성
	croppedImg := image.NewRGBA(image.Rect(0, 0, targetWidth, height))

	// 원본 이미지에서 가운데 부분만 새 이미지로 복사
	draw.Draw(croppedImg, croppedImg.Bounds(), sourceImg, cropRect.Min, draw.Src)

	// 새 파일 이름 생성 (원본 파일명_center_cropped)
	ext := filepath.Ext(sourcePath)
	basePath := sourcePath[:len(sourcePath)-len(ext)]
	croppedPath := basePath + "_center_cropped.jpg"

	// 새 파일 생성
	outFile, err := os.Create(croppedPath)
	if err != nil {
		return "", fmt.Errorf("출력 파일 생성 실패: %v", err)
	}
	defer outFile.Close()

	// 이미지를 JPEG 형식으로 저장
	err = jpeg.Encode(outFile, croppedImg, &jpeg.Options{Quality: 90})
	if err != nil {
		return "", fmt.Errorf("이미지 인코딩 실패: %v", err)
	}

	return croppedPath, nil
}

// CropImageOptimal은 이미지 비율에 따라 최적의 방식으로 크롭합니다.
// 세로가 긴 이미지는 상단 부분만, 가로가 긴 이미지는 가운데 부분만 잘라냅니다.
func CropImageOptimal(sourcePath string) (string, error) {
	// 원본 파일 열기 및 이미지 크기 확인
	dimensions, err := GetImageDimensions(sourcePath)
	if err != nil {
		return "", fmt.Errorf("이미지 크기 확인 실패: %v", err)
	}

	width := dimensions.Width
	height := dimensions.Height

	// 비율에 따라 다른 크롭 방식 적용
	aspectRatio := float64(width) / float64(height)
	isWideTooMuch := width > types.OPTIMAL_WIDTH*1.5   // 너비가 최적값의 1.5배 이상
	isTallTooMuch := height > types.OPTIMAL_HEIGHT*1.5 // 높이가 최적값의 1.5배 이상

	fmt.Printf("이미지 크기: %dx%d, 비율: %.2f\n", width, height, aspectRatio)

	// 이미지가 이미 적정 크기면 원본 반환
	if width <= types.OPTIMAL_WIDTH && height <= types.OPTIMAL_HEIGHT {
		return sourcePath, nil
	}

	var croppedPath string

	// 가로가 매우 긴 경우 (가로 > 세로*2): 가운데 부분 크롭
	if aspectRatio > 2.0 && isWideTooMuch {
		fmt.Printf("가로가 매우 긴 이미지: 가운데 부분 %d픽셀 크롭\n", types.CROP_WIDTH)
		croppedPath, err = CropImageCenter(sourcePath, types.CROP_WIDTH)
		if err != nil {
			return "", err
		}

		// 크롭 후에도 세로가 너무 길면 상단 부분도 크롭
		newDimensions, _ := GetImageDimensions(croppedPath)
		if newDimensions != nil && newDimensions.Height > types.OPTIMAL_HEIGHT*1.5 {
			fmt.Printf("세로도 긴 이미지: 상단 %d픽셀만 사용\n", types.CROP_HEIGHT)
			return CropImageTop(croppedPath, types.CROP_HEIGHT)
		}

		return croppedPath, nil
	} else if aspectRatio < 1.0 && isTallTooMuch {
		// 세로가 매우 긴 경우: 상단 부분 크롭
		fmt.Printf("세로가 긴 이미지: 상단 %d픽셀만 사용\n", types.CROP_HEIGHT)
		return CropImageTop(sourcePath, types.CROP_HEIGHT)
	} else if aspectRatio > 1.0 && aspectRatio < 2.0 && isWideTooMuch {
		// 가로가 약간 긴 경우 (1.0 < 비율 < 2.0): 너비가 너무 넓으면 가운데 크롭
		// 너비를 적절히 줄이기 위한 크롭 범위 계산
		cropAmount := (width - types.OPTIMAL_WIDTH) / 2
		if cropAmount > 0 {
			fmt.Printf("가로가 약간 긴 이미지: 좌우 각각 %d픽셀 제거\n", cropAmount)
			return CropImageCenter(sourcePath, cropAmount)
		}
	}

	// 특별한 경우가 아니면 원본 반환
	return sourcePath, nil
}

// IsGifImage는 파일이 GIF 이미지인지 확인합니다 (파일 시그니처 검사)
func IsGifImage(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// GIF 파일 시그니처 확인 (GIF87a 또는 GIF89a)
	header := make([]byte, 6)
	if _, err := file.Read(header); err != nil {
		return false
	}

	return string(header) == "GIF87a" || string(header) == "GIF89a"
}
