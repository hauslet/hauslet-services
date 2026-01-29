package oauth

import (
	"context"
	"crypto/sha256"

	"github.com/go-pkgz/auth/v2"
	"github.com/go-pkgz/auth/v2/provider"
	"github.com/go-pkgz/auth/v2/token"
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
			InfoURL: "https://www.googleapis.com/oauth2/v3/userinfo",
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

const passwordlessTokenTTLMinutes = 10

func setupEmailProvider(service *auth.Service, deps Dependencies) {
	if deps.SendPasswordlessEmail == nil {
		deps.Log.Info("Passwordless email provider not configured: SendPasswordlessEmail is nil")
		return
	}
	if deps.GenerateAndStoreOTP == nil {
		deps.Log.Info("Passwordless email provider not configured: GenerateAndStoreOTP is nil")
		return
	}

	sender := NewPasswordlessSender(
		deps.SendPasswordlessEmail,
		deps.GenerateAndStoreOTP,
		deps.Config.ClientRedirectURL, // base URL for magic links
		passwordlessTokenTTLMinutes,
	)

	// The template just passes the token - actual email rendering happens in SendPasswordlessEmail
	service.AddVerifProvider("email", "{{.Token}}", sender)
	deps.Log.Info("Passwordless email login provider enabled (hybrid: code + magic link)")
}
