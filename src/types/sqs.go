package types

import "time"

// OcrQueueState는 Ocr 처리 상태를 관리합니다
type OcrQueueState struct {
	JobId           string       `json:"jobId" dynamodbav:"jobId"`                                 // 작업 ID
	ReqId           string       `json:"reqId" dynamodbav:"reqId"`                                 // 요청 ID (SSE 매핑용)
	CurrentPosition OcrPosition  `json:"currentPosition" dynamodbav:"currentPosition"`             // 현재 OCR 위치
	Is2025OrLater   bool         `json:"is2025OrLater" dynamodbav:"is2025OrLater"`                 // 2025년 이후 포스트 여부
	CrawlResult     *CrawlResult `json:"crawlResult,omitempty" dynamodbav:"crawlResult,omitempty"` // 크롤링 결과
	RequestedAt     time.Time    `json:"requestedAt" dynamodbav:"requestedAt"`                     // 요청 시간
}

type CrawlResult struct {
	Url              string `json:"url" dynamodbav:"url"`
	FirstParagraph   string `json:"firstParagraph" dynamodbav:"firstParagraph"`
	LastParagraph    string `json:"lastParagraph" dynamodbav:"lastParagraph"`
	Content          string `json:"content" dynamodbav:"content"`
	FirstImageUrl    string `json:"firstImageUrl" dynamodbav:"firstImageUrl"`
	LastImageUrl     string `json:"lastImageUrl" dynamodbav:"lastImageUrl"`
	FirstStickerUrl  string `json:"firstStickerUrl" dynamodbav:"firstStickerUrl"`
	SecondStickerUrl string `json:"secondStickerUrl" dynamodbav:"secondStickerUrl"`
	LastStickerUrl   string `json:"lastStickerUrl" dynamodbav:"lastStickerUrl"`
}

// GetImageUrlByPosition은 OcrPosition에 해당하는 이미지 URL을 반환합니다.
func (c *CrawlResult) GetImageUrlByPosition(position OcrPosition) string {
	switch position {
	case OcrPositionFirstImage:
		return c.FirstImageUrl
	case OcrPositionFirstSticker:
		return c.FirstStickerUrl
	case OcrPositionSecondSticker:
		return c.SecondStickerUrl
	case OcrPositionLastImage:
		return c.LastImageUrl
	case OcrPositionLastSticker:
		return c.LastStickerUrl
	default:
		return ""
	}
}

type TableName string

const (
	OcrResultTableName      TableName = "OcrResult"
	OcrQueueStatusTableName TableName = "OcrQueueStatus"
)
