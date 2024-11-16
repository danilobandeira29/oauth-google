package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func handlerCallback(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("state") != "random-state" {
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	tokenEx, err := CONFIG.Exchange(context.Background(), code)
	TOKEN = tokenEx
	if err != nil {
		http.Error(w, "error when trying to exchange code for token", http.StatusBadRequest)
		return
	}
	html := `
<html>
	<body>
		<a href="/files" style="display: block;">See User's Google Drive files names</a>
		<a href="/token" style="display: block;">See Token details</a>
	</body>
</html>
`
	if _, errFprint := fmt.Fprintf(w, html); errFprint != nil {
		log.Fatalf("cannot send response to the client %v\n", errFprint)
	}
}

func handlerProfile(w http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		log.Fatalf("cannot create request to see user's profile %v\n", err)
	}
	req.Header.Set("Authorization", "Bearer "+TOKEN.AccessToken)
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
	req, err := http.NewRequest("GET", "https://oauth2.googleapis.com/tokeninfo?access_token="+TOKEN.AccessToken, nil)
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
			return
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
	req.Header.Set("Authorization", "Bearer "+TOKEN.AccessToken)
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
