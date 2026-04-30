package model

import "time"

type ContentFile struct {
	ID           int64     `json:"id"`
	URL          string    `json:"url"`
	KeyHash      string    `json:"key_hash"`
	FileType     int16     `json:"file_type"`
	OriginalName string    `json:"original_name"`
	OriginalSize int64     `json:"original_size"`
	CreatedAt    time.Time `json:"created_at"`
}

const (
	ContentFileTypeAudio  int16 = 1
	ContentFileTypeVideo  int16 = 2
	ContentFileTypeImage  int16 = 3
	ContentFileTypeCover  int16 = 4
)
