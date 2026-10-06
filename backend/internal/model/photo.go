package model

import "time"

const (
	// MaxPhotoBytes is the largest photo that can be uploaded (5 MB).
	MaxPhotoBytes = 5 << 20
	// MaxPhotosPerEntity limits how many photos one entity can have.
	MaxPhotosPerEntity = 5
)

// PhotoExtensions maps every allowed photo content type to the file
// extension it is stored with. The content type is detected from the file's
// bytes, never taken from the client.
var PhotoExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// Photo is an image attached to an entity. The file lives on disk; URL is
// where clients fetch it.
type Photo struct {
	ID          string    `json:"id"`
	EntityID    string    `json:"entity_id"`
	URL         string    `json:"url"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}

// PhotoURL is the API path that serves the photo with the given id.
func PhotoURL(id string) string {
	return "/api/photos/" + id
}
