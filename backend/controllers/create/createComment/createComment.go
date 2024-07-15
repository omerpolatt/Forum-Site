package createcomment

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"forum/backend/auth"
	"forum/backend/database"
)

func CreateComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "ERROR: Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	postId := r.FormValue("id")
	comment := r.FormValue("comment")

	if !IsForumCommentValid(postId, comment) {
		http.Error(w, "ERROR: Id or comment cannot empty", http.StatusBadRequest)
		return
	}

	postIdInt, atoiErr := strconv.Atoi(postId)
	if atoiErr != nil {
		http.Error(w, "ERROR: Invalid ID format", http.StatusBadRequest)
		return
	}

	db, errDb := database.OpenDb(w)
	if errDb != nil {
		http.Error(w, "ERROR: Database cannot open", http.StatusBadRequest)
		return
	}
	defer db.Close()

	authenticated, userId, userName := auth.IsAuthenticated(r, db)
	if !authenticated {
		http.Error(w, "ERROR: You are not authorized to create comment", http.StatusUnauthorized)
		return
	}
	var secondUserId int
	err := db.QueryRow("SELECT UserId FROM POSTS WHERE ID = ?", postIdInt).Scan(&secondUserId)
	if err != nil {
		http.Error(w, "ERROR: Invalid post ID", http.StatusBadRequest)
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

	_, errEx := db.Exec(`INSERT INTO COMMENTS (PostID, UserId, UserName, Comment,ImagePath) VALUES (?, ?, ?, ?,?)`, postIdInt, userId, userName, comment, imagePath)
	if errEx != nil {
		http.Error(w, "ERROR: Post did not add to the database", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Comment successfully created")
}

func IsForumCommentValid(id, comment string) bool {
	return id != "" && comment != ""
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
