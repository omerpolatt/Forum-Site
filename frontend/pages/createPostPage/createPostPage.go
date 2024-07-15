package createpostpage

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"forum/backend/requests"
)

const createPostApiUrl = "http://localhost:8080/api/createpost"

func CreatePostPage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "frontend/pages/createPostPage/createPostPage.html")
	case "POST":
		cookie, cookieErr := r.Cookie("session_token")
		if cookieErr != nil {
			http.Error(w, "ERROR: You are not authorized to create post", http.StatusUnauthorized)
			return
		}
		title := r.FormValue("title")
		content := r.FormValue("content")
		categoryDatas := GetCategoryDatas(r)

		// Handle image file
		image, header, err := r.FormFile("image")
		var imagePath string
		if err == nil {
			defer image.Close()
			// Create a unique file name
			imagePath = filepath.Join("uploads", header.Filename)
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

		err = requests.CreatePostRequest(createPostApiUrl, title, content, categoryDatas, cookie.Value, imagePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, "/myposts", http.StatusSeeOther)
	}
}

func GetCategoryDatas(r *http.Request) map[string]string {
	categories := []string{"go", "html", "css", "php", "python", "c", "cpp", "csharp", "js", "assembly", "react", "flutter", "rust"}
	categoryDatas := make(map[string]string)

	for _, category := range categories {
		if r.FormValue(category) == "true" {
			categoryDatas[category] = "true"
		} else {
			categoryDatas[category] = "false"
		}
	}
	return categoryDatas
}
