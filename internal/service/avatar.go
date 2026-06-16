package service

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/hyoureii/hrbackend/internal/lib"
)

type AvatarHandler struct {
	s3     *s3.Client
	bucket string
}

var allowedExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

func NewAvatarHandler(s3 *s3.Client, bucket string) *AvatarHandler {
	return &AvatarHandler{s3: s3, bucket: bucket}
}

func (h *AvatarHandler) Upload(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	if _, err := h.authenticate(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		http.Error(w, "File too large or invalid form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	ext, ok := allowedExtensions[contentType]
	if !ok {
		http.Error(w, "Unsupported file type", http.StatusUnsupportedMediaType)
		return
	}

	uuid := uuid.New()
	key := "avatars/" + uuid.String() + ext
	if _, err = h.s3.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &h.bucket,
		Key:         &key,
		Body:        file,
		ContentType: &contentType,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Upload failed: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"url":"/api/v1/avatars/%s%s"}`, uuid, ext)
}

func (h *AvatarHandler) Serve(w http.ResponseWriter, r *http.Request, params map[string]string) {
	if _, err := h.authenticate(r); err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	key := "avatars/" + params["filename"]
	result, err := h.s3.GetObject(r.Context(), &s3.GetObjectInput{
		Bucket: &h.bucket,
		Key:    &key,
	})
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	defer result.Body.Close()

	w.Header().Set("Content-Type", *result.ContentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	io.Copy(w, result.Body)
}

func (h *AvatarHandler) authenticate(r *http.Request) (*lib.AuthClaims, error) {
	token := strings.TrimPrefix(r.Header.Get("authorization"), "Bearer ")
	if token == "" {
		return nil, fmt.Errorf("Missing authorization header")
	}
	return lib.ValidateJwt(token)
}
