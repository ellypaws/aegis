package compositor

import (
	"io"
)

var Passthrough passthrough

type passthrough struct{}

func (p passthrough) TileImages(imageBufs []io.Reader) (io.Reader, error) {
	if len(imageBufs) == 0 {
		return nil, nil
	}

	return imageBufs[0], nil
}
