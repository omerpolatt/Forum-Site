package requests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"forum/backend/auth"
	"forum/backend/controllers/structs"
	"forum/backend/database"
)

func GetDataForServe(apiURL string) ([]structs.Post, error) {
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var posts []structs.Post
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, err
	}

	return posts, nil
}

func GetDataForServeWithReq(apiURL string, cookieValue string) ([]structs.Post, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}

	req.AddCookie(&http.Cookie{Name: "session_token", Value: cookieValue})

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return nil, nil
	}
	defer resp.Body.Close()

	var posts []structs.Post
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, err
	}

	return posts, nil
}

func GetCommentDataForServeWithReq(apiURL string, cookieValue string) ([]structs.Comment, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}

	req.AddCookie(&http.Cookie{Name: "session_token", Value: cookieValue})

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return nil, nil
	}
	defer resp.Body.Close()

	var posts []structs.Comment
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, err
	}

	return posts, nil
}

func GetPostWithComments(apiURL string, postId string) (structs.PostWithComments, error) {
	req, err := http.NewRequest("GET", apiURL+"?id="+postId, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return structs.PostWithComments{}, err
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return structs.PostWithComments{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return structs.PostWithComments{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var data structs.PostWithComments
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return structs.PostWithComments{}, err
	}

	return data, nil
}

func GetSearchedDataForServeWithReq(apiURL string, filter string, category string, search string) ([]structs.Post, error) {
	req, err := http.NewRequest("GET", apiURL+"?filter="+filter+"&category="+category+"&search="+search, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return nil, nil
	}
	defer resp.Body.Close()

	var posts []structs.Post
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, err
	}

	return posts, nil
}

func RegisterRequest(apiURL string, email string, userName string, password string) error {
	formData := url.Values{}
	formData.Set("email", email)
	formData.Set("username", userName)
	formData.Set("password", password)

	encodedFormData := formData.Encode()

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(encodedFormData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return fmt.Errorf(bodyString)
	}

	return nil
}

func LoginRequest(apiURL string, email string, password string, w http.ResponseWriter) error {
	formData := url.Values{}
	formData.Set("email", email)
	formData.Set("password", password)

	encodedFormData := formData.Encode()

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(encodedFormData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return fmt.Errorf(bodyString)
	}

	db, errDb := database.OpenDb(w)
	if errDb != nil {
		http.Error(w, "ERROR: Database cannot open", http.StatusBadRequest)
		return errDb
	}
	defer db.Close()

	sessionToken, errToken := auth.CreateSessionToken()
	if errToken != nil {
		http.Error(w, "ERROR: Internal Server Error", http.StatusInternalServerError)
		return errToken
	}

	var userID int
	errQue := db.QueryRow("SELECT ID FROM USERS WHERE Email = ?", email).Scan(&userID)
	if errQue != nil {
		http.Error(w, "ERROR: Invalid email", http.StatusBadRequest)
		return err
	}

	errSetToken := auth.SetTokenInDatabase(w, db, sessionToken, userID)
	if errSetToken != nil {
		http.Error(w, "ERROR: Internal Server Error", http.StatusInternalServerError)
		return errSetToken
	}

	auth.SetCookie(w, sessionToken)

	return nil
}

func CreatePostRequest(apiURL string, title string, content string, categoryDatas map[string]string, cookieValue string, imagePath string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add title and content fields
	_ = writer.WriteField("title", title)
	_ = writer.WriteField("content", content)

	// Add categories fields
	for key, val := range categoryDatas {
		_ = writer.WriteField(key, val)
	}

	// Add image file
	if imagePath != "" {
		file, err := os.Open(imagePath)
		if err != nil {
			return err
		}
		defer file.Close()

		part, err := writer.CreateFormFile("image", filepath.Base(imagePath))
		if err != nil {
			return err
		}

		_, err = io.Copy(part, file)
		if err != nil {
			return err
		}
	}

	err := writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		return err
	}

	req.AddCookie(&http.Cookie{Name: "session_token", Value: cookieValue})
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return fmt.Errorf(bodyString)
	}

	return nil
}

func CreateCommentRequest(apiURL string, postId string, comment string, cookieValue string, imagePath string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add post ID and comment fields
	_ = writer.WriteField("id", postId)
	_ = writer.WriteField("comment", comment)

	// Add image file if it exists
	if imagePath != "" {
		file, err := os.Open(imagePath)
		if err != nil {
			return err
		}
		defer file.Close()

		part, err := writer.CreateFormFile("image", filepath.Base(imagePath))
		if err != nil {
			return err
		}

		_, err = io.Copy(part, file)
		if err != nil {
			return err
		}
	}

	err := writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		return err
	}

	req.AddCookie(&http.Cookie{Name: "session_token", Value: cookieValue})
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return fmt.Errorf(bodyString)
	}

	return nil
}

func VoteRequest(apiURL string, id string, isComment string, postId string, cookieValue string) error {
	formData := url.Values{}
	formData.Set("id", id)
	formData.Set("isComment", isComment)
	formData.Set("post_id", postId)

	encodedFormData := formData.Encode()

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(encodedFormData))
	if err != nil {
		return err
	}

	req.AddCookie(&http.Cookie{Name: "session_token", Value: cookieValue})

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return fmt.Errorf(bodyString)
	}

	return nil
}

func DeleteAccountRequest(apiURL string, password string, cookieValue string) error {
	formData := url.Values{}
	formData.Set("password", password)

	encodedFormData := formData.Encode()

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(encodedFormData))
	if err != nil {
		return err
	}

	req.AddCookie(&http.Cookie{Name: "session_token", Value: cookieValue})

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return fmt.Errorf(bodyString)
	}

	return nil
}

func DeletePostRequest(apiURL string, postId string, cookieValue string) error {
	formData := url.Values{}
	formData.Set("id", postId)

	encodedFormData := formData.Encode()

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(encodedFormData))
	if err != nil {
		return err
	}

	req.AddCookie(&http.Cookie{Name: "session_token", Value: cookieValue})

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return fmt.Errorf(bodyString)
	}

	return nil
}

func DeleteCommentRequest(apiURL string, commentId string, cookieValue string) error {
	formData := url.Values{}
	formData.Set("id", commentId)

	encodedFormData := formData.Encode()

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(encodedFormData))
	if err != nil {
		return err
	}

	req.AddCookie(&http.Cookie{Name: "session_token", Value: cookieValue})

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return fmt.Errorf(bodyString)
	}

	return nil
}

// GetAllUsersRequest admin sayfasında tüm kullanıcı bilgilerini çeker
func GetAllUsers(db *sql.DB) ([]structs.User, error) {
	query := "SELECT id, username, email, role FROM USERS"
	rows, err := db.Query(query)
	if err != nil {
		fmt.Println("Error fetching users:", err)
		return nil, err
	}
	defer rows.Close()

	var users []structs.User
	for rows.Next() {
		var user structs.User
		err := rows.Scan(&user.ID, &user.UserName, &user.Email, &user.Role)
		if err != nil {
			fmt.Println("Error scanning user:", err)
			return nil, err
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		fmt.Println("Error during rows iteration:", err)
		return nil, err
	}

	return users, nil
}

// PromoteUserRequest kullanıcının rolünü günceller
func PromoteUserRequest(apiURL string, username string, newRole string, cookieValue string) error {
	formData := url.Values{}
	formData.Set("username", username)
	formData.Set("role", newRole)

	encodedFormData := formData.Encode()

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(encodedFormData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: cookieValue})

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return fmt.Errorf(bodyString)
	}

	return nil
}
