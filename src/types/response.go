package types

// ErrorResponse는 람다 함수의 에러 응답 구조체입니다.
type ErrorResponse struct {
	Message   string `json:"message"`
	JobID     string `json:"jobId,omitempty"`
	ImageURL  string `json:"imageUrl,omitempty"`
	ErrorCode string `json:"errorCode"`
}
