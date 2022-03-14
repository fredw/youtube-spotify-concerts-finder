package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

var (
	youtubeClient = make(chan *http.Client)
	youtubeState  = uuid.New().String()
)

//const missingClientSecretsMessage = `
//Please configure OAuth 2.0
//To make this sample run, you need to populate the client_secrets.json file
//found at:
//   %v
//with information from the {{ Google Cloud Console }}
//{{ https://cloud.google.com/console }}
//For more information about the client_secrets.json file format, please visit:
//https://developers.google.com/api-client-library/python/guide/aaa_client_secrets
//`

//var (
//	//clientSecretsFile = flag.String("secrets", "client_secrets.json", "Client Secrets configuration")
//	clientSecretsFile = "client_secrets.json"
//	//cache             = flag.String("cache", "request.token", "token cache file")
//)
//
//// ClientConfig is a data structure definition for the client_secrets.json file.
//// The code unmarshals the JSON configuration file into this structure.
//type ClientConfig struct {
//	ClientID     string   `json:"client_id"`
//	ClientSecret string   `json:"client_secret"`
//	RedirectURIs []string `json:"redirect_uris"`
//	AuthURI      string   `json:"auth_uri"`
//	TokenURI     string   `json:"token_uri"`
//}
//
//// readConfig reads the configuration from clientSecretsFile.
//// It returns an oauth configuration object for use with the Google API client.
//func readConfig(scopes []string) (*oauth2.Config, error) {
//	// Read the secrets file
//	data, err := ioutil.ReadFile(clientSecretsFile)
//	if err != nil {
//		pwd, _ := os.Getwd()
//		fullPath := filepath.Join(pwd, clientSecretsFile)
//		return nil, fmt.Errorf(missingClientSecretsMessage, fullPath)
//	}
//
//	cfg := new(ClientConfig)
//	err = json.Unmarshal(data, &cfg)
//	if err != nil {
//		return nil, err
//	}
//
//	var oCfg *oauth2.Config
//
//	redirURL := ""
//	if len(cfg.RedirectURIs) > 0 {
//		redirURL = cfg.RedirectURIs[0]
//	} else {
//		fmt.Printf("Redirect URL could not be found. Using default: http://localhost:8080/oauth2callback")
//		redirURL = "http://localhost:8080/oauth2callback"
//	}
//
//	oCfg = &oauth2.Config{
//		ClientID:     cfg.ClientID,
//		ClientSecret: cfg.ClientSecret,
//		Scopes:       scopes,
//		Endpoint: oauth2.Endpoint{
//			AuthURL:  cfg.AuthURI,
//			TokenURL: cfg.TokenURI,
//		},
//		RedirectURL: redirURL,
//	}
//	return oCfg, nil
//}

type configYoutube struct {
	// ID from your application: https://console.developers.google.com/apis/credentials
	ID string `envconfig:"ID" required:"true"`
	// Secret from your application: https://console.developers.google.com/apis/credentials
	Secret string `envconfig:"SECRET" required:"true"`
	// CallbackPath the URL path to process the OAuth callback
	CallbackPath string `envconfig:"CALLBACK_PATH" required:"true" default:"/youtube/callback"`
	// TokenExpireDuration (e.g.: 2h45m) (units are "ns", "us", "ms", "s", "m", "h")
	TokenExpireDuration string `envconfig:"TOKEN_EXPIRE_DURATION" default:"0"`
}

// getYoutubeClient returns the youtube client
func getYoutubeClient(c config) (*youtube.Service, error) {
	auth := NewYoutubeAuthenticator(
		fmt.Sprintf("http://localhost:%v%s", c.Port, c.Youtube.CallbackPath),
		youtube.YoutubeScope,
		youtube.YoutubeReadonlyScope,
	)

	http.HandleFunc(c.Youtube.CallbackPath, youtubeCompleteAuth(c, auth))

	go func() {
		log.Printf("serving http server on port: %v", c.Port)
		err := http.ListenAndServe(fmt.Sprintf(":%v", c.Port), nil)
		if err != nil {
			log.Fatalf("fail to listen http requests: %v", err)
		}
	}()

	//url := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	tokenFile, err := ioutil.ReadFile("youtube-token.json")

	// if fails to read the token file, request the authorization
	if err != nil {
		url := auth.AuthURL(youtubeState)
		fmt.Println("please log in to Google by visiting the following page in your browser:", url)
		// wait for auth to complete
		client := <-youtubeClient

		//   config := &oauth2.Config{...}
		//   // ...
		//   token, err := config.Exchange(ctx, ...)
		//   youtubeService, err := youtube.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, token)))

		return youtube.NewService(
			context.Background(),
			option.WithHTTPClient(client),
		)
	}

	// otherwise uses the stored token
	var token *oauth2.Token
	err = json.Unmarshal(tokenFile, &token)
	if err != nil {
		log.Fatalf("problem to unmarshal Google token: %v", err)
	}

	client := auth.NewClient(token)
	return youtube.NewService(
		context.Background(),
		option.WithHTTPClient(client),
	)
}

