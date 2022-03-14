package main

import (
	"fmt"
	"log"

	"github.com/kelseyhightower/envconfig"
)

type config struct {
	Port    int           `envconfig:"PORT" default:"8080"`
	Spotify configSpotify `envconfig:"SPOTIFY"`
	Youtube configYoutube `envconfig:"YOUTUBE"`
}

//type configGoogle struct {
//	// ApplicationCredentials is the file with the Google authentication secrets
//	ApplicationCredentials string `envconfig:"APPLICATION_CREDENTIALS" required:"true"`
//}

func main() {
	c := config{}
	err := envconfig.Process("", &c)
	if err != nil {
		log.Fatalf("problem to load configuration: %v", err)
	}

	//spotifyClient := getSpotifyClient(c)
	//
	//artists, err := getSpotifyFollowedArtists(spotifyClient, "")
	//if err != nil {
	//	log.Fatalf("couldn't get spotify artists: %v", err)
	//}
	//fmt.Println(artists)

	youtubeClient, err := getYoutubeClient(c)
	if err != nil {
		log.Fatalf("youtube error: %v", err)
	}
	response, err := youtubeClient.VideoCategories.List("contentDetails").Do()
	fmt.Println(response)
	fmt.Println(err)
}
