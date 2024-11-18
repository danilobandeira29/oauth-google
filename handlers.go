package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func handlerCallback(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("state") != "random-state" {
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}
	tokenEx, err := config.Exchange(context.Background(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "error when trying to exchange code for token", http.StatusBadRequest)
		log.Printf("error when trying to exchange code for token %v\n", err)
		return
	}
	token = tokenEx
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "id_token not found", http.StatusInternalServerError)
		return
	}
	idToken, err := verifier.Verify(context.Background(), rawIDToken)
	if err != nil {
		http.Error(w, "invalid id_token", http.StatusInternalServerError)
		return
	}
	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err = idToken.Claims(&claims); err != nil {
		http.Error(w, "error when trying to decode claims", http.StatusInternalServerError)
		return
	}
	req, err := http.NewRequest("GET", provider.UserInfoEndpoint(), nil)
	if err != nil {
		http.Error(w, "error when trying to create request for UserInfoEndpoint", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "error when trying to get user's info from identity provider", http.StatusInternalServerError)
		return
	}
	defer func(r io.ReadCloser) {
		if errClose := r.Close(); errClose != nil {
			log.Printf("callback: error when trying to close resp body %v\n", errClose)
		}
	}(resp.Body)
	result := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		http.Error(w, "cannot decode json", http.StatusInternalServerError)
		return
	}
	fmt.Printf("User's info: %v\n", result)
	fmt.Printf("ID Token: %s\n", rawIDToken)
	fmt.Printf("Access Token: %s\n", token.AccessToken)
	html := fmt.Sprintf(`
<html>
	<body>
		<p>Welcome back, %s</p>
		<p>Your email is: %s</p>
		<a href="/files" style="display: block;">See User's Google Drive files names</a>
		<a href="/token" style="display: block;">See Token details</a>
	</body>
</html>
`, strings.Split(claims.Name, " ")[0], claims.Email[:3]+"*********@******")
	if _, errFprint := fmt.Fprintf(w, html); errFprint != nil {
		log.Fatalf("cannot send response to the client %v\n", errFprint)
	}
}

func handlerProfile(w http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		log.Fatalf("cannot create request to see user's profile %v\n", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("error when request to google profile api %v\n", err)
	}
	defer func(b io.ReadCloser) {
		if errClose := b.Close(); errClose != nil {
			log.Printf("error when tryint to close google drive response %v\n", errClose)
			return
		}
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		b, errRead := io.ReadAll(resp.Body)
		if errRead != nil {
			log.Fatalf("cannot read body response, status code: %d err: %v\n", resp.StatusCode, errRead)
		}
		if _, errPrint := fmt.Fprintf(w, "error response google api: %v\n", string(b)); errPrint != nil {
			log.Printf("error when send google api response to the client %v\n", errPrint)
			return
		}
	}
	type user struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}
	var u *user
	err = json.NewDecoder(resp.Body).Decode(&u)
	if err != nil {
		log.Fatalf("cannot decode resp from google profile api %v\n", err)
	}
	if _, err = fmt.Fprintf(w, "user's info\nName: %s\nEmail: %s\nPicture: %s\n", u.Name, u.Email, u.Picture); err != nil {
		log.Printf("cannot send response user's info back to the client %v\n", err)
		return
	}

}

func handlerToken(w http.ResponseWriter, r *http.Request) {
	if token == nil {
		html := `
<html>
	<body>
	<p>You need to login to be able to see token's details</p>
		<a href="/login">Login with Google</a>
	</body>
</html `
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(html)); err != nil {
			log.Printf("token: error when tryint to send response to client %v\n", err)
		}
		return
	}
	req, err := http.NewRequest("GET", "https://oauth2.googleapis.com/tokeninfo?access_token="+token.AccessToken, nil)
	if err != nil {
		log.Fatalf("cannot create request to get user's token info %v\n", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("error when trying to request user's token info %v\n", err)
	}
	defer func(b io.ReadCloser) {
		if errClose := b.Close(); errClose != nil {
			log.Printf("cannot close body %v\n", errClose)
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("error when trying to read response body %v\n", err)
	}
	if resp.StatusCode != http.StatusOK {
		if _, errPrint := fmt.Fprintf(w, "error: status code: %d body: %s\n", resp.StatusCode, string(body)); errPrint != nil {
			log.Printf("error when trying to send response to the client %v\n", errPrint)
		}
		return
	}
	if _, errPrint := fmt.Fprintf(w, "token info: %s\n", string(body)); errPrint != nil {
		log.Printf("error when tryint to send token info to client %v\n", errPrint)
		return
	}
}

func handlerFiles(w http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files", nil)
	if err != nil {
		log.Fatalf("cannot create request to see drive's files %v\n", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("error when request to google drive api %v\n", err)
	}
	defer func(b io.ReadCloser) {
		if errClose := b.Close(); errClose != nil {
			log.Printf("error when tryint to close google drive response %v\n", errClose)
			return
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("cannot read body %v\n", err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("status code is not OK: %d %s\n", resp.StatusCode, string(body))
	}
	result := make(map[string]interface{})
	if errJsonUnmarshal := json.Unmarshal(body, &result); errJsonUnmarshal != nil {
		log.Fatalf("error when trying to unmarshal body %v\n", errJsonUnmarshal)
	}
	files, ok := result["files"].([]interface{})
	if !ok {
		http.Error(w, "error when trying to iterate 'files' property", http.StatusBadRequest)
		return
	}
	for _, f := range files {
		info, okInfo := f.(map[string]interface{})
		if !okInfo {
			http.Error(w, "error when trying to iterate each file", http.StatusBadRequest)
			return
		}
		if _, errInfo := fmt.Fprintf(w, "file name: %s and id: %s\n", info["name"], info["id"]); errInfo != nil {
			log.Printf("error when tryint to return to client file's info %v\n", errInfo)
		}
	}
}
