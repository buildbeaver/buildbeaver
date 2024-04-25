package models

// BlobDescriptor describes a blob in the blob store.
type BlobDescriptor struct {
	Key       string
	SizeBytes int64
}
