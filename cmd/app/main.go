package main

import (
	"github.com/to-dy/music-playlist-converter/api/router"
	"github.com/to-dy/music-playlist-converter/initializers"
)

func init() {
	initializers.LoadEnv()
}

func main() {
	router.SetupServer()
}
