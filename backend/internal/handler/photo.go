package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/service"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/validation"
)

// photoFormField is the multipart field that carries the uploaded file.
const photoFormField = "photo"

var (
	photoTooLarge = fmt.Sprintf("must be at most %d MB", model.MaxPhotoBytes>>20)
	// Room for multipart headers and boundaries around a maximum-size photo.
	maxUploadBodyBytes int64 = model.MaxPhotoBytes + 1<<20
)

type photoHandler struct {
	svc *service.PhotoService
}

func (h *photoHandler) list(w http.ResponseWriter, r *http.Request) {
	photos, err := h.svc.List(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, photos)
}

func (h *photoHandler) upload(w http.ResponseWriter, r *http.Request) {
	data, ok := readPhotoUpload(w, r)
	if !ok {
		return
	}

	p, err := h.svc.Upload(r.Context(), chi.URLParam(r, "id"), data)
	switch {
	case errors.Is(err, model.ErrUnsupportedPhoto):
		writeValidation(w, validation.FieldErrors{photoFormField: "must be a JPEG, PNG or WebP image"})
	case errors.Is(err, model.ErrTooManyPhotos):
		writeValidation(w, validation.FieldErrors{photoFormField: fmt.Sprintf(
			"this entity already has the maximum of %d photos", model.MaxPhotosPerEntity)})
	case err != nil:
		writeServiceError(w, r, err)
	default:
		w.Header().Set("Location", p.URL)
		writeData(w, http.StatusCreated, p)
	}
}

// get serves the image itself. Range requests and If-Modified-Since are
// handled by http.ServeContent.
func (h *photoHandler) get(w http.ResponseWriter, r *http.Request) {
	p, f, err := h.svc.Open(r.Context(), chi.URLParam(r, "photoID"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", p.ContentType)
	// Never let a browser guess another type (e.g. HTML) from the bytes.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Photos never change, they are only deleted; private because they need a login.
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(w, r, "", p.CreatedAt, f)
}

func (h *photoHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "photoID")); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// readPhotoUpload reads the single "photo" file from a multipart body. On
// failure it writes the error response and returns false.
func readPhotoUpload(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBodyBytes)
	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request",
			fmt.Sprintf("request body must be multipart/form-data with a %q file", photoFormField))
		return nil, false
	}

	var (
		data  []byte
		found bool
	)
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			writeUploadReadError(w, err)
			return nil, false
		}
		if part.FormName() != photoFormField {
			writeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("unknown field %q", part.FormName()))
			return nil, false
		}
		if found {
			writeError(w, http.StatusBadRequest, "invalid_request", "only one photo can be uploaded per request")
			return nil, false
		}
		found = true

		// Read one byte past the limit to tell "exactly 5 MB" from "too big".
		data, err = io.ReadAll(io.LimitReader(part, model.MaxPhotoBytes+1))
		if err != nil {
			writeUploadReadError(w, err)
			return nil, false
		}
		if len(data) > model.MaxPhotoBytes {
			writeValidation(w, validation.FieldErrors{photoFormField: photoTooLarge})
			return nil, false
		}
	}

	if len(data) == 0 {
		writeValidation(w, validation.FieldErrors{photoFormField: "is required"})
		return nil, false
	}
	return data, true
}

func writeUploadReadError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeValidation(w, validation.FieldErrors{photoFormField: photoTooLarge})
		return
	}
	writeError(w, http.StatusBadRequest, "invalid_request", "malformed multipart body")
}
