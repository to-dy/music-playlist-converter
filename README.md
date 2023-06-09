# Music playlist converter, supports Spotify, YouTubeMusic & Apple Music

## Overview

---

Built primarily to learn Go and solve a personal need. This app converts music playlists between platforms. It currently supports conversions between Spotify & YouTube Music.
_Conversion from Apple Music coming soon_

## Setup

---

```shell
cd your-repo
go mod tidy
```

<br />

**Set up the required environment variables in** `internal/env/.env`

<br />

**Start the API server:**

```shell
go run cmd/app/main.go
```

<br />

## Flow

---

_verify playlist url -> authorize -> convert_

<br />

## Endpoints

---

- #### `GET /api/auth/spotify`

```
 Description: Initiates the authentication flow with Spotify.
 Status: 307
 Response: Temporary redirect to the Spotify authorization page.
```

- #### GET /api/auth/spotify_callback

```
Description: Callback endpoint for Spotify authentication.
Status: 302
Response: Redirect to the UI
```

- #### GET /api/auth/youtube

```
Description: Initiates the authentication flow with YouTube.
Status: 307
Response: Temporary redirect to the YouTube authorization page.
```

- #### GET /api/auth/youtube_callback

```
Description: Callback endpoint for YouTube authentication.
Status: 302
Response: Redirect to the UI
```

- #### GET /api/playlist/verify

```
Description: Verify if a playlist URL is valid.
Method: GET
Query Parameter: url - The URL of the playlist to verify.
Response: JSON data with verification result.
```

**Example Response**

```json
{
  "isPlaylistValid": true,
  "supportedConversions": ["spotify", "apple-music"],
  "playlistData": {
    "url": "string",
    "thumbnailUrl": "string",
    "tracks": [],
    "playlistName": "string",
    "playlistTracksCount": 5,
    "source": "string"
  }
}
```

- #### GET /api/playlist/convert/preview

```
Description: Preview the playlist to be converted.
Session Required: Yes
Response: JSON data containing the preview of the converted playlist.
```

- #### POST /api/playlist/convert/start

```
Description: Start the playlist conversion process.
Session Required: Yes
Request Body : {"title" : "string"}
Status: 201 Created
Response Body: JSON data containing the new URL of the converted playlist or an error message.
```

- #### GET /api/playlist/convert/start/stream

```
Description: Start the playlist conversion process and stream process.
Session Required: Yes
Query Parameter: title - The title of the converted playlist.
Response Content-Type: text/stream
```

**Example Response**

```text/stream
event: create
data:  {url: ""}

event: search
data:  {"trackId": "string","status": "searching"}

event: convert
data:  {"trackId": "string","status": "success"}

event: done
data: {"status": "success"}
```
