# Youtube / Spotify - Concert finder

This is a command line application which return a list of concerts on Youtube, based on the bands you follow on Spotify.

## TODO

- Create a `.env.example` based `.env`
- Create a parameter to allow the sort on results (e.g. `-sort=time.asc`). The sort options must be based on sort options on youtube API 
- Have a way to avoid Spotify authentication every time the application runs
- Update the README with more relevant things (like how to build, etc)

## Setup

### Youtube API

Talking to the Youtube API requires oauth2 authentication. As such, you must:

1. Create an account on the [Google Developers Console](https://console.developers.google.com)
1. Register a new app there
1. Enable the Youtube API (APIs & Auth -> APIs)
1. Create Client ID (APIs & Auth -> Credentials), select 'Web application'
1. Add an 'Authorized redirect URI' of 'http://localhost:8080/oauth2callback'
1. Take note of the `Client ID` and `Client secret` values

The utility looks for `client_secrets.json` in the local directory. Create it first using the details from above:

```
{
  "installed": {
    "client_id": "xxxxxxxxxxxxxxxxxxx.apps.googleusercontent.com",
    "client_secret": "xxxxxxxxxxxxxxxxxxxxx",
    "redirect_uris": ["http://localhost:8080/oauth2callback"],
    "auth_uri": "https://accounts.google.com/o/oauth2/auth",
    "token_uri": "https://accounts.google.com/o/oauth2/token"
  }
}
```

Update `client_id` and `client_secret` to match your details