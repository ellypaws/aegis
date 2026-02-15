package types

import (
	"bytes"
	"io"
	"net/http"

	"gorm.io/gorm"

	"drigo/pkg/utils"
)

// Image is the parent record; Images are stored as child blobs to avoid JSON.
// Order is preserved via Index.
type Image struct {
	gorm.Model

	// Enforce one Image per Post via unique index on PostID
	PostID    uint   `gorm:"uniqueIndex" json:"postId"`
	Thumbnail []byte `gorm:"type:blob" json:"thumbnail"`

	// Computed field
	HasThumbnail *bool `gorm:"->;type:boolean" json:"hasThumbnail"`

	Blobs []ImageBlob `gorm:"constraint:OnDelete:CASCADE" json:"blobs"`
}

type ImageBlob struct {
	gorm.Model

	// Enforce unique order for blobs per image
	ImageID     uint   `gorm:"index;uniqueIndex:idx_image_blob_order" json:"imageId"`
	Index       int    `gorm:"index;uniqueIndex:idx_image_blob_order" json:"index"` // stable order
	Data        []byte `gorm:"type:blob" json:"data"`
	ContentType string `json:"contentType"`
	Size        int64  `gorm:"->;column:size" json:"size"`
	Filename    string `json:"filename"`
}

func (im *Image) GetImages() (thumb []byte, imgs [][]byte) {
	thumb = im.Thumbnail
	if len(im.Blobs) == 0 {
		return thumb, nil
	}
	// GORM does not guarantee order unless specified in query; caller should
	// use Preload("Blobs", func(db *gorm.DB) *gorm.DB { return db.Order("index ASC") })
	imgs = make([][]byte, len(im.Blobs))
	for i := range im.Blobs {
		imgs[i] = append([]byte(nil), im.Blobs[i].Data...)
	}
	return
}

type ImageReader struct {
	Data        []byte
	Reader      io.Reader
	contentType string
}

type ReaderType interface {
	io.Reader
	ContentType() string
}

func (im *Image) Readers() []io.Reader {
	readers := make([]io.Reader, len(im.Blobs))
	for i, blob := range im.Blobs {
		readers[i] = &ImageReader{
			Data:        blob.Data,
			Reader:      bytes.NewReader(blob.Data),
			contentType: blob.ContentType,
		}
	}
	return readers
}

func (im *Image) HasVideo() bool {
	for i := range im.Blobs {
		if im.Blobs[i].IsVideoType() {
			return true
		}
	}
	return false
}

func (ir *ImageReader) Read(p []byte) (n int, err error) {
	return ir.Reader.Read(p)
}

func (ir *ImageReader) ContentType() string {
	if ir.contentType == "" {
		ir.contentType = utils.ContentType(ir.Data)
	}
	return ir.contentType
}

func (im *Image) SetImages(thumb []byte, imgs [][]byte) {
	im.Thumbnail = append([]byte(nil), thumb...)
	im.Blobs = im.Blobs[:0]
	for i, b := range imgs {
		im.Blobs = append(im.Blobs, ImageBlob{
			Index:       i,
			Data:        append([]byte(nil), b...),
			ContentType: http.DetectContentType(b),
		})
	}
}

func (ib *ImageBlob) GetContentType() string {
	if ib.ContentType == "" {
		ib.ContentType = utils.ContentType(ib.Data)
	}
	return ib.ContentType
}

func (ib *ImageBlob) GetFileExtension() string {
	if ib.ContentType == "" {
		ib.ContentType = utils.ContentType(ib.Data)
	}
	return utils.GetFileExtension(ib.ContentType)
}

func (ib *ImageBlob) IsVideoType() bool {
	if ib.ContentType == "" {
		ib.ContentType = utils.ContentType(ib.Data)
	}
	return utils.IsVideoContentType(ib.ContentType)
}
