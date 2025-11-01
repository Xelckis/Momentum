package database

import (
	"Momentum/internal/logger"
	"errors"
	"fmt"
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
		logger.LogToLogFile(c, fmt.Sprintf("Delete Job Type [SQL]: Error while deleting from job_types table `%v`", err))
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
		logger.LogToLogFile(c, fmt.Sprintf("Get Job Type Handler: Error finding job {ID: %s} `%v`", id, err))
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
		logger.LogToLogFile(c, "Edit Job Type DB: Name is empty")
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
		logger.LogToLogFile(c, fmt.Sprintf("Edit Job Type DB [SQL]: Error while updating job_types table `%v`", err))
		c.Header("HX-Retarget", "#add-form-feedback")

		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "Error: Cannot save changes.",
		})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		logger.LogToLogFile(c, "Edit Job Type DB [SQL]: No rows were affected by the update")
		c.Header("HX-Retarget", "#add-form-feedback")

		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "Error: Job type not found",
		})
		return
	}

	var updatedJob jobType
	err = updatedJob.findJobTypeByID(c, id)
	if err != nil {
		logger.LogToLogFile(c, fmt.Sprintf("Edit Job Type DB: Job type not found after changes `%v`", err))
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
			logger.LogToLogFile(c, fmt.Sprintf("Job Type Edit Form: Job Type %s not found", id))
			c.Header("HX-Retarget", "#add-form-feedback")

			c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
				"Message": "Error: Job type not found.",
			})
			return
		}
		logger.LogToLogFile(c, fmt.Sprintf("Job Type Edit Form: Error while fetching job type info `%v`", err))
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
		logger.LogToLogFile(c, fmt.Sprintf("Job Type List [SQL]: Error while querying job_types table `%v`", err))
		c.Header("HX-Retarget", "#add-form-feedback")
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "An internal error occurred. Please try again.",
		})

		return
	}
	defer rows.Close()

	var jobTypes []jobType
	for rows.Next() {
		var job jobType
		if err := rows.Scan(&job.ID, &job.Name, &job.Description, &job.CreatedAt); err != nil {
			logger.LogToLogFile(c, fmt.Sprintf("Job Type List [SQL]: Error while scanning row `%v`", err))
			c.Header("HX-Retarget", "#add-form-feedback")
			c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
				"Message": "An internal error occurred. Please try again.",
			})
			return
		}
		jobTypes = append(jobTypes, job)

	}

	if err := rows.Err(); err != nil {
		logger.LogToLogFile(c, fmt.Sprintf("Job Type List [SQL]: Error while iterating rows `%v`", err))
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
		logger.LogToLogFile(c, "Create Job Type: Name is empty")
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
			logger.LogToLogFile(c, fmt.Sprintf("Create Job Type [SQL]: The `%s` job type already exists", name))
			c.Header("HX-Retarget", "#add-form-feedback")
			c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
				"Message": "Error: A Job Type with this name already exists.",
			})
			return
		}
		logger.LogToLogFile(c, fmt.Sprintf("Create Job Type [SQL]: Error while inserting `%s` job type `%v`", name, err))
		c.Header("HX-Retarget", "#add-form-feedback")
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "An internal error occurred. Please try again.",
		})
		return
	}

	c.HTML(http.StatusOK, "jobTypeItem.html", newJobType)
}
