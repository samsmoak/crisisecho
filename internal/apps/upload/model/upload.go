package model

type PresignResponse struct {
	URL          string `json:"url"`
	PublicURL    string `json:"publicUrl"`
	FileName     string `json:"fileName"`
	FileType     string `json:"fileType"`
	FilePath     string `json:"filePath"`
	UploadMethod string `json:"uploadMethod"`
	ExpiresIn    int    `json:"expiresIn"`
}

type RetrieveResponse struct {
	URL       string `json:"url"`
	ExpiresIn int    `json:"expiresIn"`
}

type UploadResponse struct {
	PublicURL string `json:"publicUrl"`
	FileName  string `json:"fileName"`
	FileType  string `json:"fileType"`
	Size      int64  `json:"size"`
}
