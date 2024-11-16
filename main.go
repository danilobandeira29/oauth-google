package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"log"
	"net/http"
	"os"
)

var TOKEN *oauth2.Token
var CONFIG oauth2.Config

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
	CONFIG = oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/drive.readonly", "https://www.googleapis.com/auth/userinfo.profile"},
	}
}

func main() {
	server := http.NewServeMux()
	server.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		html := `<html><body><a href="/login">Login with Google</a></body></html>`
		if _, err := fmt.Fprint(w, html); err != nil {
			log.Fatalf("error when trying to return to client %v\n", err)
		}
	})
	server.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		url := CONFIG.AuthCodeURL("random-state", oauth2.AccessTypeOffline)
		http.Redirect(w, r, url, http.StatusFound)
	})
	server.HandleFunc("GET /callback", HandlerCallback)
	server.HandleFunc("GET /profile", HandlerProfile)
	server.HandleFunc("GET /token", HandlerToken)
	server.HandleFunc("GET /files", HandlerFiles)
	fmt.Println("server running at http://localhost:8080")
	log.Fatalln(http.ListenAndServe(":8080", server))
}
