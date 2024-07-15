package postpage

import (
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"forum/backend/requests"
)

func PostPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "ERROR: Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	postId := r.FormValue("id")

	data, err := requests.GetPostWithComments("http://localhost:8080/api/postandcomments", postId)
	if err != nil {
		http.Error(w, "ERROR: Cannot get post and comments", http.StatusBadRequest)
		return
	}

	tmpl, err := template.ParseFiles("frontend/pages/postPage/postPage.html")
	if err != nil {
		http.Error(w, "ERROR: Unable to parse template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "ERROR: Unable to execute template", http.StatusInternalServerError)
		return
	}
}

func PostPageCreateComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "ERROR: Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	postId := r.FormValue("id")
	comment := r.FormValue("comment")

	cookie, cookieErr := r.Cookie("session_token")
	if cookieErr != nil {
		http.Error(w, "ERROR: You are not authorized to create comment", http.StatusUnauthorized)
		return
	}

	image, header, err := r.FormFile("image")
	var imagePath string
	if err == nil {
		defer image.Close()

		// Validate the image type
		if !isValidImageType(header) {
			http.Error(w, "ERROR: Only PNG, JPEG, and GIF images are allowed", http.StatusBadRequest)
			return
		}

		if header.Size > 20*1024*1024 { // 20MB limit
			errorMsg := fmt.Sprintf("ERROR: Image size (%.2f MB) exceeds 20MB limit", float64(header.Size)/1024/1024)
			http.Error(w, errorMsg, http.StatusBadRequest)
			return
		}

		// Create a unique file name
		imageFileName := fmt.Sprintf("%d-%s", time.Now().Unix(), header.Filename)
		imagePath = filepath.Join("uploads", "comments", imageFileName)
		outFile, err := os.Create(imagePath)
		if err != nil {
			http.Error(w, "ERROR: Could not save the image", http.StatusInternalServerError)
			return
		}
		defer outFile.Close()
		_, err = io.Copy(outFile, image)
		if err != nil {
			http.Error(w, "ERROR: Could not save the image", http.StatusInternalServerError)
			return
		}
	}

	err = requests.CreateCommentRequest("http://localhost:8080/api/createcomment", postId, comment, cookie.Value, imagePath)
	if err != nil {
		http.Error(w, "ERROR: Bad request", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/post?id="+postId, http.StatusSeeOther)
}

func PostPageUpVote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "ERROR: Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	isComment := r.FormValue("isComment")
	postId := r.FormValue("post_id")

	cookie, cookieErr := r.Cookie("session_token")
	if cookieErr != nil {
		http.Error(w, "ERROR: You are not authorized to up vote", http.StatusUnauthorized)
		return
	}

	err := requests.VoteRequest("http://localhost:8080/api/upvote", id, isComment, postId, cookie.Value)
	if err != nil {
		http.Error(w, "ERROR: Bad request", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/post?id="+postId, http.StatusSeeOther)
}

func PostPageDownVote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "ERROR: Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	isComment := r.FormValue("isComment")
	postId := r.FormValue("post_id")

	cookie, cookieErr := r.Cookie("session_token")
	if cookieErr != nil {
		http.Error(w, "ERROR: You are not authorized to down vote", http.StatusUnauthorized)
		return
	}

	err := requests.VoteRequest("http://localhost:8080/api/downvote", id, isComment, postId, cookie.Value)
	if err != nil {
		http.Error(w, "ERROR: Bad request", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/post?id="+postId, http.StatusSeeOther)
}

func isValidImageType(header *multipart.FileHeader) bool {
	// Dosya uzantısını kontrol et
	switch filepath.Ext(header.Filename) {
	case ".jpg", ".jpeg", ".png", ".gif", ".PNG", ".JPEG", ".GIF":
		// MIME türünü kontrol et
		file, err := header.Open()
		if err != nil {
			return false
		}
		defer file.Close()

		buffer := make([]byte, 512)
		_, err = file.Read(buffer)
		if err != nil {
			return false
		}

		contentType := http.DetectContentType(buffer)
		return contentType == "image/jpeg" || contentType == "image/png" || contentType == "image/gif"
	default:
		return false
	}
}
