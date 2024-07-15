package getpostandcomments

import (
	"encoding/json"
	"net/http"
	"strconv"

	"forum/backend/controllers/structs"
	"forum/backend/database"
)

func GetPostAndComments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "ERROR: Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	db, errDb := database.OpenDb(w)
	if errDb != nil {
		http.Error(w, "ERROR: Database cannot open", http.StatusBadRequest)
		return
	}
	defer db.Close()

	postId := r.FormValue("id")

	postIdInt, err := strconv.Atoi(postId)
	if err != nil {
		http.Error(w, "ERROR: ID cannot use", http.StatusBadRequest)
		return
	}

	var post structs.Post

	err = db.QueryRow("SELECT ID, UserID, UserName, Title, Content, LikeCount, ImagePath FROM POSTS WHERE ID = ?", postIdInt).Scan(&post.ID, &post.UserID, &post.UserName, &post.Title, &post.Content, &post.LikeCount, &post.ImagePath)
	if err != nil {
		http.Error(w, "ERROR: Query execution failed", http.StatusInternalServerError)
		return
	}

	commentRows, err := db.Query("SELECT ID, PostId, UserId, Comment, UserName, LikeCount, ImagePath FROM COMMENTS WHERE PostID = ?", postIdInt)
	if err != nil {
		http.Error(w, "ERROR: Query error for comments", http.StatusBadRequest)
		return
	}
	defer commentRows.Close()

	var comments []structs.Comment

	for commentRows.Next() {
		var comment structs.Comment
		err := commentRows.Scan(&comment.ID, &comment.PostId, &comment.UserId, &comment.Comment, &comment.UserName, &comment.LikeCount, &comment.ImagePath)
		if err != nil {
			http.Error(w, "ERROR: Database scan error", http.StatusBadRequest)
			return
		}
		comments = append(comments, comment)
	}

	if err = commentRows.Err(); err != nil {
		http.Error(w, "ERROR: Rows iteration error", http.StatusInternalServerError)
		return
	}

	data := structs.PostWithComments{
		Post:     post,
		Comments: comments,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		http.Error(w, "ERROR: Failed to encode posts to JSON", http.StatusInternalServerError)
		return
	}
}
