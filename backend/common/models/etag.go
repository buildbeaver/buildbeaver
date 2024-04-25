package models

const ETagAny = "*"

type ETag string

func (e ETag) String() string {
	return string(e)
}

func GetETag(resource MutableResource, etag ETag) ETag {
	if etag != "" {
		return etag
	}
	return resource.GetETag()
}
