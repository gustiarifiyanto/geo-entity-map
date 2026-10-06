package service

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

// PhotoRepository is the photo metadata storage the photo service depends on.
type PhotoRepository interface {
	CreatePhoto(ctx context.Context, p model.Photo, limit int) error
	ListPhotos(ctx context.Context, entityID string) ([]model.Photo, error)
	GetPhoto(ctx context.Context, id string) (model.Photo, error)
	DeletePhoto(ctx context.Context, id string) (model.Photo, error)
}

// PhotoFiles stores the photo files themselves.
type PhotoFiles interface {
	Save(entityID, name string, data []byte) error
	Open(entityID, name string) (*os.File, error)
	Remove(entityID, name string) error
}

// EntityLookup finds an entity, or returns model.ErrNotFound.
type EntityLookup interface {
	Get(ctx context.Context, id string) (model.Entity, error)
}

// PhotoService implements uploading, listing, serving and deleting photos.
type PhotoService struct {
	photos   PhotoRepository
	files    PhotoFiles
	entities EntityLookup
	now      func() time.Time
}

// NewPhotoService returns a photo service.
func NewPhotoService(photos PhotoRepository, files PhotoFiles, entities EntityLookup) *PhotoService {
	return &PhotoService{
		photos:   photos,
		files:    files,
		entities: entities,
		now:      func() time.Time { return time.Now().UTC().Truncate(time.Second) },
	}
}

// Upload stores data as a new photo of an entity. It returns model.ErrNotFound
// for an unknown entity, model.ErrUnsupportedPhoto if data is not a JPEG, PNG
// or WebP image, and model.ErrTooManyPhotos when the entity is full. The size
// limit is enforced by the caller while reading the upload.
func (s *PhotoService) Upload(ctx context.Context, entityID string, data []byte) (model.Photo, error) {
	if _, err := s.entities.Get(ctx, entityID); err != nil {
		return model.Photo{}, err
	}
	// The type comes from the bytes, never from the file name or the client's header.
	contentType := http.DetectContentType(data)
	ext, ok := model.PhotoExtensions[contentType]
	if !ok {
		return model.Photo{}, model.ErrUnsupportedPhoto
	}

	p := model.Photo{
		ID:          uuid.NewString(),
		EntityID:    entityID,
		ContentType: contentType,
		SizeBytes:   int64(len(data)),
		CreatedAt:   s.now(),
	}
	if err := s.files.Save(entityID, p.ID+ext, data); err != nil {
		return model.Photo{}, err
	}
	if err := s.photos.CreatePhoto(ctx, p, model.MaxPhotosPerEntity); err != nil {
		// Do not leave a file without a database row.
		_ = s.files.Remove(entityID, p.ID+ext)
		return model.Photo{}, err
	}
	return withURL(p), nil
}

// List returns an entity's photos, oldest first, or model.ErrNotFound.
func (s *PhotoService) List(ctx context.Context, entityID string) ([]model.Photo, error) {
	if _, err := s.entities.Get(ctx, entityID); err != nil {
		return nil, err
	}
	photos, err := s.photos.ListPhotos(ctx, entityID)
	if err != nil {
		return nil, err
	}
	for i := range photos {
		photos[i] = withURL(photos[i])
	}
	return photos, nil
}

// Open returns a photo and its file, or model.ErrPhotoNotFound. The caller
// must close the file.
func (s *PhotoService) Open(ctx context.Context, id string) (model.Photo, *os.File, error) {
	if !validID(id) {
		return model.Photo{}, nil, model.ErrPhotoNotFound
	}
	p, err := s.photos.GetPhoto(ctx, id)
	if err != nil {
		return model.Photo{}, nil, err
	}
	f, err := s.files.Open(p.EntityID, p.ID+model.PhotoExtensions[p.ContentType])
	if errors.Is(err, fs.ErrNotExist) {
		return model.Photo{}, nil, model.ErrPhotoNotFound
	}
	if err != nil {
		return model.Photo{}, nil, err
	}
	return withURL(p), f, nil
}

// Delete removes a photo and its file, or returns model.ErrPhotoNotFound.
func (s *PhotoService) Delete(ctx context.Context, id string) error {
	if !validID(id) {
		return model.ErrPhotoNotFound
	}
	p, err := s.photos.DeletePhoto(ctx, id)
	if err != nil {
		return err
	}
	return s.files.Remove(p.EntityID, p.ID+model.PhotoExtensions[p.ContentType])
}

func withURL(p model.Photo) model.Photo {
	p.URL = model.PhotoURL(p.ID)
	return p
}
