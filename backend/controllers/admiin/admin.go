package admin

import (
	"database/sql"
	"encoding/json"
	"forum/backend/database"
	"forum/backend/requests"
	"net/http"
)

func Admin(w http.ResponseWriter, r *http.Request) {
	db, errDb := database.OpenDb(w)
	if errDb != nil {
		http.Error(w, "ERROR: Database cannot open", http.StatusBadRequest)
		return
	}
	defer db.Close()

	if !isAdmin(r, db) {
		http.Error(w, "Unauthorized access - Admins only", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case "GET":
		// Kullanıcıları listele
		users, err := requests.GetAllUsers(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)

	case "POST":
		// Kullanıcı rolünü değiştir
		userID := r.FormValue("user_id")
		if userID == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		var currentRole string
		err := db.QueryRow("SELECT role FROM USERS WHERE id = ?", userID).Scan(&currentRole)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		newRole := "user"
		if currentRole == "user" {
			newRole = "moderator"
		} else if currentRole == "moderator" {
			newRole = "user"
		}

		_, err = db.Exec("UPDATE USERS SET role = ? WHERE id = ?", newRole, userID)
		if err != nil {
			http.Error(w, "Failed to update user role", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"newRole": newRole})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func isAdmin(r *http.Request, db *sql.DB) bool {
	sessionToken, err := r.Cookie("session_token")
	if err != nil {
		return false
	}

	var role string
	err = db.QueryRow("SELECT role FROM USERS WHERE session_token = ?", sessionToken.Value).Scan(&role)
	if err != nil {
		return false
	}

	return role == "admin"
}
