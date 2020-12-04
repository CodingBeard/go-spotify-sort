#Spotify Sort

Will sort a playlist by:

    - Album artist
    - Album release date
    - Album name
    - Disc number
    - Track number

Start by registering your application at the following page:

https://developer.spotify.com/my-applications/.

You'll get a client ID and secret key for your application. An easy way to provide this data to your application is to set the SPOTIFY_ID and SPOTIFY_SECRET environment variables.

You will also need a PLAYLIST_ID environment variable for the playlist to sort. You can get this by right clicking a spotify playlist and copying the URI and taking the last string after the final semicolon.

Example usage:

```
SPOTIFY_ID=0928342342asfd230498 SPOTIFY_SECRET=120938134098asdf12039812 PLAYLIST_ID=asd98fyASDF8asdf go run main.go
```