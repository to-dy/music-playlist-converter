package services

import "github.com/to-dy/music-playlist-converter/api/services/shared_types"

type SearchTrackList []SearchTrack

type SearchTrack struct {
	Title    string
	Artists  shared_types.Artists
	Duration int64
	Album    shared_types.Album
}
