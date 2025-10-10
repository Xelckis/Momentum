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

	/*LocationContact string
	WorkPhone       string
	HomePhone       string
	OtherInfo       string

	CustomFields map[string]any

	CreatedAt time.Time
	UpdatedAt time.Time*/
}

func UserList(c *gin.Context) {
	var pagination Pagination
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//query := "SELECT id, username, role, full_name, location_contact, work_phone, home_phone, other_info, custom_fields, created_at, updated_at FROM users WHERE id > $1 ORDER BY id ASC LIMIT $2;"

	searchQuery := c.Query("q")

	query := `
        SELECT id, username, role, full_name 
        FROM users 
        WHERE id > $1 AND (username ILIKE '%' || $3 || '%' OR full_name ILIKE '%' || $3 || '%')
        ORDER BY id ASC 
        LIMIT $2;`

	//rows, err := conn.Query(c.Request.Context(), query, pagination.After, pagination.Limit)
	rows, err := conn.Query(c.Request.Context(), query, pagination.After, pagination.Limit, searchQuery)
	if err != nil {
		log.Printf("failed to query items: %v", err)
		return
	}
	defer rows.Close()

	var users []Users
	for rows.Next() {
		var user Users
		//if err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.FullName, &user.LocationContact, &user.WorkPhone, &user.HomePhone, &user.OtherInfo, &user.CustomFields, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.FullName); err != nil {
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

	tokenString, err := jwt.CreateToken(username, user.Role)
	if err != nil {
		c.String(http.StatusBadRequest, "<div class='error'>Internal server error</div>")
		return
	}

	c.String(http.StatusOK, "<div class='success'>Logged in</div>")

	c.SetCookie("token", tokenString, 3600, "/", "localhost", false, true)
	c.Redirect(http.StatusSeeOther, "/")

}
