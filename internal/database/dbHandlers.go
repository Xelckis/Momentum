package database

import (
	"Momentum/internal/jwt"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

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

func UserProfile(c *gin.Context) {
	id, _ := c.Get("userID")
	var user Users
	err := user.findUserByID(c, id.(string))
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	c.HTML(http.StatusOK, "editProfile.html", gin.H{
		"User": user,
	})

}

func EditPassword(c *gin.Context) {
	id, ok := c.Get("userID")
	if !ok {
		c.HTML(http.StatusOK, "changePasswordModal.html", gin.H{
			"Error": "An internal error occurred. Please try again.",
		})
		return
	}

	currentPassword := c.PostForm("current_password")
	newPassword := c.PostForm("new_password")
	confirmPassword := c.PostForm("confirm_password")

	if newPassword != confirmPassword {
		c.HTML(http.StatusOK, "changePasswordModal.html", gin.H{
			"Error": "The new passwords do not match.",
		})
		return
	}

	query := `SELECT password_hash FROM users WHERE id = $1`
	var passwordHash string
	err := conn.QueryRow(c.Request.Context(), query, id).Scan(&passwordHash)
	if err != nil {
		c.HTML(http.StatusOK, "changePasswordModal.html", gin.H{
			"Error": "An internal error occurred. Please try again.",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword)); err != nil {
		c.HTML(http.StatusOK, "changePasswordModal.html", gin.H{
			"Error": "The current password do not match.",
		})
		return
	}

	newPasswordHash, err := createPasswordHash(newPassword)
	if err != nil {
		c.HTML(http.StatusOK, "changePasswordModal.html", gin.H{
			"Error": "An internal error occurred. Please try again.",
		})
		return
	}

	query = `UPDATE users SET password_hash = $1 WHERE id = $2;`
	cmdTag, err := conn.Exec(c.Request.Context(), query, newPasswordHash, id)
	if err != nil {
		c.HTML(http.StatusOK, "changePasswordModal.html", gin.H{
			"Error": "Error saving changes.",
		})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		c.HTML(http.StatusOK, "changePasswordModal.html", gin.H{
			"Error": "User not found",
		})
		return
	}

	successFragment := `<div id="form-feedback" hx-swap-oob="true" class="success">Password updated successfully!</div>`

	closeModalFragment := `<div id="modal-placeholder" hx-swap-oob="true"></div>`

	finalHTML := successFragment + closeModalFragment
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(finalHTML))

}

func createPasswordHash(password string) ([]byte, error) {
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return []byte{}, fmt.Errorf("Internal server error")
	}
	return hashPassword, nil
}

func UserList(c *gin.Context) {
	var pagination Pagination
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	searchQuery := c.Query("q")

	query := `
        SELECT id, username, role, full_name, location_contact, work_phone, home_phone, created_at, updated_at 
        FROM users 
        WHERE id > $1 AND (username ILIKE '%' || $3 || '%' OR full_name ILIKE '%' || $3 || '%')
        ORDER BY id ASC 
        LIMIT $2;`

	rows, err := conn.Query(c.Request.Context(), query, pagination.After, pagination.Limit, searchQuery)
	if err != nil {
		log.Printf("failed to query items: %v", err)
		return
	}
	defer rows.Close()

	var users []Users
	for rows.Next() {
		var user Users
		if err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.FullName, &user.LocationContact, &user.WorkPhone, &user.HomePhone, &user.CreatedAt, &user.UpdatedAt); err != nil {
			log.Printf("failed to scan row: %v", err)
			return
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		log.Printf("error iterating rows: %v", err)
		c.String(http.StatusInternalServerError, "Erro ao ler lista de usuários.")
		return
	}

	var nextCursor int
	if len(users) > 0 {
		nextCursor = users[len(users)-1].ID
	}

	c.HTML(http.StatusOK, "userListRows.html", gin.H{
		"Users":       users,
		"NextCursor":  nextCursor,
		"SearchQuery": searchQuery,
	})

}

func EditUserDB(c *gin.Context) {
	id := c.Param("id")
	fullName := c.PostForm("full_name")
	role := c.PostForm("role")
	locationContact := c.PostForm("location_contact")
	workPhone := c.PostForm("work_phone")
	homePhone := c.PostForm("home_phone")

	query := `
		UPDATE users SET full_name = $1, role = $2, location_contact = $3, work_phone = $4, home_phone = $5  WHERE id = $6;
	`

	cmdTag, err := conn.Exec(c.Request.Context(), query, fullName, role, locationContact, workPhone, homePhone, id)
	if err != nil {
		log.Printf("Error editing user\n")
		return
	}

	if cmdTag.RowsAffected() == 0 {
		log.Println("User not found.")
		return
	}

}

func EditUserPage(c *gin.Context) {
	id := c.Param("id")

	var user Users
	err := user.findUserByID(c, id)
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	c.HTML(http.StatusOK, "editUser.html", gin.H{
		"User": user,
	})
}

func (u *Users) findUserByID(c *gin.Context, id string) error {
	query := `
        SELECT id, username, role, full_name, location_contact, work_phone, home_phone 
        FROM users 
        WHERE id = $1`

	err := conn.QueryRow(c.Request.Context(), query, id).Scan(&u.ID, &u.Username, &u.Role, &u.FullName, &u.LocationContact, &u.WorkPhone, &u.HomePhone)
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
		c.String(http.StatusBadRequest, "<div class='error'>All fields are required</div>")
		return
	}

	err := conn.QueryRow(c.Request.Context(), query, username).Scan(&user.Username, &user.PasswordHash, &user.Role, &user.ID)
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

	tokenString, err := jwt.CreateToken(username, user.Role, user.ID)
	if err != nil {
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
		c.Data(http.StatusUnprocessableEntity, "text/html; charset=utf-8", []byte(`<div id="form-feedback" class="error">Erro: Invalid role.</div>`))
		return
	}

	hashPassword, err := createPasswordHash(password)
	if err != nil {
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
			c.Data(http.StatusConflict, "text/html; charset=utf-8", []byte(`<div id="form-feedback" class="error">Erro: Nome de usuário já está em uso.</div>`))
			return
		}

		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(`<div id="form-feedback" class="error">Erro interno do servidor ao criar usuário.</div>`))
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
		c.String(http.StatusForbidden, "Você não pode excluir sua própria conta.")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Inavalid User ID.")
		return
	}

	query := "DELETE FROM users WHERE id = $1"
	cmdTag, err := conn.Exec(c.Request.Context(), query, id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}

	if cmdTag.RowsAffected() == 0 {
		c.String(http.StatusNotFound, "User not found.")
		return
	}

	c.Status(http.StatusOK)
}
