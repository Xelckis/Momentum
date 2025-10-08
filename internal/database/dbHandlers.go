package database

import (
	"Momentum/internal/jwt"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type userAuthData struct {
	Username     string
	PasswordHash string
	Role         string
}

func Login(c *gin.Context) {
	query := `SELECT username, password_hash,role FROM users WHERE username = $1`
	var user userAuthData
	username := c.PostForm("username")
	password := c.PostForm("password")
	if username == "" || password == "" {
		c.String(http.StatusBadRequest, "<div class='error'>All fields are required</div>")
		return
	}

	err := conn.QueryRow(c.Request.Context(), query, username).Scan(&user.Username, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			bcrypt.CompareHashAndPassword([]byte{}, []byte(password))
			c.String(http.StatusBadRequest, "<div class='error'>Invalid username or password</div>")
			return
		}
		c.String(http.StatusBadRequest, "<div class='error'>Internal server error</div>")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		c.String(http.StatusBadRequest, "<div class='error'>Invalid username or password</div>")
		return
	}

	tokenString, err := jwt.CreateToken(username, user.Role)
	if err != nil {
		c.String(http.StatusBadRequest, "<div class='error'>Internal server error</div>")
		return
	}

	c.String(http.StatusOK, "<div class='success'>Logged in</div>")

	c.SetCookie("token", tokenString, 3600, "/", "localhost", false, true)
	c.Redirect(http.StatusSeeOther, "/")

}
