package api

// ImageStateSummary provides summarized information about the state of an image.
type ImageStateSummary struct {
	// Size of the image
	Size uint64
	// Used to track how many nodes have this image
	NumNodes uint32
}
