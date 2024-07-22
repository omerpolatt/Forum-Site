package structs

type Post struct {
	ID        int    `json:"id"`
	UserID    int    `json:"userid"`
	UserName  string `json:"username"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	LikeCount int    `json:"likecount"`
	ImagePath string `json:"imagePath"`
}

type Comment struct {
	ID        int    `json:"id"`
	PostId    int    `json:"postid"`
	UserId    int    `json:"userid"`
	UserName  string `json:"username"`
	Comment   string `json:"comment"`
	LikeCount int    `json:"likecount"`
	ImagePath string `json:"imagePath"`
}

type PostWithComments struct {
	Post     Post
	Comments []Comment
}

type User struct {
	ID       int    `json:"id"`
	UserName string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}
