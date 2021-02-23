package types

import (
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
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

var GoogleOauth = OauthProvider{
	Config: oauth2.Config{
		ClientID:     "771130202315-b17g52r80dlcja1qkffsralopnkd17be.apps.googleusercontent.com",
		ClientSecret: "iv2qRoPnYKuhKv-OBvtEr97t",
		RedirectURL:  "http://localhost:8080/oauth/google/redirect",
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
		ClientID:     "5051487981591665",
		ClientSecret: "71f680f2c46e360f13401e4a3a397564",
		RedirectURL:  "http://localhost:8080/oauth/facebook/redirect",
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

var GithubOauth = OauthProvider{
	Config: oauth2.Config{
		ClientID:     "fab65d432ef4b19d33c7",
		ClientSecret: "4380b71c34e8b227f76c7107c3349067d33582b3",
		RedirectURL:  "http://localhost:8080/oauth/github/redirect",
		Scopes: []string{
		},
		Endpoint: github.Endpoint,
	},
	GetProfileRequestUrl: func(token *oauth2.Token) string {
		return "https://api.github.com/user"
	},
	GetGenericResponseData: func(response map[string]interface{}) OauthResponse {
		return githubResponseToGeneric(response)
	},
}

var OauthProviders = map[string]*OauthProvider{
	"google":   &GoogleOauth,
	"github":   &GithubOauth,
	"facebook": &FacebookOauth,
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
