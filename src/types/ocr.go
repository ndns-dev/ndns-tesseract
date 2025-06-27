package types

import "time"

type JobStatus string

const (
	JobStatusPending   JobStatus = "PENDING"
	JobStatusCompleted JobStatus = "COMPLETED"
	JobStatusFailed    JobStatus = "FAILED"
)

// OCRJobDetails 타입은 API요청에 대한 응답을 저장하는 데 사용됩니다.
type OCRJobDetails struct {
	JobId     string    `json:"JobId" dynamodbav:"JobId,omitempty"`
	ImageURL  string    `json:"imageUrl" dynamodbav:"imageUrl,omitempty"`
	OCRText   string    `json:"ocrText" dynamodbav:"ocrText,omitempty"`
	Status    JobStatus `json:"status" dynamodbav:"status,omitempty"`
	Error     string    `json:"error" dynamodbav:"error"`
	Timestamp string    `json:"timestamp" dynamodbav:"timestamp,omitempty"`
}

type OcrRequest struct {
	ImageUrl string `json:"imageUrl,omitempty"`
}

type OcrPosition string

const (
	OcrPositionFirstImage    OcrPosition = "FirstImageUrl"
	OcrPositionFirstSticker  OcrPosition = "FirstStickerUrl"
	OcrPositionSecondSticker OcrPosition = "SecondStickerUrl"
	OcrPositionLastImage     OcrPosition = "LastImageUrl"
	OcrPositionLastSticker   OcrPosition = "LastStickerUrl"
)

// OcrResult는 DynamoDB에 저장될 Ocr 결과 아이템을 나타냅니다.
type OcrResult struct {
	ImageUrl    string      `json:"imageUrl" dynamodbav:"imageUrl"`       // 프라이머리 키
	JobId       string      `json:"jobId" dynamodbav:"jobId"`             // State 키
	Position    OcrPosition `json:"position" dynamodbav:"position"`       // Ocr 위치
	OcrText     string      `json:"ocrText" dynamodbav:"ocrText"`         // Ocr 결과 텍스트
	ProcessedAt time.Time   `json:"processedAt" dynamodbav:"processedAt"` // 처리 시간
	Error       string      `json:"error" dynamodbav:"error"`             // 오류 메시지
}
