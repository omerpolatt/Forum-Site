package facebook

import (
	"bufio"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
)

var store = sessions.NewCookieStore([]byte("a-very-secret-key"))
var facebookClientID string
var facebookClientSecret string

const facebookRedirectURI = "http://localhost:8080/callback/facebook"

func LoadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		log.Fatalf("Error opening .env file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]
		os.Setenv(key, value)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading .env file: %v", err)
	}

	facebookClientID = os.Getenv("FACEBOOK_CLIENT_ID")
	facebookClientSecret = os.Getenv("FACEBOOK_CLIENT_SECRET")
}

func generateState() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func HandleFacebookRegister(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}
	session.Values["oauthState"] = state
	session.Values["action"] = "register"
	session.Save(r, w)

	url := "https://www.facebook.com/v10.0/dialog/oauth?client_id=" + facebookClientID +
		"&redirect_uri=" + facebookRedirectURI +
		"&scope=email" +
		"&response_type=code" +
		"&state=" + state

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func HandleFacebookLogin(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}
	session.Values["oauthState"] = state
	session.Values["action"] = "login"
	session.Save(r, w)

	url := "https://www.facebook.com/v10.0/dialog/oauth?client_id=" + facebookClientID +
		"&redirect_uri=" + facebookRedirectURI +
		"&scope=email" +
		"&response_type=code" +
		"&state=" + state
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func HandleFacebookCallback(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	state := r.URL.Query().Get("state")
	if state != session.Values["oauthState"] {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	db, err := sql.Open("sqlite3", "./forum.db")
	if err != nil {
		http.Error(w, "ERROR: Database cannot open", http.StatusBadRequest)
		return
	}
	defer db.Close()

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code in response", http.StatusBadRequest)
		return
	}

	token, err := exchangeFacebookCodeForToken(code)
	if err != nil {
		http.Error(w, "Failed to exchange code for token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userInfo, err := getFacebookUserInfo(token)
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	email, emailOk := userInfo["email"].(string)
	name, nameOk := userInfo["name"].(string)

	if !emailOk || !nameOk || email == "" {
		log.Printf("Facebook user info is missing required fields: %+v", userInfo)
		http.Error(w, "Failed to get valid user info", http.StatusInternalServerError)
		return
	}

	var userID int
	action := session.Values["action"].(string)
	if action == "register" {
		err = db.QueryRow("SELECT ID FROM USERS WHERE Email = ?", email).Scan(&userID)
		if err == sql.ErrNoRows {
			_, err = db.Exec("INSERT INTO USERS (Email, UserName, Password, AuthProvider) VALUES (?, ?, ?, ?)", email, name, "", "facebook")
			if err != nil {
				http.Error(w, "Failed to save user: "+err.Error(), http.StatusInternalServerError)
				return
			}
			err = db.QueryRow("SELECT ID FROM USERS WHERE Email = ?", email).Scan(&userID)
			if err != nil {
				http.Error(w, "Failed to retrieve user ID after insert: "+err.Error(), http.StatusInternalServerError)
				return
			}
		} else if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else if action == "login" {
		err = db.QueryRow("SELECT ID FROM USERS WHERE Email = ?", email).Scan(&userID)
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	sessionToken, err := uuid.NewV4()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken.String(),
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	_, err = db.Exec("UPDATE USERS SET session_token = ? WHERE ID = ?", sessionToken.String(), userID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func exchangeFacebookCodeForToken(code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", facebookClientID)
	data.Set("redirect_uri", facebookRedirectURI)
	data.Set("client_secret", facebookClientSecret)
	data.Set("code", code)

	resp, err := http.PostForm("https://graph.facebook.com/v10.0/oauth/access_token", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if token, ok := result["access_token"].(string); ok {
		return token, nil
	}
	return "", fmt.Errorf("no access token in response")
}

func getFacebookUserInfo(token string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", "https://graph.facebook.com/me?fields=id,name,email", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&userInfo)
	return userInfo, nil
}
