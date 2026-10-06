package handler_test

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

func encodePNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func encodeJPEG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

// webpHeader is enough for content sniffing; the backend does not decode images.
func webpHeader() []byte {
	return append([]byte("RIFF\x24\x00\x00\x00WEBPVP8 "), make([]byte, 32)...)
}

// pngOfSize is a PNG signature padded with zeros to exactly n bytes.
func pngOfSize(t *testing.T, n int) []byte {
	t.Helper()
	data := make([]byte, n)
	copy(data, encodePNG(t))
	return data
}

type filePart struct {
	field, filename string
	data            []byte
}

// multipartBody builds a multipart/form-data body with the given file parts.
func multipartBody(t *testing.T, parts ...filePart) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for _, p := range parts {
		w, err := mw.CreateFormFile(p.field, p.filename)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := w.Write(p.data); err != nil {
			t.Fatalf("write form file: %v", err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	return &buf, mw.FormDataContentType()
}

func upload(t *testing.T, h http.Handler, entityID string, cookie *http.Cookie, parts ...filePart) *httptest.ResponseRecorder {
	t.Helper()
	body, contentType := multipartBody(t, parts...)
	req := httptest.NewRequest(http.MethodPost, "/api/entities/"+entityID+"/photos", body)
	req.Header.Set("Content-Type", contentType)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func photo(data []byte) filePart {
	return filePart{field: "photo", filename: "upload.bin", data: data}
}

func decodePhoto(t *testing.T, rec *httptest.ResponseRecorder) model.Photo {
	t.Helper()
	return decode[struct{ Data model.Photo }](t, rec).Data
}

// photoApp returns an app, an admin cookie and one entity to attach photos to.
func photoApp(t *testing.T) (testApp, *http.Cookie, string) {
	t.Helper()
	app := newApp(t)
	admin := app.login(t, adminEmail, adminPassword)
	rec := do(t, app.router, http.MethodPost, "/api/entities", validBody, admin)
	expectStatus(t, rec, http.StatusCreated)
	return app, admin, decodeEntity(t, rec).ID
}

func filesOf(t *testing.T, app testApp, entityID string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(app.uploadDir, entityID))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("read upload folder: %v", err)
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return names
}

func TestUploadAndListPhotos(t *testing.T) {
	app, admin, entityID := photoApp(t)

	tests := []struct {
		name, wantType, wantExt string
		data                    []byte
	}{
		{"jpeg", "image/jpeg", ".jpg", encodeJPEG(t)},
		{"png", "image/png", ".png", encodePNG(t)},
		{"webp", "image/webp", ".webp", webpHeader()},
	}
	var uploaded []model.Photo
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The file name is ignored: the type comes from the bytes.
			rec := upload(t, app.router, entityID, admin, filePart{"photo", "misleading-name.gif", tc.data})
			expectStatus(t, rec, http.StatusCreated)
			p := decodePhoto(t, rec)
			if p.ContentType != tc.wantType || p.EntityID != entityID || p.SizeBytes != int64(len(tc.data)) ||
				p.URL != "/api/photos/"+p.ID || rec.Header().Get("Location") != p.URL || p.CreatedAt.IsZero() {
				t.Errorf("unexpected photo %+v (Location %q)", p, rec.Header().Get("Location"))
			}
			if _, err := os.Stat(filepath.Join(app.uploadDir, entityID, p.ID+tc.wantExt)); err != nil {
				t.Errorf("file not stored: %v", err)
			}
			uploaded = append(uploaded, p)
		})
	}

	rec := do(t, app.router, http.MethodGet, "/api/entities/"+entityID+"/photos", "", admin)
	expectStatus(t, rec, http.StatusOK)
	list := decode[struct{ Data []model.Photo }](t, rec).Data
	if len(list) != len(uploaded) {
		t.Fatalf("listed %d photos, want %d", len(list), len(uploaded))
	}
	for i := range list {
		if list[i].ID != uploaded[i].ID || list[i].URL != uploaded[i].URL {
			t.Errorf("photo %d = %+v, want %+v (oldest first)", i, list[i], uploaded[i])
		}
	}
}

