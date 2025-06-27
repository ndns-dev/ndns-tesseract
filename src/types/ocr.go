package types

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

type OCRRequest struct {
	ImageUrl string `json:"imageUrl,omitempty"`
}

type OCRStatus string

const (
	OCRStatusPending      OCRStatus = "PENDING"
	OCRStatusSponsored    OCRStatus = "SPONSORED"
	OCRStatusNonSponsored OCRStatus = "NON-SPONSORED"
)

type OCRResults struct {
	ImageURL  string    `json:"imageUrl" dynamodbav:"imageUrl,omitempty"`
	OCRText   string    `json:"ocrText" dynamodbav:"ocrText,omitempty"`
	Status    OCRStatus `json:"status" dynamodbav:"status,omitempty"`
	Error     string    `json:"error" dynamodbav:"error"`
	Timestamp string    `json:"timestamp" dynamodbav:"timestamp,omitempty"`
}
