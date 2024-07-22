package database

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

const (
	databasePath  = "./forum.db"
	adminEmail    = "admin@gmail.com"
	adminUsername = "admin"
	adminPassword = "admin123" // Not: Gerçek bir uygulamada şifreyi hashleyin.
)

func OpenDb(w http.ResponseWriter) (*sql.DB, error) {
	db, errDb := sql.Open("sqlite3", databasePath)
	if errDb != nil {
		return nil, errDb
	}
	return db, errDb
}

func CreateDatabaseIfNotExists() {
	db, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Tabloları oluşturun
	createTables := `
    CREATE TABLE IF NOT EXISTS USERS (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        Email TEXT NOT NULL UNIQUE,
        UserName TEXT NOT NULL,
        Password TEXT NOT NULL,
        Role TEXT,
        session_token TEXT
    );
    CREATE TABLE IF NOT EXISTS POSTS (
        ID INTEGER UNIQUE,
        UserID INTEGER,
        UserName TEXT,
        Title TEXT,
        Content TEXT,
        LikeCount INTEGER DEFAULT 0,
        PostDate TEXT NOT NULL DEFAULT (datetime('now')),
        PRIMARY KEY(ID AUTOINCREMENT)
    );
    CREATE TABLE IF NOT EXISTS COMMENTS (
        ID INTEGER PRIMARY KEY AUTOINCREMENT,
        PostId INTEGER,
        UserId INTEGER,
        UserName TEXT,
        Comment TEXT NOT NULL,
        LikeCount INTEGER DEFAULT 0,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY(PostId) REFERENCES posts(ID),
        FOREIGN KEY(UserId) REFERENCES users(ID)
    );
    CREATE TABLE IF NOT EXISTS USERLIKES (
        ID INTEGER PRIMARY KEY AUTOINCREMENT,
        UserID INTEGER,
        PostID INTEGER,
        IsComment BOOLEAN,
        DeleteID INTEGER,
        Liked BOOLEAN,
        Disliked BOOLEAN,
        UNIQUE(UserID, PostID, IsComment)
    );
    CREATE TABLE IF NOT EXISTS CATEGORIES (
        ID INTEGER PRIMARY KEY AUTOINCREMENT,
        USERID INTEGER,
        PostID INTEGER,
        GO INTEGER DEFAULT 0 CHECK(GO IN (0, 1)),
        HTML INTEGER DEFAULT 0 CHECK(HTML IN (0, 1)),
        CSS INTEGER DEFAULT 0 CHECK(CSS IN (0, 1)),
        PHP INTEGER DEFAULT 0 CHECK(PHP IN (0, 1)),
        PYTHON INTEGER DEFAULT 0 CHECK(PYTHON IN (0, 1)),
        C INTEGER DEFAULT 0 CHECK(C IN (0, 1)),
        CPP INTEGER DEFAULT 0 CHECK(CPP IN (0, 1)),
        CSHARP INTEGER DEFAULT 0 CHECK(CSHARP IN (0, 1)),
        JS INTEGER DEFAULT 0 CHECK(JS IN (0, 1)),
        ASSEMBLY INTEGER DEFAULT 0 CHECK(ASSEMBLY IN (0, 1)),
        REACT INTEGER DEFAULT 0 CHECK(REACT IN (0, 1)),
        FLUTTER INTEGER DEFAULT 0 CHECK(FLUTTER IN (0, 1)),
        RUST INTEGER DEFAULT 0 CHECK(RUST IN (0, 1)),
        FOREIGN KEY(PostID) REFERENCES POSTS(ID),
        FOREIGN KEY(USERID) REFERENCES USERS(ID)
    );`

	_, err = db.Exec(createTables)
	if err != nil {
		log.Fatal(err)
	}

	// Admin kullanıcısının var olup olmadığını kontrol edin
	var id int
	err = db.QueryRow("SELECT id FROM USERS WHERE Email = ?", adminEmail).Scan(&id)
	if err == sql.ErrNoRows {
		// Admin kullanıcısı yok, oluşturun
		createAdminUser(db)
	} else if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Admin kullanıcı zaten mevcut.")
	}
}

func createAdminUser(db *sql.DB) {

	hashedPassword, errHash := HashThePasswd(adminPassword)
	if errHash != nil {
		log.Fatalf("Admin şifre hashleme hatası: %v", errHash)
		return
	}

	insertAdmin := `INSERT INTO USERS (Email, UserName, Password, Role) VALUES (?, ?, ?, 'admin');`
	_, err := db.Exec(insertAdmin, adminEmail, adminUsername, hashedPassword)
	if err != nil {
		log.Fatalf("Admin kullanıcı oluşturma hatası: %v", err)
	}
	fmt.Println("Admin kullanıcı başarıyla oluşturuldu.")
}

func HashThePasswd(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}
