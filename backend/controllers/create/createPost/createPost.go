package createpost

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"forum/backend/auth"
	"forum/backend/database"
)

func CreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "ERROR: Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")

	if !IsForumPostValid(title, content) {
		http.Error(w, "ERROR: Content or title cannot empty", http.StatusBadRequest)
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
		http.Error(w, "ERROR: You are not authorized to create post", http.StatusUnauthorized)
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
		imagePath = filepath.Join("imageuploads", "posts", imageFileName)
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

	result, errEx := db.Exec(`INSERT INTO POSTS (UserID, UserName, Title, Content, ImagePath) VALUES (?, ?, ?, ?, ?)`, userId, userName, title, content, imagePath)
	if errEx != nil {
		http.Error(w, "ERROR: Post did not add to the database", http.StatusBadRequest)
		return
	}

	postID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "ERROR: Could not retrieve post ID", http.StatusBadRequest)
		return
	}

	categoryValues := GetCategoryValues(r)
	_, err = db.Exec(`INSERT INTO CATEGORIES (USERID, PostID, GO, HTML, CSS, PHP, PYTHON, C, "CPP", "CSHARP", JS, ASSEMBLY, REACT, FLUTTER, RUST) 
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userId, postID, categoryValues["go"], categoryValues["html"], categoryValues["css"], categoryValues["php"],
		categoryValues["python"], categoryValues["c"], categoryValues["cpp"], categoryValues["csharp"],
		categoryValues["js"], categoryValues["assembly"], categoryValues["react"], categoryValues["flutter"], categoryValues["rust"])
	if err != nil {
		http.Error(w, "ERROR: Could not add categories to the database", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Post successfully created")
}

func IsForumPostValid(title, content string) bool {
	return title != "" && content != ""
}

func GetCategoryValues(r *http.Request) map[string]int {
	categories := []string{"go", "html", "css", "php", "python", "c", "cpp", "csharp", "js", "assembly", "react", "flutter", "rust"}
	categoryValues := make(map[string]int)

	for _, category := range categories {
		if r.FormValue(category) == "true" {
			categoryValues[category] = 1
		} else {
			categoryValues[category] = 0
		}
	}
	return categoryValues
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
