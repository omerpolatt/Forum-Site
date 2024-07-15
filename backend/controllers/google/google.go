package google

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
var googleClientID string
var googleClientSecret string

const redirectURI = "http://localhost:8080/callback/google"

// LoadEnv fonksiyonu .env dosyasından  değişkenleri çekeriz
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

	googleClientID = os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
}

// generateState fonksiyonu OAuth 2.0 state parametresi olarak kullanılacak rastgele bir dize üretir
func generateState() (string, error) {
	b := make([]byte, 16) // state güvenlikden sorumludur ve ayrıca giriş yapılacak platform arasında giriş yetkisi gibi yetkileri gönderdiğimiz parametremiz
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil // giriş yapılan kimliğin teyit edilmesi için gönderilen istek
}

// bu fonk ile kullanıcı google ile giriş yapmak için yönlendirildiği sayfayı çagırır
func HandleGoogleRegister(w http.ResponseWriter, r *http.Request) {
	// OAuth state parametresi üretir
	state, err := generateState()
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	session, err := store.Get(r, "session-name") // hesap olup olmadığını kontrol etmek için alıyoruz
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}
	// Oturumda oauthState ve action bilgilerini saklar
	session.Values["oauthState"] = state
	session.Values["action"] = "register"
	session.Save(r, w)

	// Google OAuth 2.0 yetkilendirme URL'ini oluşturur
	url := "https://accounts.google.com/o/oauth2/auth?client_id=" + googleClientID +
		"&redirect_uri=" + redirectURI +
		"&scope=https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/userinfo.email" +
		"&response_type=code" +
		"&state=" + state

	// Kullanıcıyı Google yetkilendirme sayfasına geçici olarak yönlendirir
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// registerden farkı yönlendirdiği action adı ile biri register biri login den kayıt işlemi yapması
func HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	// OAuth state parametresi üretir
	state, err := generateState()
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	// Kullanıcı oturumunu alır veya oturum açma hatası varsa hata mesajı döndürür
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}
	// Oturumda oauthState ve action bilgilerini saklar
	session.Values["oauthState"] = state
	session.Values["action"] = "login"
	session.Save(r, w)

	// Google OAuth 2.0 yetkilendirme URL'ini oluşturur
	url := "https://accounts.google.com/o/oauth2/auth?client_id=" + googleClientID +
		"&redirect_uri=" + redirectURI +
		"&scope=https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/userinfo.email" +
		"&response_type=code" +
		"&state=" + state

	// Kullanıcıyı Google yetkilendirme sayfasına geçici olarak yönlendirir
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Kullanıcı oturumunu alır veya oturum açma hatası varsa hata mesajı döndürür
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	// OAuth state parametresinin doğruluğunu kontrol eder
	state := r.URL.Query().Get("state")
	if state != session.Values["oauthState"] {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	// Veritabanını açar
	db, err := sql.Open("sqlite3", "./forum.db")
	if err != nil {
		http.Error(w, "ERROR: Database cannot open", http.StatusBadRequest)
		return
	}
	defer db.Close()

	// Google'dan dönen yetkilendirme kodunu alır
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code in response", http.StatusBadRequest)
		return
	}

	// Yetkilendirme kodunu erişim tokeni ile değiştirir
	token, err := exchangeGoogleCodeForToken(code)
	if err != nil {
		http.Error(w, "Failed to exchange code for token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Google kullanıcı bilgilerini alır
	userInfo, err := getGoogleUserInfo(token)
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	email, emailOk := userInfo["email"].(string)
	name, nameOk := userInfo["name"].(string)

	// Kullanıcı bilgilerinin geçerliliğini kontrol eder
	if !emailOk || !nameOk || email == "" {
		log.Printf("Google user info is missing required fields: %+v", userInfo)
		http.Error(w, "Failed to get valid user info", http.StatusInternalServerError)
		return
	}

	// Kullanıcı ID'sini saklamak için bir değişken oluşturur
	var userID int
	// Oturumdaki işlem türüne göre kullanıcıyı kaydeder veya giriş yapar
	action := session.Values["action"].(string)
	if action == "register" {
		// Kullanıcıyı veritabanında arar
		err = db.QueryRow("SELECT ID FROM USERS WHERE Email = ?", email).Scan(&userID)
		if err == sql.ErrNoRows {
			// Kullanıcı yoksa yeni kullanıcı kaydeder
			_, err = db.Exec("INSERT INTO USERS (Email, UserName, Password, AuthProvider) VALUES (?, ?, ?, ?)", email, name, "", "google")
			if err != nil {
				http.Error(w, "Failed to save user: "+err.Error(), http.StatusInternalServerError)
				return
			}
			// Yeni kaydedilen kullanıcının ID'sini alır
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
		// Kullanıcıyı veritabanında arar
		err = db.QueryRow("SELECT ID FROM USERS WHERE Email = ?", email).Scan(&userID)
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Oturum tokeni oluşturur
	sessionToken, err := uuid.NewV4()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Oturum tokenini çerez olarak ayarlar
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken.String(),
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	// Kullanıcının oturum tokenini veritabanında günceller
	_, err = db.Exec("UPDATE USERS SET session_token = ? WHERE ID = ?", sessionToken.String(), userID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Kullanıcıyı ana sayfaya yönlendirir
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// exchangeGoogleCodeForToken fonksiyonu, yetkilendirme kodunu Google'dan erişim tokeni ile değiştirir
func exchangeGoogleCodeForToken(code string) (string, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", googleClientID)
	data.Set("client_secret", googleClientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
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

// getGoogleUserInfo fonksiyonu, erişim tokenini kullanarak Google'dan kullanıcı bilgilerini alır
func getGoogleUserInfo(token string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
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
