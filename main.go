package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/srinathgs/mysqlstore"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type Question struct {
	ID      int    `json:"id,omitempty" form:"id,omitempty" db:"ID"`
	Content string `json:"content,omitempty" form:"content"  db:"Content"`
	Answer  string `json:"answer,omitempty" form:"answer"  db:"Answer"`
}

type Post struct {
	ID         int    `json:"id,omitempty"  db:"ID"`
	Content    string `json:"content,omitempty"  db:"Content"`
	QuestionID int    `json:"questionId,omitempty"  db:"QuestionID"`
}

type QuestionAnswer struct {
	Answer string `json:"answer,omitempty" form:"answer"  db:"Answer"`
}

var (
	db *sqlx.DB
)

func main() {
	_db, err := sqlx.Connect("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", os.Getenv("DB_USERNAME"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOSTNAME"), os.Getenv("DB_PORT"), os.Getenv("DB_DATABASE")))
	if err != nil {
		log.Fatalf("Cannot Connect to Database: %s", err)
	}
	db = _db

	store, err := mysqlstore.NewMySQLStoreFromConnection(db.DB, "sessions", "/", 60*60*24*14, []byte("secret-token"))
	if err != nil {
		panic(err)
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(session.Middleware(store))

	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})
	e.GET("/questions", getQuestions)
	e.GET("/question", getPosts)
	e.GET("/posts", getPosts)
	e.POST("/login", postLoginHandler)
	e.POST("/signup", postSignUpHandler)
	e.POST("/send_question", createQuestion)
	e.POST("/send_post", createPost)

	withLogin := e.Group("")
	withLogin.Use(checkLogin)
	withLogin.POST("/answer", postAnswer)
	withLogin.GET("/whoami", getWhoAmIHandler)

	e.Start(":4000")
}

type LoginRequestBody struct {
	Username string `json:"username,omitempty" form:"username"`
	Password string `json:"password,omitempty" form:"password"`
}

type User struct {
	Username   string `json:"username,omitempty"  db:"Username"`
	HashedPass string `json:"-"  db:"HashedPass"`
}

type Me struct {
	Username string `json:"username,omitempty"  db:"username"`
}

func getWhoAmIHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, Me{
		Username: c.Get("userName").(string),
	})
}

func postAnswer(c echo.Context) error {
	questionId := c.QueryParam("id")

	var questionAnswer QuestionAnswer
	if err := c.Bind(&questionAnswer); err != nil {
		return c.String(http.StatusBadRequest, "400 Bad Request")
	}

	var question Question
	err := db.Get(&question, "SELECT ID, Content, Answer FROM questions WHERE ID=?", questionId)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("(1)db error: %v", err))
	}

	_, err = db.Exec("UPDATE questions SET Answer=? WHERE ID=?", questionAnswer.Answer, questionId)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("(2)db error: %v", err))
	}

	var questions []Question
	questions = make([]Question, 0)
	rows, err := db.Query("SELECT ID, Content, Answer FROM questions")
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("(3)db error: %v", err))
	}

	for rows.Next() {
		var question Question
		if err = rows.Scan(&question.ID, &question.Content, &question.Answer); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("(4)db error: %v", err))
		}
		questions = append(questions, question)
	}

	return c.JSON(http.StatusCreated, questions)
}

func getPosts(c echo.Context) error {
	questionId := c.QueryParam("id")

	var content string
	err := db.Get(&content, "SELECT Content FROM questions WHERE ID=?", questionId)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
	}

	var posts []Post
	posts = make([]Post, 0)
	rows, err := db.Query("SELECT * FROM posts WHERE QuestionID=?", questionId)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
	}

	for rows.Next() {
		var post Post
		if err = rows.Scan(&post.ID, &post.Content, &post.QuestionID); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
		}
		posts = append(posts, post)
	}

	return c.JSON(http.StatusOK, struct {
		Content string `json:"content"`
		Posts   []Post `json:"posts"`
	}{
		Content: content,
		Posts:   posts,
	})
}

func getQuestions(c echo.Context) error {
	var questions []Question
	questions = make([]Question, 0)
	rows, err := db.Query("SELECT ID, Content, Answer FROM questions")
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
	}

	for rows.Next() {
		var question Question
		if err = rows.Scan(&question.ID, &question.Content, &question.Answer); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
		}
		questions = append(questions, question)
	}

	return c.JSON(http.StatusOK, questions)
}

func createQuestion(c echo.Context) error {
	var question Question

	if err := c.Bind(&question); err != nil {
		return c.String(http.StatusBadRequest, "400 Bad Request")
	}

	_, err := db.Exec("INSERT INTO questions (Content) VALUES (?)", question.Content)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
	}

	var questions []Question
	questions = make([]Question, 0)
	rows, err := db.Query("SELECT ID, Content, Answer FROM questions")
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
	}

	for rows.Next() {
		var question Question
		if err = rows.Scan(&question.ID, &question.Content, &question.Answer); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
		}
		questions = append(questions, question)
	}

	return c.JSON(http.StatusCreated, questions)
}

func createPost(c echo.Context) error {
	var post Post

	if err := c.Bind(&post); err != nil {
		return c.String(http.StatusBadRequest, "400 Bad Request")
	}

	_, err := db.Exec("INSERT INTO posts (Content, QuestionID) VALUES (?, ?)", post.Content, post.QuestionID)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
	}

	return c.NoContent(http.StatusCreated)
}

func postSignUpHandler(c echo.Context) error {
	req := LoginRequestBody{}
	c.Bind(&req)

	if req.Password == "" || req.Username == "" {
		return c.String(http.StatusBadRequest, "項目が空です")
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("bcrypt generate error: %v", err))
	}

	var count int

	err = db.Get(&count, "SELECT COUNT(*) FROM users WHERE Username=?", req.Username)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
	}

	if count > 0 {
		return c.String(http.StatusConflict, "ユーザーが既に存在しています")
	}

	_, err = db.Exec("INSERT INTO users (Username, HashedPass) VALUES (?, ?)", req.Username, hashedPass)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
	}
	return c.NoContent(http.StatusCreated)
}

func postLoginHandler(c echo.Context) error {
	req := LoginRequestBody{}
	c.Bind(&req)

	user := User{}
	err := db.Get(&user, "SELECT * FROM users WHERE username=?", req.Username)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPass), []byte(req.Password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return c.NoContent(http.StatusForbidden)
		} else {
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	sess, err := session.Get("sessions", c)
	if err != nil {
		fmt.Println(err)
		return c.String(http.StatusInternalServerError, "something wrong in getting session")
	}
	sess.Values["userName"] = req.Username
	sess.Save(c.Request(), c.Response())

	return c.NoContent(http.StatusOK)
}

func checkLogin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get("sessions", c)
		if err != nil {
			fmt.Println(err)
			return c.String(http.StatusInternalServerError, "something wrong in getting session")
		}

		if sess.Values["userName"] == nil {
			return c.String(http.StatusForbidden, "please login")
		}
		c.Set("userName", sess.Values["userName"].(string))

		return next(c)
	}
}
