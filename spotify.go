package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

var (
	spotifyClient = make(chan *spotify.Client)
	spotifyState  = uuid.New().String()
)

type configSpotify struct {
	// ID from your application: https://developer.spotify.com/dashboard/applications
	ID string `envconfig:"ID" required:"true"`
	// Secret from your application: https://developer.spotify.com/dashboard/applications
	Secret string `envconfig:"SECRET" required:"true"`
	// CallbackPath the URL path to process the OAuth callback
	CallbackPath string `envconfig:"CALLBACK_PATH" required:"true" default:"/spotify/callback"`
	// TokenExpireDuration (e.g.: 2h45m) (units are "ns", "us", "ms", "s", "m", "h")
	TokenExpireDuration string `envconfig:"TOKEN_EXPIRE_DURATION" default:"0"`
}

// getSpotifyClient returns the Spotify client
func getSpotifyClient(c config) *spotify.Client {
	auth := spotify.NewAuthenticator(
		fmt.Sprintf("http://localhost:%v%s", c.Port, c.Spotify.CallbackPath),
		spotify.ScopeUserFollowRead,
		spotify.ScopeUserLibraryRead,
		spotify.ScopeUserReadPrivate,
	)

	http.HandleFunc(c.Spotify.CallbackPath, spotifyCompleteAuth(c, auth))

	go func() {
		log.Printf("serving http server on port: %v", c.Port)
		err := http.ListenAndServe(fmt.Sprintf(":%v", c.Port), nil)
		if err != nil {
			log.Fatalf("fail to listen http requests: %v", err)
		}
	}()

	tokenFile, err := ioutil.ReadFile("spotify-token.json")

	// if fails to read the token file, request the authorization
	if err != nil {
		url := auth.AuthURL(spotifyState)
		fmt.Println("please log in to Spotify by visiting the following page in your browser:", url)
		// wait for auth to complete
		return <-spotifyClient
	}

	// otherwise uses the stored token
	var token *oauth2.Token
	err = json.Unmarshal(tokenFile, &token)
	if err != nil {
		log.Fatalf("problem to unmarshal Spotify token: %v", err)
	}
	client := auth.NewClient(token)
	return &client
}

// spotifyCompleteAuth handles the callback from spotify after the authorization
func spotifyCompleteAuth(c config, auth spotify.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Token(spotifyState, r)
		if err != nil {
			http.Error(w, "Couldn't get token", http.StatusForbidden)
			log.Fatal(err)
		}
		if st := r.FormValue("state"); st != spotifyState {
			http.NotFound(w, r)
			log.Fatalf("State mismatch: %s != %s\n", st, spotifyState)
		}

		// sets token expiration
		if c.Spotify.TokenExpireDuration != "" {
			if c.Spotify.TokenExpireDuration == "0" {
				token.Expiry = time.Unix(0, 0)
			} else {
				duration, err := time.ParseDuration(c.Spotify.TokenExpireDuration)
				if err != nil {
					log.Fatalf("prolem to parse token epire duration: %v", err)
				}
				token.Expiry = time.Now().Add(duration)
			}
		}

		// stores token in a local file
		tokenDecoded, err := json.Marshal(token)
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile("spotify-token.json", tokenDecoded, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}

		// use the token to get an authenticated client
		client := auth.NewClient(token)
		spotifyClient <- &client

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		_, err = w.Write([]byte("Spotify authorization completed successfully!\r\nYou can now safely close this browser window."))
		if err != nil {
			log.Fatal(err)
		}
	}
}

// getSpotifyFollowedArtists returns the list of artists
// to return all artist just send after as empty string
func getSpotifyFollowedArtists(client *spotify.Client, after string) ([]string, error) {
	facp, err := client.CurrentUsersFollowedArtistsOpt(50, after)
	if err != nil {
		return nil, err
	}

	var artists []string
	var lastID string
	for _, artist := range facp.Artists {
		artists = append(artists, artist.Name)
		lastID = artist.ID.String()
	}

	// if returned the maximum amount of artists, fetch the next page
	if len(facp.Artists) == 50 {
		a, err := getSpotifyFollowedArtists(client, lastID)
		if err != nil {
			return nil, err
		}
		artists = append(artists, a...)
	}
	return artists, nil
}
