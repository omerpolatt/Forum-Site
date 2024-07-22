package handlers

import (
	"net/http"

	admin "forum/backend/controllers/admiin"
	createcomment "forum/backend/controllers/create/createComment"
	createpost "forum/backend/controllers/create/createPost"
	deleteaccount "forum/backend/controllers/delete/deleteAccount"
	deletecomment "forum/backend/controllers/delete/deleteComment"
	deletepost "forum/backend/controllers/delete/deletePost"
	"forum/backend/controllers/facebook"
	getallposts "forum/backend/controllers/get/getAllPosts"
	getmycomments "forum/backend/controllers/get/getMyComments"
	getmyposts "forum/backend/controllers/get/getMyPosts"
	getmyvotedposts "forum/backend/controllers/get/getMyVotedPosts"
	getpostandcomments "forum/backend/controllers/get/getPostAndComments"
	getsearchedposts "forum/backend/controllers/get/getSearchedPosts"
	github "forum/backend/controllers/githubb"
	"forum/backend/controllers/google"
	"forum/backend/controllers/login"
	"forum/backend/controllers/logout"
	"forum/backend/controllers/register"
	downvote "forum/backend/controllers/votes/downVote"
	upvote "forum/backend/controllers/votes/upVote"
	adminpage "forum/frontend/pages/adminPage"
	createpostpage "forum/frontend/pages/createPostPage"
	deleteaccountpage "forum/frontend/pages/deleteAccountPage"
	loginpage "forum/frontend/pages/loginPage"
	mainpage "forum/frontend/pages/mainPage"
	postpage "forum/frontend/pages/postPage"
	mycommentspage "forum/frontend/pages/profile/myCommentsPage"
	mypostspage "forum/frontend/pages/profile/myPostsPage"
	myvotedpostspage "forum/frontend/pages/profile/myVotedPostsPage"
	registerpage "forum/frontend/pages/registerPage"
	searchedpostspage "forum/frontend/pages/searchedPostsPage"
)

func ImportHandlers() {
	// API
	http.HandleFunc("/api/register", register.Register)
	http.HandleFunc("/api/login", login.Login)
	http.HandleFunc("/api/logout", logout.Logout)
	http.HandleFunc("/api/createpost", createpost.CreatePost)
	http.HandleFunc("/api/createcomment", createcomment.CreateComment)
	http.HandleFunc("/api/deleteaccount", deleteaccount.DeleteAccount)
	http.HandleFunc("/api/deletepost", deletepost.DeletePost)
	http.HandleFunc("/api/deletecomment", deletecomment.DeleteComment)
	http.HandleFunc("/api/upvote", upvote.UpVote)
	http.HandleFunc("/api/downvote", downvote.DownVote)
	http.HandleFunc("/api/allposts", getallposts.GetAllPosts)
	http.HandleFunc("/api/postandcomments", getpostandcomments.GetPostAndComments)
	http.HandleFunc("/api/myposts", getmyposts.GetMyPosts)
	http.HandleFunc("/api/mycomments", getmycomments.GetMyComments)
	http.HandleFunc("/api/myvotedposts", getmyvotedposts.GetMyVotedPosts)
	http.HandleFunc("/api/searchedposts", getsearchedposts.GetSearchedPosts)

	google.LoadEnv()
	http.HandleFunc("/register/google", google.HandleGoogleRegister)
	http.HandleFunc("/callback/google", google.HandleGoogleCallback)
	http.HandleFunc("/login/google", google.HandleGoogleLogin)

	github.LoadEnv()
	http.HandleFunc("/github/login", github.GithubLoginHandler)
	http.HandleFunc("/github/register", github.GithubRegisterHandler)
	http.HandleFunc("/github/callback", github.GithubCallbackHandler)

	facebook.LoadEnv()
	http.HandleFunc("/register/facebook", facebook.HandleFacebookRegister)
	http.HandleFunc("/callback/facebook", facebook.HandleFacebookCallback)
	http.HandleFunc("/login/facebook", facebook.HandleFacebookLogin)

	http.HandleFunc("/api/admin", admin.Admin)

	// Front-end
	http.HandleFunc("/", mainpage.MainPage)
	http.HandleFunc("/register", registerpage.RegisterPage)
	http.HandleFunc("/login", loginpage.LoginPage)
	http.HandleFunc("/createpost", createpostpage.CreatePostPage)
	http.HandleFunc("/post", postpage.PostPage)
	http.HandleFunc("/createcomment", postpage.PostPageCreateComment)
	http.HandleFunc("/upvote", postpage.PostPageUpVote)
	http.HandleFunc("/downvote", postpage.PostPageDownVote)
	http.HandleFunc("/deleteaccount", deleteaccountpage.DeleteAccountPage)
	http.HandleFunc("/myposts", mypostspage.MyPostsPage)
	http.HandleFunc("/deletepost", mypostspage.DeleteMyPost)
	http.HandleFunc("/mycomments", mycommentspage.MyCommentsPage)
	http.HandleFunc("/deletecomment", mycommentspage.DeleteMyComment)
	http.HandleFunc("/myvotedposts", myvotedpostspage.MyVotedPostsPage)
	http.HandleFunc("/search", searchedpostspage.SearchedPostsPage)

	http.HandleFunc("/admin", adminpage.AdminPageHandler)

	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/frontend/", http.StripPrefix("/frontend/", fs))

	uploadsFs := http.FileServer(http.Dir("./imageuploads"))
	http.Handle("/imageuploads/", http.StripPrefix("/imageuploads", uploadsFs))
}
