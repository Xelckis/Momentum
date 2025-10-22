package database

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func getJobTypeName(c *gin.Context, jobTypeId string) (string, error) {
	var jobTypeName string
	query := `SELECT name FROM job_types WHERE id = $1`
	err := conn.QueryRow(c.Request.Context(), query, jobTypeId).Scan(&jobTypeName)
	if err != nil {
		return "", fmt.Errorf("ERROR Job type not found: %v", err)
	}

	return jobTypeName, nil
}

func DeleteJobType(c *gin.Context) {
	id := c.Param("id")

	query := "DELETE FROM job_types WHERE id = $1"
	cmdTag, err := conn.Exec(c.Request.Context(), query, id)
	if err != nil {
		c.Header("HX-Retarget", "#add-form-feedback")
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "Error: An internal error occurred. Please try again.",
		})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		c.Header("HX-Retarget", "#add-form-feedback")
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "Error: User not found.",
		})
		return
	}

	c.Status(http.StatusOK)
}

func GetJobTypeHandler(c *gin.Context) {
	id := c.Param("id")

	var job jobType
	err := job.findJobTypeByID(c, id)

	if err != nil {
		c.Header("HX-Retarget", "#add-form-feedback")

		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "Error: Job Type not found.",
		})
		return
	}

	c.HTML(http.StatusOK, "jobTypeItem.html", job)
}

func EditJobTypeDB(c *gin.Context) {
	id := c.Param("id")
	name := c.PostForm("name")
	description := c.PostForm("description")

	if name == "" {
		c.Header("HX-Retarget", "#add-form-feedback")

		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "Error: Name cannot be empty",
		})
		return
	}

	query := `
		UPDATE job_types SET name = $1, description = $2 WHERE id = $3;
	`

	cmdTag, err := conn.Exec(c.Request.Context(), query, name, description, id)
	if err != nil {
		c.Header("HX-Retarget", "#add-form-feedback")

		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "Error: Cannot save changes.",
		})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		c.Header("HX-Retarget", "#add-form-feedback")

		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "Error: Job type not found",
		})
		return
	}

	var updatedJob jobType
	err = updatedJob.findJobTypeByID(c, id)
	if err != nil {
		c.Header("HX-Retarget", "#add-form-feedback")

		c.HTML(http.StatusOK, "errorFe1022d1e9edback.html", gin.H{
			"Message": "Error: Job type not found after changes.",
		})
		return
	}

	c.HTML(http.StatusOK, "jobTypeItem.html", updatedJob)

}

func JobTypeEditForm(c *gin.Context) {
	id := c.Param("id")
	var job jobType

	err := job.findJobTypeByID(c, id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Header("HX-Retarget", "#add-form-feedback")

			c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
				"Message": "Error: Job type not found.",
			})
			return
		}
		log.Printf("Error fetching job type: %v", err)
		c.Header("HX-Retarget", "#add-form-feedback")

		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "Error: Job type not found.",
		})

		return
	}

	c.HTML(http.StatusOK, "jobTypeEditForm.html", job)

}

func (j *jobType) findJobTypeByID(c *gin.Context, id string) error {
	query := `SELECT id, name, description, created_at FROM job_types WHERE id = $1`

	err := conn.QueryRow(c.Request.Context(), query, id).Scan(&j.ID, &j.Name, &j.Description, &j.CreatedAt)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	return nil
}

func JobTypeList(c *gin.Context) {
	query := "SELECT id, name, description, created_at FROM job_types"

	rows, err := conn.Query(c.Request.Context(), query)
	if err != nil {
		c.Header("HX-Retarget", "#add-form-feedback")
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "An internal error occurred. Please try again.",
		})

		log.Printf("Failed to fetch job types: %v", err)
		return
	}
	defer rows.Close()

	var jobTypes []jobType
	for rows.Next() {
		var job jobType
		if err := rows.Scan(&job.ID, &job.Name, &job.Description, &job.CreatedAt); err != nil {
			c.Header("HX-Retarget", "#add-form-feedback")
			c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
				"Message": "An internal error occurred. Please try again.",
			})

			log.Printf("failed to scan row: %v", err)
			return
		}
		jobTypes = append(jobTypes, job)

	}

	if err := rows.Err(); err != nil {
		log.Printf("error iterating rows: %v", err)
		c.Header("HX-Retarget", "#add-form-feedback")
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "An internal error occurred. Please try again.",
		})

		return
	}
	c.HTML(http.StatusOK, "manageJobTypes.html", gin.H{
		"JobTypes": jobTypes,
	})
}

func CreateJobType(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")

	if name == "" {
		c.Header("HX-Retarget", "#add-form-feedback")

		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "Error: Name cannot be empty",
		})
		return
	}

	query := "INSERT INTO job_types (name, description) VALUES ($1, $2) RETURNING id, name, description, created_at"
	var newJobType jobType

	err := conn.QueryRow(c.Request.Context(), query, name, description).Scan(&newJobType.ID, &newJobType.Name, &newJobType.Description, &newJobType.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.Header("HX-Retarget", "#add-form-feedback")
			c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
				"Message": "Error: A Job Type with this name already exists.",
			})
			return
		}

		c.Header("HX-Retarget", "#add-form-feedback")
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "An internal error occurred. Please try again.",
		})
		return
	}

	c.HTML(http.StatusOK, "jobTypeItem.html", newJobType)
}
