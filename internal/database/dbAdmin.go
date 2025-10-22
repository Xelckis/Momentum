package database

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

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
		c.String(http.StatusInternalServerError, "Error saving changes.")
		return
	}

	if cmdTag.RowsAffected() == 0 {
		c.String(http.StatusNotFound, "Error user not found")
		return
	}

	var updatedUser Users
	err = updatedUser.findUserByID(c, id)
	if err != nil {
		c.String(http.StatusNotFound, "User not found after changes.")
		return
	}

	c.Header("HX-Trigger", "closeModalEvent")

	c.HTML(http.StatusOK, "userListRows.html", gin.H{
		"Users": []Users{updatedUser},
	})

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
