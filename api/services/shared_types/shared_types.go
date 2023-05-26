package shared_types

type Artist struct {
	Name string `json:"name"`
}
type Album struct {
	AlbumType string
	Name      string `json:"name"`
}

type Artists []Artist
