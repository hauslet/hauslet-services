package oauth

import (
	"context"
	"crypto/sha256"

	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/provider"
	"github.com/go-pkgz/auth/token"
	"golang.org/x/oauth2"
)

func setupGoogleProvider(service *auth.Service, deps Dependencies) {
	service.AddCustomProvider(
		"google",
		auth.Client{
			Cid:     deps.Config.GoogleClientID,
			Csecret: deps.Config.GoogleCLSecret,
		},
		provider.CustomHandlerOpt{
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
			InfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
			Scopes:  []string{"openid", "email", "profile"},
			MapUserFn: func(data provider.UserData, _ []byte) token.User {
				userInfo := token.User{
					ID:   "google_" + token.HashID(sha256.New(), data.Value("sub")),
					Name: data.Value("name"),
				}

				if email := data.Value("email"); email != "" {
					userInfo.Email = email
				}
				return userInfo
			},
		},
	)
}

func setupDirectProvider(service *auth.Service, deps Dependencies) {
	if deps.AuthenticatePassword == nil {
		deps.Log.Error("Direct provider is not configured: AuthenticatePassword is nil")
		return
	}

	service.AddDirectProvider("password", provider.CredCheckerFunc(func(user, password string) (ok bool, err error) {
		ctx := context.Background()
		if _, authErr := deps.AuthenticatePassword(ctx, user, password); authErr != nil {
			return false, nil
		}
		return true, nil
	}))
}
