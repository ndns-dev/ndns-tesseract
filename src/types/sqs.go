package types

import "time"

// OcrQueueState는 Ocr 처리 상태를 관리합니다
type OcrQueueState struct {
	JobId           string       `json:"JobId" dynamodbav:"JobId"`                     // 프라이머리 키
	CrawlResult     *CrawlResult `json:"crawlResult" dynamodbav:"crawlResult"`         // 크롤링 결과
	CurrentPosition OcrPosition  `json:"currentPosition" dynamodbav:"currentPosition"` // 현재 Ocr 위치
	Is2025OrLater   bool         `json:"is2025OrLater" dynamodbav:"is2025OrLater"`     // 2025년 이후 여부
	RequestedAt     time.Time    `json:"requestedAt" dynamodbav:"requestedAt"`         // 요청 시간
}
type CrawlResult struct {
	Url              string
	FirstParagraph   string
	LastParagraph    string
	Content          string
	FirstImageUrl    string
	LastImageUrl     string
	FirstStickerUrl  string
	SecondStickerUrl string
	LastStickerUrl   string
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
