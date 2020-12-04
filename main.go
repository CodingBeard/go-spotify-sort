package main

import (
	"fmt"
	"github.com/zmb3/spotify"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// redirectURI is the OAuth redirect URI for the application.
// You must register an application at Spotify's developer portal
// and enter this value.
const redirectURI = "http://localhost:8080/callback"

var (
	auth  = spotify.NewAuthenticator(redirectURI, spotify.ScopeUserReadPrivate, spotify.ScopePlaylistModifyPrivate, spotify.ScopePlaylistModifyPublic)
	ch    = make(chan *spotify.Client)
	state = "abc123"
)

func main() {
	// first start an HTTP server
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go http.ListenAndServe(":8080", nil)

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

	// wait for auth to complete
	client := <-ch

	playlistId := os.Getenv("PLAYLIST_ID")

	// use the client to make calls that require authorization
	user, err := client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)

	tracks := getTracks(client, playlistId)

	sortedTracks := sortTracks(tracks)

	originalPositions := make(map[spotify.ID]int)
	sortedPositions := make(map[spotify.ID]int)

	for originalPosition, originalTrack := range tracks {
		originalPositions[originalTrack.ID] = originalPosition
	}
	for sortedPosition, sortedTrack := range sortedTracks {
		sortedPositions[sortedTrack.ID] = sortedPosition
	}

	var startPosition int
	var fromPosition int
	var lastPosition int
	var length int

	for true {
		moved := false
		for sortedPosition, sortedTrack := range sortedTracks {
			if sortedPosition == 0 {
				startPosition = sortedPosition
				fromPosition = originalPositions[sortedTrack.ID]
				lastPosition = fromPosition
				continue
			}

			if lastPosition+1 != originalPositions[sortedTrack.ID] || length >= 100 {
				if fromPosition != startPosition {
					moved = true
					tracks = moveTracks(client, playlistId, tracks, fromPosition, startPosition, length)
					for originalPosition, originalTrack := range tracks {
						originalPositions[originalTrack.ID] = originalPosition
					}
				}
				lastPosition = originalPositions[sortedTrack.ID]
				startPosition = sortedPosition
				fromPosition = originalPositions[sortedTrack.ID]
				length = 1
				continue
			}

			length++

			lastPosition = originalPositions[sortedTrack.ID]
		}

		if !moved {
			break
		}
	}

	log.Println("Api reorders: ", apiCalls)

}

func getTracks(client *spotify.Client, playlistId string) []spotify.FullTrack {
	var tracks []spotify.FullTrack

	limit := 100
	offset := 0
	for true {
		playlistPager, err := client.GetPlaylistTracksOpt(spotify.ID(playlistId), &spotify.Options{
			Limit:  &limit,
			Offset: &offset,
		}, "")
		if err != nil {
			log.Fatal(err)
		}

		if len(playlistPager.Tracks) == 0 {
			break
		}

		for _, playlistTrack := range playlistPager.Tracks {
			tracks = append(tracks, playlistTrack.Track)
		}

		offset += 100

		log.Println("fetching tracks: ", offset, "/", playlistPager.Total)
	}

	return tracks
}

func sortTracks(tracks []spotify.FullTrack) []spotify.FullTrack {
	var sorted []spotify.FullTrack
	for _, track := range tracks {
		sorted = append(sorted, track)
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Artists[0].Name == sorted[j].Artists[0].Name {
			if sorted[i].Album.ReleaseDate == sorted[j].Album.ReleaseDate {
				if sorted[i].Album.Name == sorted[j].Album.Name {
					if sorted[i].DiscNumber == sorted[j].DiscNumber {
						if sorted[i].TrackNumber == sorted[j].TrackNumber {
							return true
						}
						return sorted[i].TrackNumber < sorted[j].TrackNumber
					}
					return sorted[i].DiscNumber < sorted[j].DiscNumber
				}
				return strings.Compare(sorted[i].Album.Name, sorted[j].Album.Name) == -1
			}
			return sorted[i].Album.ReleaseDateTime().Unix() < sorted[j].Album.ReleaseDateTime().Unix()
		}

		iname := sorted[i].Album.Artists[0].Name
		if iname == "Various Artists" {
			iname = sorted[i].Album.Name
		}
		jname := sorted[j].Album.Artists[0].Name
		if jname == "Various Artists" {
			jname = sorted[j].Album.Name
		}
		return strings.Compare(strings.ToLower(iname), strings.ToLower(jname)) == -1
	})

	return sorted
}

var apiCalls = 0

func moveTracks(client *spotify.Client, playlistId string, tracks []spotify.FullTrack, position int, newPosition int, length int) []spotify.FullTrack {
	apiCalls++
	log.Println("Moving track: ", tracks[position].Artists[0].Name, tracks[position].Name, "from", position, "to", newPosition, "length", length)

	_, err := client.ReorderPlaylistTracks(spotify.ID(playlistId), spotify.PlaylistReorderOptions{
		RangeStart:   position,
		RangeLength:  length,
		InsertBefore: newPosition,
	})
	if err != nil {
		log.Println(err.Error())
		time.Sleep(time.Second)
		return tracks
	}

	var val []spotify.FullTrack
	for _, track := range tracks[position : position+length] {
		val = append(val, track)
	}

	var removed []spotify.FullTrack
	removed = append(tracks[:position], tracks[position+length:]...)

	a := removed[:newPosition]
	b := removed[newPosition:]

	var newtracks []spotify.FullTrack

	newtracks = append(newtracks, a...)
	newtracks = append(newtracks, val...)
	newtracks = append(newtracks, b...)

	tracks = newtracks

	return tracks
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	// use the token to get an authenticated client
	client := auth.NewClient(tok)
	client.AutoRetry = true
	fmt.Fprintf(w, "Login Completed!")
	ch <- &client
}