func TestListPhotosEmpty(t *testing.T) {
	app, admin, entityID := photoApp(t)
	rec := do(t, app.router, http.MethodGet, "/api/entities/"+entityID+"/photos", "", admin)
	expectStatus(t, rec, http.StatusOK)
	if got := rec.Body.String(); got != "{\"data\":[]}\n" {
		t.Errorf("body = %q, want an empty list (not null)", got)
	}
}

func TestGetPhoto(t *testing.T) {
	app, admin, entityID := photoApp(t)
	data := encodePNG(t)
	p := decodePhoto(t, upload(t, app.router, entityID, admin, photo(data)))

	rec := do(t, app.router, http.MethodGet, p.URL, "", admin)
	expectStatus(t, rec, http.StatusOK)
	if !bytes.Equal(rec.Body.Bytes(), data) {
		t.Error("served bytes differ from the upload")
	}
	for header, want := range map[string]string{
		"Content-Type":           "image/png",
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "private, max-age=86400",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestUploadPhotoValidation(t *testing.T) {
	app, admin, entityID := photoApp(t)
	tests := []struct {
		name   string
		parts  []filePart
		status int
		code   string
		field  string // expected fields.photo message, for 422
	}{
		{"text file named .jpg", []filePart{{"photo", "evil.jpg", []byte("<html><script>alert(1)</script></html>")}},
			http.StatusUnprocessableEntity, "validation_failed", "must be a JPEG, PNG or WebP image"},
		{"gif", []filePart{photo([]byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"))},
			http.StatusUnprocessableEntity, "validation_failed", "must be a JPEG, PNG or WebP image"},
		{"empty file", []filePart{photo(nil)},
			http.StatusUnprocessableEntity, "validation_failed", "is required"},
		{"no parts", nil,
			http.StatusUnprocessableEntity, "validation_failed", "is required"},
		{"5 MB + 1 byte", []filePart{photo(pngOfSize(t, model.MaxPhotoBytes+1))},
			http.StatusUnprocessableEntity, "validation_failed", "must be at most 5 MB"},
		{"wrong field name", []filePart{{"image", "a.png", encodePNG(t)}},
			http.StatusBadRequest, "invalid_request", ""},
		{"two photos", []filePart{photo(encodePNG(t)), photo(encodePNG(t))},
			http.StatusBadRequest, "invalid_request", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := expectError(t, upload(t, app.router, entityID, admin, tc.parts...), tc.status, tc.code)
			if tc.field != "" && got.Fields["photo"] != tc.field {
				t.Errorf("fields = %v, want photo: %q", got.Fields, tc.field)
			}
		})
	}

	if files := filesOf(t, app, entityID); len(files) != 0 {
		t.Errorf("rejected uploads left files behind: %v", files)
	}

	// Exactly 5 MB is allowed.
	expectStatus(t, upload(t, app.router, entityID, admin, photo(pngOfSize(t, model.MaxPhotoBytes))), http.StatusCreated)
}

func TestUploadPhotoNotMultipart(t *testing.T) {
	app, admin, entityID := photoApp(t)
	rec := do(t, app.router, http.MethodPost, "/api/entities/"+entityID+"/photos", `{"photo": "x"}`, admin)
	expectError(t, rec, http.StatusBadRequest, "invalid_request")
}

func TestUploadPhotoLimit(t *testing.T) {
	app, admin, entityID := photoApp(t)
	for i := range model.MaxPhotosPerEntity {
		if rec := upload(t, app.router, entityID, admin, photo(encodePNG(t))); rec.Code != http.StatusCreated {
			t.Fatalf("upload %d: status %d, body %s", i+1, rec.Code, rec.Body)
		}
	}
	got := expectError(t, upload(t, app.router, entityID, admin, photo(encodePNG(t))),
		http.StatusUnprocessableEntity, "validation_failed")
	if got.Fields["photo"] != "this entity already has the maximum of 5 photos" {
		t.Errorf("fields = %v", got.Fields)
	}
	if files := filesOf(t, app, entityID); len(files) != model.MaxPhotosPerEntity {
		t.Errorf("files on disk = %d, want %d (the rejected upload must not stay)", len(files), model.MaxPhotosPerEntity)
	}
}

func TestPhotoNotFound(t *testing.T) {
	app, admin, _ := photoApp(t)

	entityMissing := []*httptest.ResponseRecorder{
		upload(t, app.router, missingID, admin, photo(encodePNG(t))),
		upload(t, app.router, "not-a-uuid", admin, photo(encodePNG(t))),
		do(t, app.router, http.MethodGet, "/api/entities/"+missingID+"/photos", "", admin),
	}
	for i, rec := range entityMissing {
		if got := expectError(t, rec, http.StatusNotFound, "not_found"); got.Message != "entity not found" {
			t.Errorf("request %d: message = %q", i, got.Message)
		}
	}

	for _, id := range []string{missingID, "not-a-uuid"} {
		for _, method := range []string{http.MethodGet, http.MethodDelete} {
			got := expectError(t, do(t, app.router, method, "/api/photos/"+id, "", admin), http.StatusNotFound, "not_found")
			if got.Message != "photo not found" {
				t.Errorf("%s %s: message = %q", method, id, got.Message)
			}
		}
	}
}

func TestDeletePhoto(t *testing.T) {
	app, admin, entityID := photoApp(t)
	keep := decodePhoto(t, upload(t, app.router, entityID, admin, photo(encodePNG(t))))
	gone := decodePhoto(t, upload(t, app.router, entityID, admin, photo(encodeJPEG(t))))

	expectStatus(t, do(t, app.router, http.MethodDelete, gone.URL, "", admin), http.StatusNoContent)
	expectError(t, do(t, app.router, http.MethodGet, gone.URL, "", admin), http.StatusNotFound, "not_found")

	if files := filesOf(t, app, entityID); len(files) != 1 || files[0] != keep.ID+".png" {
		t.Errorf("files after delete = %v, want only %s.png", files, keep.ID)
	}
}

func TestDeleteEntityRemovesPhotos(t *testing.T) {
	app, admin, entityID := photoApp(t)
	p := decodePhoto(t, upload(t, app.router, entityID, admin, photo(encodePNG(t))))

	expectStatus(t, do(t, app.router, http.MethodDelete, "/api/entities/"+entityID, "", admin), http.StatusNoContent)

	if _, err := os.Stat(filepath.Join(app.uploadDir, entityID)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("entity upload folder still exists: %v", err)
	}
	var n int
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM entity_photos`).Scan(&n); err != nil || n != 0 {
		t.Errorf("photo rows after deleting the entity = %d, %v; want 0", n, err)
	}
	expectError(t, do(t, app.router, http.MethodGet, p.URL, "", admin), http.StatusNotFound, "not_found")
}

func TestPhotoAccessControl(t *testing.T) {
	app, admin, entityID := photoApp(t)
	user := register(t, app, "budi@example.com", "rahasia123")
	p := decodePhoto(t, upload(t, app.router, entityID, admin, photo(encodePNG(t))))
	listPath := "/api/entities/" + entityID + "/photos"

	// Anonymous: everything is 401.
	expectError(t, upload(t, app.router, entityID, nil, photo(encodePNG(t))), http.StatusUnauthorized, "unauthorized")
	for _, rt := range []struct{ method, path string }{
		{http.MethodGet, listPath}, {http.MethodGet, p.URL}, {http.MethodDelete, p.URL},
	} {
		expectError(t, do(t, app.router, rt.method, rt.path, ""), http.StatusUnauthorized, "unauthorized")
	}

	// Role user: may view, may not change.
	expectStatus(t, do(t, app.router, http.MethodGet, listPath, "", user), http.StatusOK)
	expectStatus(t, do(t, app.router, http.MethodGet, p.URL, "", user), http.StatusOK)
	expectError(t, upload(t, app.router, entityID, user, photo(encodePNG(t))), http.StatusForbidden, "forbidden")
	expectError(t, do(t, app.router, http.MethodDelete, p.URL, "", user), http.StatusForbidden, "forbidden")

	// The forbidden requests changed nothing.
	if files := filesOf(t, app, entityID); len(files) != 1 {
		t.Errorf("files = %v, want the one admin upload", files)
	}
	body, _ := io.ReadAll(do(t, app.router, http.MethodGet, p.URL, "", admin).Body)
	if !bytes.Equal(body, encodePNG(t)) {
		t.Error("photo changed after forbidden requests")
	}
}
