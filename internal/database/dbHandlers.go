package database

import (
	"Momentum/internal/jwt"
	"errors"
	"fmt"
	"net/http"
	"time"

	"Momentum/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type userAuthData struct {
	Username     string
	PasswordHash string
	Role         string
	ID           string
}

type Pagination struct {
	Limit int `form:"limit,default=20"`
	After int `form:"after,default=0"` // Corresponds to the "cursor" ID
}

type jobType struct {
	ID                     int
	Name                   string
	Description            string
	CustomFieldDefinitions []CustomFieldDef `json:"-"`

	CreatedAt time.Time
}

type Users struct {
	ID       int
	Username string
	FullName string
	Role     string

	LocationContact string
	WorkPhone       string
	HomePhone       string
	OtherInfo       string

	CustomFields map[string]any

	CreatedAt time.Time
	UpdatedAt time.Time
}

func createPasswordHash(password string) ([]byte, error) {
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return []byte{}, fmt.Errorf("Internal server error")
	}
	return hashPassword, nil
}

func (u *Users) findUserByID(c *gin.Context, id string) error {
	query := `
        SELECT id, username, role, full_name, location_contact, work_phone, home_phone, created_at, updated_at 
        FROM users 
        WHERE id = $1`

	err := conn.QueryRow(c.Request.Context(), query, id).Scan(&u.ID, &u.Username, &u.Role, &u.FullName, &u.LocationContact, &u.WorkPhone, &u.HomePhone, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	return nil
}

func Login(c *gin.Context) {
	query := `SELECT username, password_hash,role, id FROM users WHERE username = $1`
	var user userAuthData
	username := c.PostForm("username")
	password := c.PostForm("password")
	if username == "" || password == "" {
		logger.LogToLogFile(c, "Login: All fields are required")
		c.String(http.StatusBadRequest, "<div class='error'>All fields are required</div>")
		return
	}

	err := conn.QueryRow(c.Request.Context(), query, username).Scan(&user.Username, &user.PasswordHash, &user.Role, &user.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			bcrypt.CompareHashAndPassword([]byte{}, []byte(password))
			logger.LogToLogFile(c, "Login: Wrong username")
			c.String(http.StatusBadRequest, "<div class='error'>Invalid username or password</div>")
			return
		}
		logger.LogToLogFile(c, fmt.Sprintf("Login [SQL]: Error while querying user username `%v`", err))
		c.String(http.StatusBadRequest, "<div class='error'>Internal server error</div>")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		logger.LogToLogFile(c, "Login: Wrong password")
		c.String(http.StatusBadRequest, "<div class='error'>Invalid username or password</div>")
		return
	}

	tokenString, err := jwt.CreateToken(username, user.Role, user.ID)
	if err != nil {
		logger.LogToLogFile(c, fmt.Sprintf("Login [JWT]: Error while creating JWT `%v`", err))
		c.String(http.StatusBadRequest, "<div class='error'>Internal server error (JWT)</div>")
		return
	}

	c.SetCookie("token", tokenString, 3600, "/", "localhost", false, true)
	c.Redirect(http.StatusSeeOther, "/")

}

func Register(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	role := c.PostForm("role")
	fullName := c.PostForm("full_name")
	locationContact := c.PostForm("location_contact")
	workPhone := c.PostForm("work_phone")
	homePhone := c.PostForm("home_phone")

	if role != "admin" && role != "user" {
		logger.LogToLogFile(c, "Register: The user's role is not admin or user.")
		c.Data(http.StatusUnprocessableEntity, "text/html; charset=utf-8", []byte(`<div id="form-feedback" class="error">Erro: Invalid role.</div>`))
		return
	}

	hashPassword, err := createPasswordHash(password)
	if err != nil {
		logger.LogToLogFile(c, fmt.Sprintf("Register [bcrypt]: Error while creating password hash `%v`", err))
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(`<div id="form-feedback" class="error">Erro: Internal server error</div>`))
		return
	}

	queryCreateUser := `INSERT INTO users (username, password_hash, role, full_name, location_contact, work_phone, home_phone) VALUES (@username, @passwordHash, @role, @fullName, @locationContact, @workPhone, @homePhone) RETURNING id`
	args := pgx.NamedArgs{
		"username":        username,
		"passwordHash":    hashPassword,
		"role":            role,
		"fullName":        fullName,
		"locationContact": locationContact,
		"workPhone":       workPhone,
		"homePhone":       homePhone,
	}

	var newID int
	err = conn.QueryRow(c.Request.Context(), queryCreateUser, args).Scan(&newID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "2350G" {
			logger.LogToLogFile(c, "Register: Username alredy in use")
			c.Data(http.StatusConflict, "text/html; charset=utf-8", []byte(`<div id="form-feedback" class="error">Error: Username is already in use.</div>`))
			return
		}
		logger.LogToLogFile(c, fmt.Sprintf("Register [SQL]: Error while inserting user into the database `%v`", err))
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(`<div id="form-feedback" class="error">Error: Internal Server Error, Try Again.</div>`))
		return
	}

	successHTML := `<div id="form-feedback" class="success">New user registered! ID: ` + fmt.Sprintf("%d", newID) + `</div>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(successHTML))
}

func DeleteUser(c *gin.Context) {
	idStr := c.Param("id")

	loggedInUserIDAny, _ := c.Get("userID")
	loggedInUserID, _ := loggedInUserIDAny.(string)
	if idStr == loggedInUserID {
		logger.LogToLogFile(c, "Delete User: You cannot delete your own account")
		c.String(http.StatusForbidden, "You cannot delete your own account")
		return
	}

	query := "DELETE FROM users WHERE id = $1"
	cmdTag, err := conn.Exec(c.Request.Context(), query, idStr)
	if err != nil {
		logger.LogToLogFile(c, fmt.Sprintf("Delete User [SQL]: Error while deleting user from database `%v`", err))
		c.String(http.StatusInternalServerError, "Internal Server Error, Try Again")
		return
	}

	if cmdTag.RowsAffected() == 0 {
		logger.LogToLogFile(c, "Delete User [SQL]: Error user not found")
		c.String(http.StatusNotFound, "User not found.")
		return
	}

	c.Status(http.StatusOK)
}
