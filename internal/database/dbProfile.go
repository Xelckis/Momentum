package database

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

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
