package types

import (
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/instagram"
	"golang.org/x/oauth2/spotify"
	"os"
	"strconv"
)

type OauthResponse struct {
	Id      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"omitEmpty"`
}

type OauthProvider struct {
	oauth2.Config
	GetProfileRequestUrl   func(token *oauth2.Token) string
	GetGenericResponseData func(response map[string]interface{}) OauthResponse
}

var OauthProviders = map[string]*OauthProvider{
	AuthProviderGoogle:    &GoogleOauth,
	AuthProviderGitHub:    &GithubOauth,
	AuthProviderFacebook:  &FacebookOauth,
	AuthProviderInstagram: &InstagramOauth,
	AuthProviderSpotify:   &SpotifyOauth,
}

var GoogleOauth = OauthProvider{
	Config: oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_SECRET_ID"),
		RedirectURL:  fmt.Sprintf("%s/oauth/%s/redirect", os.Getenv("OAUTH_CALLBACK_HOST"), AuthProviderGoogle),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	},
	GetProfileRequestUrl: func(token *oauth2.Token) string {
		return "https://www.googleapis.com/oauth2/v2/userinfo"
	},
	GetGenericResponseData: func(response map[string]interface{}) OauthResponse {
		return googleResponseToGeneric(response)
	},
}

var FacebookOauth = OauthProvider{
	Config: oauth2.Config{
		ClientID:     os.Getenv("FACEBOOK_CLIENT_ID"),
		ClientSecret: os.Getenv("FACEBOOK_SECRET_ID"),
		RedirectURL:  fmt.Sprintf("%s/oauth/%s/redirect", os.Getenv("OAUTH_CALLBACK_HOST"), AuthProviderFacebook),
		Scopes: []string{
			"public_profile",
			"email",
		},
		Endpoint: facebook.Endpoint,
	},
	GetProfileRequestUrl: func(token *oauth2.Token) string {
		// Facebook doesn't accept auth headers
		return fmt.Sprintf("https://graph.facebook.com/me?fields=email,name,picture&access_token=%s", token.AccessToken)
	},
	GetGenericResponseData: func(response map[string]interface{}) OauthResponse {
		return facebookResponseToGeneric(response)
	},
}

var InstagramOauth = OauthProvider{
	Config: oauth2.Config{
		ClientID:     os.Getenv("FACEBOOK_CLIENT_ID"),
		ClientSecret: os.Getenv("FACEBOOK_SECRET_ID"),
		RedirectURL:  fmt.Sprintf("%s/oauth/%s/redirect", os.Getenv("OAUTH_CALLBACK_HOST"), AuthProviderInstagram),
		Scopes: []string{
			"public_profile",
			"email",
		},
		Endpoint: instagram.Endpoint,
	},
	GetProfileRequestUrl: func(token *oauth2.Token) string {
		// Facebook doesn't accept auth headers
		return fmt.Sprintf("https://graph.instagram.com/me?fields=email,name,picture&access_token=%s", token.AccessToken)
	},
	GetGenericResponseData: func(response map[string]interface{}) OauthResponse {
		return facebookResponseToGeneric(response)
	},
}

var SpotifyOauth = OauthProvider{
	Config: oauth2.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_SECRET_ID"),
		RedirectURL:  fmt.Sprintf("%s/oauth/%s/redirect", os.Getenv("OAUTH_CALLBACK_HOST"), AuthProviderSpotify),
		Scopes: []string{
			"user-read-private",
			"user-read-email",
		},
		Endpoint: spotify.Endpoint,
	},
	GetProfileRequestUrl: func(token *oauth2.Token) string {
		return "https://api.spotify.com/v1/me"
	},
	GetGenericResponseData: func(response map[string]interface{}) OauthResponse {
		return spotifyResponseToGeneric(response)
	},
}

var GithubOauth = OauthProvider{
	Config: oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_SECRET_ID"),
		RedirectURL:  fmt.Sprintf("%s/oauth/%s/redirect", os.Getenv("OAUTH_CALLBACK_HOST"), AuthProviderGitHub),
		Scopes:       []string{},
		Endpoint:     github.Endpoint,
	},
	GetProfileRequestUrl: func(token *oauth2.Token) string {
		return "https://api.github.com/user"
	},
	GetGenericResponseData: func(response map[string]interface{}) OauthResponse {
		return githubResponseToGeneric(response)
	},
}

func githubResponseToGeneric(response map[string]interface{}) OauthResponse {
	return OauthResponse{
		Id:      strconv.FormatFloat(response["id"].(float64), 10, 2, 64),
		Email:   response["email"].(string),
		Name:    response["name"].(string),
		Picture: response["avatar_url"].(string),
	}
}

func googleResponseToGeneric(response map[string]interface{}) OauthResponse {
	return OauthResponse{
		Id:      response["id"].(string),
		Email:   response["email"].(string),
		Name:    response["name"].(string),
		Picture: response["picture"].(string),
	}
}

func facebookResponseToGeneric(response map[string]interface{}) OauthResponse {
	// These values are always strings, unless it's from facebook, then it's just a mess
	pictureUrl := response["picture"].(map[string]interface{})["data"].(map[string]interface{})["url"].(string)
	return OauthResponse{
		Id:      response["id"].(string),
		Email:   response["email"].(string),
		Name:    response["name"].(string),
		Picture: pictureUrl,
	}
}

func spotifyResponseToGeneric(response map[string]interface{}) OauthResponse {
	return OauthResponse{
		Id:      response["id"].(string),
		Email:   response["email"].(string),
		Name:    response["display_name"].(string),
		Picture: response["images"].([]string)[0],
	}
}
