package server

import (
	"runtime"

	"github.com/charmbracelet/log"
)

type PreloadJob struct {
	ImageID uint
	User    *JwtCustomClaims
}

type PreloadQueue struct {
	jobs   chan PreloadJob
	server *Server
}

func NewPreloadQueue(server *Server) *PreloadQueue {
	// Queue size can be tweaked. 1024 is reasonable buffer.
	q := &PreloadQueue{
		jobs:   make(chan PreloadJob, 1024),
		server: server,
	}
	q.StartWorkers()
	return q
}

func (q *PreloadQueue) StartWorkers() {
	workerCount := runtime.NumCPU() * 4
	log.Info("Starting preload workers", "count", workerCount)
	for i := 0; i < workerCount; i++ {
		go q.worker()
	}
}

func (q *PreloadQueue) worker() {
	for job := range q.jobs {
		// Prioritize check: Check if already cached
		if q.server.isImageExifCached(job.ImageID, job.User) {
			continue
		}

		// Generate and cache
		_, err := q.server.generateAndCacheImageExif(job.ImageID, job.User)
		if err != nil {
			log.Debug("Failed to preload image EXIF", "id", job.ImageID, "error", err)
		} else {
			// log.Debug("Preloaded image EXIF", "id", job.ImageID)
		}
	}
}

func (q *PreloadQueue) Enqueue(imageID uint, user *JwtCustomClaims) {
	if user == nil {
		return
	}

	select {
	case q.jobs <- PreloadJob{ImageID: imageID, User: user}:
		// Queued
	default:
		// Queue full, drop request (standard pattern for shedding load in preloading)
		log.Debug("Preload queue full, dropping job", "id", imageID)
	}
}
