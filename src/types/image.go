package types

import "time"

// 이미지 크기 제한
const (
	MAX_IMAGE_SIZE      = 12000000 // 1200만 픽셀 (약 4000x3000 크기)
	MAX_IMAGE_DIMENSION = 1200     // 픽셀 단위 (1200x1200 이상인 이미지는 크롭)
	CROP_HEIGHT         = 500
	CROP_WIDTH          = 100
	OPTIMAL_WIDTH       = 1000 // 최적의 이미지 너비
	OPTIMAL_HEIGHT      = 500  // 최적의 이미지 높이
)

// 크롤링 설정
const (
	CRAWL_MAX_RETRIES = 3
	CRAWL_RETRY_DELAY = 500 * time.Millisecond
)
