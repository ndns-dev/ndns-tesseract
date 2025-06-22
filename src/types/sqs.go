package types

type SQSMessageBody struct {
	ImageURL string `json:"imageUrl"`
	ImageID  string `json:"imageId,omitempty"`
	JobID    string `json:"jobId"`
}
