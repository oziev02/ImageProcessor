package http

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/oziev02/ImageProcessor/internal/domain"
	"github.com/oziev02/ImageProcessor/internal/service"
)

type Handler struct {
	imageService service.ImageService
	storageRepo  StorageReader
}

type StorageReader interface {
	Read(ctx context.Context, path string) (io.ReadCloser, error)
}

func NewHandler(imageService service.ImageService, storageRepo StorageReader) *Handler {
	return &Handler{
		imageService: imageService,
		storageRepo:  storageRepo,
	}
}

//go:embed web
var webFiles embed.FS

func (h *Handler) RegisterRoutes(r chi.Router) {
	// Serve static files
	staticFS, err := fs.Sub(webFiles, "web/static")
	if err == nil {
		r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	// Serve index.html
	r.Get("/", h.Index)

	// API routes
	r.Post("/upload", h.Upload)
	r.Get("/image/{id}", h.GetImage)
	r.Get("/api/image/{id}", h.GetImageInfo)
	r.Get("/api/images", h.ListImages)
	r.Delete("/image/{id}", h.DeleteImage)
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "failed to get file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, err := h.imageService.Upload(r.Context(), file, header)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to upload image: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(img)
}

func (h *Handler) GetImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "image id is required", http.StatusBadRequest)
		return
	}

	img, err := h.imageService.GetByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrImageNotFound {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to get image: %v", err), http.StatusInternalServerError)
		return
	}

	// Determine which image to serve
	imagePath := img.ProcessedPath
	if imagePath == "" {
		imagePath = img.OriginalPath
	}

	reader, err := h.storageRepo.Read(r.Context(), imagePath)
	if err != nil {
		http.Error(w, "failed to read image file", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	// Set content type
	contentType := "image/jpeg"
	switch img.Format {
	case domain.FormatPNG:
		contentType = "image/png"
	case domain.FormatGIF:
		contentType = "image/gif"
	}

	w.Header().Set("Content-Type", contentType)
	io.Copy(w, reader)
}

func (h *Handler) GetImageInfo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "image id is required", http.StatusBadRequest)
		return
	}

	img, err := h.imageService.GetByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrImageNotFound {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to get image: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(img)
}

func (h *Handler) ListImages(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	images, err := h.imageService.List(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list images: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(images)
}

func (h *Handler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "image id is required", http.StatusBadRequest)
		return
	}

	if err := h.imageService.Delete(r.Context(), id); err != nil {
		if err == domain.ErrImageNotFound {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to delete image: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	indexFile, err := webFiles.Open("web/index.html")
	if err != nil {
		http.Error(w, "failed to load index.html", http.StatusInternalServerError)
		return
	}
	defer indexFile.Close()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.Copy(w, indexFile)
}
