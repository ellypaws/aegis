package types

import "gorm.io/gorm"

// Image is the parent record; Images are stored as child blobs to avoid JSON.
// Order is preserved via Index.
type Image struct {
	gorm.Model

	// Enforce one Image per Post via unique index on PostID
	PostID    uint   `gorm:"uniqueIndex"`
	Thumbnail []byte `gorm:"type:blob"`

	Blobs []ImageBlob `gorm:"constraint:OnDelete:CASCADE"`
}

type ImageBlob struct {
	gorm.Model

	// Enforce unique order for blobs per image
	ImageID uint   `gorm:"index;uniqueIndex:idx_image_blob_order"`
	Index   int    `gorm:"index;uniqueIndex:idx_image_blob_order"` // stable order
	Data    []byte `gorm:"type:blob"`
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

func (im *Image) SetImages(thumb []byte, imgs [][]byte) {
	im.Thumbnail = append([]byte(nil), thumb...)
	im.Blobs = im.Blobs[:0]
	for i, b := range imgs {
		im.Blobs = append(im.Blobs, ImageBlob{
			Index: i,
			Data:  append([]byte(nil), b...),
		})
	}
}
