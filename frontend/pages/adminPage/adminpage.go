package adminpage

import (
	"database/sql"
	"forum/backend/database"
	"forum/backend/requests"
	"html/template"
	"net/http"
)

// AdminPageHandler admin sayfasını render eder
func AdminPageHandler(w http.ResponseWriter, r *http.Request) {
	db, err := database.OpenDb(w) // Veritabanını aç
	if err != nil {
		http.Error(w, "Database cannot open", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Kullanıcı yetki kontrolü
	if isAdmin(r, db) {
		// Admin kullanıcı için admin sayfasını yükle
		tmpl, err := template.ParseFiles("frontend/pages/adminPage/adminPage.html")
		if err != nil {
			http.Error(w, "ERROR: Unable to load admin template", http.StatusInternalServerError)
			return
		}

		// Tüm kullanıcıları çek
		users, err := requests.GetAllUsers(db)
		if err != nil {
			http.Error(w, "Unable to fetch users", http.StatusInternalServerError)
			return
		}

		// Admin şablonunu kullanıcılara göre render et
		if err := tmpl.Execute(w, users); err != nil {
			http.Error(w, "Unable to execute admin template", http.StatusInternalServerError)
		}
	} else {
		// Kullanıcı admin değilse ana sayfaya yönlendir
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func isAdmin(r *http.Request, db *sql.DB) bool {
	sessionToken, err := r.Cookie("session_token")
	if err != nil {
		return false // Cookie bulunamadı, kullanıcı yetkili değil
	}

	var role string
	err = db.QueryRow("SELECT role FROM USERS WHERE session_token = ?", sessionToken.Value).Scan(&role)
	if err != nil {
		return false // Sorgu hatası veya kullanıcı bulunamadı
	}

	return role == "admin" // Role admin ise true, değilse false döner
}
