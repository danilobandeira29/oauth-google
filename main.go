package main

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"log"
	"net/http"
	"os"
)

var (
	token    *oauth2.Token
	config   *oauth2.Config
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("error when trying to load .env file %v\n", err)
	}
	clientId := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	if clientId == "" || clientSecret == "" {
		log.Fatalf("env variable 'CLIENT_ID' and 'CLIENT_SECRET' are mandatory\n")
	}
	provider, err = oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		log.Fatalf("error when instantiate new OIDC provider %v\n", err)
	}
	config = &oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{oidc.ScopeOpenID, "https://www.googleapis.com/auth/drive.readonly", "https://www.googleapis.com/auth/userinfo.profile", "profile", "email"},
	}
	verifier = provider.Verifier(&oidc.Config{ClientID: clientId})
}

func main() {
	server := http.NewServeMux()
	server.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		html := `
<html>
	<body>
		<a href="/login">Login with Google</a>
	</body>
</html>`
		if _, err := fmt.Fprint(w, html); err != nil {
			log.Fatalf("error when trying to return to client %v\n", err)
		}
	})
	server.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		url := config.AuthCodeURL("random-state", oauth2.AccessTypeOnline)
		http.Redirect(w, r, url, http.StatusFound)
	})
	server.HandleFunc("GET /callback", handlerCallback)
	server.HandleFunc("GET /profile", handlerProfile)
	server.HandleFunc("GET /token", handlerToken)
	server.HandleFunc("GET /files", handlerFiles)
	fmt.Println("server running at http://localhost:8080")
	log.Fatalln(http.ListenAndServe(":8080", server))
}
