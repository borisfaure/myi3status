package main

import (
	"github.com/zmb3/spotify"
)

// redirectURI is the OAuth redirect URI for the application.
// You must register an application at Spotify's developer portal
// and enter this value.
const redirectURI = "http://localhost:8531/callback"

type SpotifyCtx struct {
	auth spotify.Authenticator
}

func NewSpotifyCtx() SpotifyCtx {
	ctx := SpotifyCtx{}

	ctx.auth = spotify.NewAuthenticator(redirectURI, spotify.ScopeUserReadCurrentlyPlaying)
	return ctx
}

func (spotify_ctx SpotifyCtx) GetCurrentPlaying() (block I3ProtocolBlock, err error) {
	block.FullText = "Foo - Bar"
	return
}
