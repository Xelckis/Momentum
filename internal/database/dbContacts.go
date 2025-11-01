package database

import (
	"Momentum/internal/logger"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Contact struct {
	ID        int
	Name      string
	Email     string
	Phone     string
	CreatedAt time.Time
}

func SearchContact(c *gin.Context) {
	contactQuery := c.Query("q_contact")
	if contactQuery == "" {
		return
	}

	searchValue := "%" + contactQuery + "%"
	query := `SELECT id, name, email, phone FROM contacts WHERE name ILIKE $1 OR email ILIKE $1 ORDER BY name LIMIT 10;`

	rows, err := conn.Query(c.Request.Context(), query, searchValue)
	if err != nil {
		logger.LogToLogFile(c, fmt.Sprintf("Search Contact [SQL]: Failed to fetch contacts: %v", err))
		c.HTML(http.StatusOK, "addJobModal.html", gin.H{
			"Error": "An internal error occurred while searching contact. Please try again.",
		})
		return
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var contact Contact
		if err := rows.Scan(&contact.ID, &contact.Name, &contact.Email, &contact.Phone); err != nil {
			logger.LogToLogFile(c, fmt.Sprintf("Search Contact [SQL Row Scan]: Failed to scan row: %v", err))
			c.HTML(http.StatusOK, "addJobModal.html", gin.H{
				"Error": "An internal error occurred while searching contact. Please try again.",
			})
			return
		}
		contacts = append(contacts, contact)
	}

	if err := rows.Err(); err != nil {
		logger.LogToLogFile(c, fmt.Sprintf("Search Contact [SQL Row Scan]: Error while iterating rows `%v`", err))
		c.HTML(http.StatusOK, "addJobModal.html", gin.H{
			"Error": "An internal error occurred while searching contact. Please try again.",
		})
		return
	}

	c.HTML(http.StatusOK, "_contactSearchResults.html", gin.H{
		"Contacts": contacts,
	})
}

func HandleNewContactModal(c *gin.Context) {
	c.HTML(http.StatusOK, "addContactModal.html", gin.H{
		"FormData": gin.H{
			"name":  "",
			"email": "",
			"phone": "",
		},
	})
}

func CreateContact(c *gin.Context) {
	name := c.PostForm("name")
	email := c.PostForm("email")
	phone := c.PostForm("phone")
	var newContact Contact

	query := `INSERT INTO contacts (name, email, phone) VALUES ($1, $2, $3) RETURNING id, name, email`
	err := conn.QueryRow(c.Request.Context(), query, name, email, phone).Scan(&newContact.ID, &newContact.Name, &newContact.Email)

	if err != nil {
		logger.LogToLogFile(c, fmt.Sprintf("Create Contact [SQL]: Error to create a new contact `%v`", err))
		c.HTML(http.StatusUnprocessableEntity, "addContactModal.html", gin.H{
			"Error": "Failed to save contact. Please try again.",
		})
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", nil)
}
