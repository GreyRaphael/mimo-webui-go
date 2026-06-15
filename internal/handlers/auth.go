package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mimo-webui/internal/auth"
	"mimo-webui/internal/config"
	"mimo-webui/internal/db"
)

// LoginPage renders the login page.
func LoginPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "pages/login.html", gin.H{
			"Title": "登录",
		})
	}
}

// RegisterPage renders the registration page.
func RegisterPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "pages/register.html", gin.H{
			"Title": "注册",
		})
	}
}

// LoginHandler authenticates the user and sets a JWT cookie.
func LoginHandler(database *db.DB, cfg config.AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")

		if username == "" || password == "" {
			c.HTML(http.StatusOK, "pages/login.html", gin.H{
				"Title": "登录",
				"Error": "用户名和密码不能为空",
			})
			return
		}

		user, err := db.GetUserByUsername(c.Request.Context(), database, username)
		if err != nil || user == nil {
			c.HTML(http.StatusOK, "pages/login.html", gin.H{
				"Title":    "登录",
				"Error":    "用户名或密码错误",
				"Username": username,
			})
			return
		}

		if err := auth.CheckPassword(user.PasswordHash, password); err != nil {
			c.HTML(http.StatusOK, "pages/login.html", gin.H{
				"Title":    "登录",
				"Error":    "用户名或密码错误",
				"Username": username,
			})
			return
		}

		token, err := auth.GenerateToken(cfg.JWTSecret, user.ID, user.Username, user.Role, cfg.JWTExpiryHours)
		if err != nil {
			c.HTML(http.StatusOK, "pages/login.html", gin.H{
				"Title":    "登录",
				"Error":    "登录失败，请稍后重试",
				"Username": username,
			})
			return
		}

		_ = db.UpdateLastLogin(c.Request.Context(), database, user.ID)

		c.SetCookie("token", token, cfg.JWTExpiryHours*3600, "/", "", false, true)
		c.Redirect(http.StatusFound, "/")
	}
}

// RegisterHandler creates a new user account.
func RegisterHandler(database *db.DB, cfg config.AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.OpenRegistration {
			c.HTML(http.StatusForbidden, "pages/register.html", gin.H{
				"Title": "注册",
				"Error": "注册功能已关闭",
			})
			return
		}

		username := c.PostForm("username")
		password := c.PostForm("password")

		if username == "" || password == "" {
			c.HTML(http.StatusOK, "pages/register.html", gin.H{
				"Title":    "注册",
				"Error":    "用户名和密码不能为空",
				"Username": username,
			})
			return
		}

		count, err := db.CountUsers(c.Request.Context(), database)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "pages/register.html", gin.H{
				"Title":    "注册",
				"Error":    "服务器错误，请稍后重试",
				"Username": username,
			})
			return
		}
		if count >= cfg.MaxUsers {
			c.HTML(http.StatusForbidden, "pages/register.html", gin.H{
				"Title":    "注册",
				"Error":    "已达到最大用户数，无法注册",
				"Username": username,
			})
			return
		}

		hash, err := auth.HashPassword(password)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "pages/register.html", gin.H{
				"Title":    "注册",
				"Error":    "服务器错误，请稍后重试",
				"Username": username,
			})
			return
		}

		role := "user"
		_, err = db.CreateUser(c.Request.Context(), database, username, hash, role)
		if err != nil {
			c.HTML(http.StatusOK, "pages/register.html", gin.H{
				"Title":    "注册",
				"Error":    "用户名已存在或注册失败",
				"Username": username,
			})
			return
		}

		c.Redirect(http.StatusFound, "/login")
	}
}