const (
	// AuthURL is the URL to google Accounts Service's OAuth2 endpoint.
	AuthURL = "https://accounts.google.com/o/oauth2/auth"
	// TokenURL is the URL to the Google Accounts Service's OAuth2
	// token endpoint.
	TokenURL = "https://oauth2.googleapis.com/token"
)

// Authenticator provides convenience functions for implementing the OAuth2 flow.
// You should always use `NewAuthenticator` to make them.
//
// Example:
//
//     a := spotify.NewAuthenticator(redirectURL, spotify.ScopeUserLibaryRead, spotify.ScopeUserFollowRead)
//     // direct user to Spotify to log in
//     http.Redirect(w, r, a.AuthURL("state-string"), http.StatusFound)
//
//     // then, in redirect handler:
//     token, err := a.Token(state, r)
//     client := a.NewClient(token)
//
type YoutubeAuthenticator struct {
	config  *oauth2.Config
	context context.Context
}

// NewAuthenticator creates an authenticator which is used to implement the
// OAuth2 authorization flow.  The redirectURL must exactly match one of the
// URLs specified in your Spotify developer account.
//
// By default, NewAuthenticator pulls your client ID and secret key from the
// YOUTUBE_ID and YOUTUBE_SECRET environment variables.  If you'd like to provide
// them from some other source, you can call `SetAuthInfo(id, key)` on the
// returned authenticator.
func NewYoutubeAuthenticator(redirectURL string, scopes ...string) YoutubeAuthenticator {
	cfg := &oauth2.Config{
		ClientID:     os.Getenv("YOUTUBE_ID"),
		ClientSecret: os.Getenv("YOUTUBE_SECRET"),
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  AuthURL,
			TokenURL: TokenURL,
		},
	}

	// disable HTTP/2 for DefaultClient, see: https://github.com/zmb3/spotify/issues/20
	tr := &http.Transport{
		TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Transport: tr})
	return YoutubeAuthenticator{
		config:  cfg,
		context: ctx,
	}
}

// SetAuthInfo overwrites the client ID and secret key used by the authenticator.
// You can use this if you don't want to store this information in environment variables.
func (a *YoutubeAuthenticator) SetAuthInfo(clientID, secretKey string) {
	a.config.ClientID = clientID
	a.config.ClientSecret = secretKey
}

// AuthURL returns a URL to the the Google Accounts Service's OAuth2 endpoint.
//
// State is a token to protect the user from CSRF attacks.  You should pass the
// same state to `Token`, where it will be validated.  For more info, refer to
// http://tools.ietf.org/html/rfc6749#section-10.12.
func (a YoutubeAuthenticator) AuthURL(state string) string {
	return a.config.AuthCodeURL(state)
}

// NewClient creates a Client that will use the specified access token for its API requests.
func (a YoutubeAuthenticator) NewClient(token *oauth2.Token) *http.Client {
	return a.config.Client(a.context, token)
}

// Token pulls an authorization code from an HTTP request and attempts to exchange
// it for an access token.  The standard use case is to call Token from the handler
// that handles requests to your application's redirect URL.
func (a YoutubeAuthenticator) Token(state string, r *http.Request) (*oauth2.Token, error) {
	values := r.URL.Query()
	if e := values.Get("error"); e != "" {
		return nil, errors.New("google: auth failed - " + e)
	}
	code := values.Get("code")
	if code == "" {
		return nil, errors.New("google: didn't get access code")
	}
	actualState := values.Get("state")
	if actualState != state {
		return nil, errors.New("google: redirect state parameter doesn't match")
	}
	return a.config.Exchange(a.context, code)
}

// youtubeCompleteAuth handles the callback from google after the authorization
func youtubeCompleteAuth(c config, auth YoutubeAuthenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Token(youtubeState, r)
		if err != nil {
			http.Error(w, "Couldn't get token", http.StatusForbidden)
			log.Fatal(err)
		}
		if st := r.FormValue("state"); st != youtubeState {
			http.NotFound(w, r)
			log.Fatalf("State mismatch: %s != %s\n", st, youtubeState)
		}

		// sets token expiration
		if c.Youtube.TokenExpireDuration != "" {
			if c.Youtube.TokenExpireDuration == "0" {
				token.Expiry = time.Unix(0, 0)
			} else {
				duration, err := time.ParseDuration(c.Youtube.TokenExpireDuration)
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
		err = ioutil.WriteFile("youtube-token.json", tokenDecoded, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}

		// use the token to get an authenticated client
		client := auth.NewClient(token)
		youtubeClient <- client

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		_, err = w.Write([]byte("Youtube authorization completed successfully!\r\nYou can now safely close this browser window."))
		if err != nil {
			log.Fatal(err)
		}
	}
}
