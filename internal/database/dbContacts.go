package database

import (
	"log"
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
		c.HTML(http.StatusOK, "addJobModal.html", gin.H{
			"Error": "An internal error occurred while searching contact. Please try again.",
		})

		log.Printf("Failed to fetch contacts: %v", err)
		return
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var contact Contact
		if err := rows.Scan(&contact.ID, &contact.Name, &contact.Email, &contact.Phone); err != nil {
			c.HTML(http.StatusOK, "addJobModal.html", gin.H{
				"Error": "An internal error occurred while searching contact. Please try again.",
			})

			log.Printf("failed to scan row: %v", err)
			return
		}
		contacts = append(contacts, contact)
	}

	if err := rows.Err(); err != nil {
		log.Printf("error iterating rows: %v", err)
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
		c.HTML(http.StatusUnprocessableEntity, "addContactModal.html", gin.H{
			"Error": "Failed to save contact. Please try again.",
		})
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", nil)
}
