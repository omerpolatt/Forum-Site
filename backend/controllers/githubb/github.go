package github

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

var githubClientID string
var githubClientSecret string

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

	githubClientID = os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")

}

func GithubLoginHandler(w http.ResponseWriter, r *http.Request) {
	state := "login-" + uuid.Must(uuid.NewV4()).String()
	url := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&state=%s&scope=user:email&prompt=select_account",
		githubClientID,
		url.QueryEscape("http://localhost:8080/github/callback"),
		state,
	)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func GithubRegisterHandler(w http.ResponseWriter, r *http.Request) {
	state := "register-" + uuid.Must(uuid.NewV4()).String()
	url := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&state=%s&scope=user:email&prompt=select_account",
		githubClientID,
		url.QueryEscape("http://localhost:8080/github/callback"),
		state,
	)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func GithubCallbackHandler(w http.ResponseWriter, r *http.Request) {

	db, err := sql.Open("sqlite3", "./forum.db")
	if err != nil {
		http.Error(w, "ERROR: Database cannot open", http.StatusBadRequest)
		return
	}
	defer db.Close()
	if err := r.ParseForm(); err != nil {
		log.Println("Failed to parse form:", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	state := r.FormValue("state")
	if state == "" {
		log.Println("Invalid state")
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	if code == "" {
		log.Println("Invalid code")
		http.Error(w, "Invalid code", http.StatusBadRequest)
		return
	}

	token, err := GetGithubAccessToken(code)
	if err != nil {
		log.Println("Failed to get access token:", err)
		http.Error(w, fmt.Sprintf("Failed to get access token: %v", err), http.StatusInternalServerError)
		return
	}

	user, err := getGithubUser(token)
	if err != nil {
		log.Println("Failed to get user info:", err)
		http.Error(w, fmt.Sprintf("Failed to get user info: %v", err), http.StatusInternalServerError)
		return
	}

	// Kullanıcıyı veritabanına ekleyin veya oturum açmasını sağlayın
	var userID int
	err = db.QueryRow("SELECT ID FROM USERS WHERE Email = ?", user.Email).Scan(&userID)
	if err == sql.ErrNoRows {
		_, err = db.Exec("INSERT INTO USERS (Email, UserName, Password, Role, session_token) VALUES (?, ?, ?, ?, ?)", user.Email, user.Login, "", "User", "")
		if err != nil {
			log.Println("Error creating user:", err)
			http.Error(w, "Error creating user", http.StatusInternalServerError)
			return
		}
		err = db.QueryRow("SELECT ID FROM USERS WHERE Email = ?", user.Email).Scan(&userID)
		if err != nil {
			log.Println("Error retrieving user ID:", err)
			http.Error(w, "Error retrieving user ID", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		log.Println("Error retrieving user ID:", err)
		http.Error(w, "Error retrieving user ID", http.StatusInternalServerError)
		return
	}

	sessionToken, err := uuid.NewV4()
	if err != nil {
		log.Println("Error generating session token:", err)
		http.Error(w, "Error generating session token", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("UPDATE USERS SET session_token = ? WHERE ID = ?", sessionToken.String(), userID)
	if err != nil {
		log.Println("Error updating session token:", err)
		http.Error(w, "Error updating session token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken.String(),
		Expires:  time.Now().Add(24 * time.Hour),
		Path:     "/",
		HttpOnly: true,
	})

	// Kullanıcıyı uygun sayfaya yönlendir
	if strings.HasPrefix(state, "register") {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func GetGithubAccessToken(code string) (string, error) {
	values := url.Values{}
	values.Set("client_id", githubClientID)
	values.Set("client_secret", githubClientSecret)
	values.Set("code", code)
	values.Set("redirect_uri", "http://localhost:8080/github/callback")

	resp, err := http.Post("https://github.com/login/oauth/access_token", "application/x-www-form-urlencoded", strings.NewReader(values.Encode()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse response
	query, err := url.ParseQuery(string(body))
	if err != nil {
		return "", err
	}

	return query.Get("access_token"), nil
}

type GitHubUser struct {
	Login string `json:"login"`
	Email string `json:"email"`
	Name  string `json:"name"`
	ID    int    `json:"id"`
}

// getGithubUser fetches the GitHub user information using the provided token
func getGithubUser(token string) (GitHubUser, error) {
	var user GitHubUser

	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return user, err
	}
	req.Header.Set("Authorization", "token "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return user, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return user, fmt.Errorf("failed to get user info: %s", resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(&user)
	return user, err
}
