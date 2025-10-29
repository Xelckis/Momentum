package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type JobUpdate struct {
	ID           string
	AuthorUserID int
	AuthorName   string
	CreatedAt    time.Time

	Content map[string]any
}

type Job struct {
	ID        int
	Status    string
	Ticket    string
	Title     string
	JobTypeID int

	CustomFields map[string]any
	LastUpdate   *JobUpdate
}

type PaginationUpdates struct {
	Limit  int    `form:"limit,default=10"`
	Before string `form:"before,default=now"`
}

func Jobs(c *gin.Context) {
	jobTypeId := c.Param("jobTypeId")

	jobTypeName, err := getJobTypeName(c, jobTypeId)
	if err != nil {
		c.String(http.StatusNotFound, "Job type not found")
		return
	}

	c.HTML(http.StatusOK, "jobList.html", gin.H{
		"JobTypeName": jobTypeName,
		"JobTypeId":   jobTypeId,
	})
}

func NewJobModal(c *gin.Context) {
	jobTypeId := c.Param("id")
	jobTypeName, err := getJobTypeName(c, jobTypeId)
	if err != nil {
		c.Header("HX-Retarget", "#modal-placeholder")
		c.HTML(http.StatusOK, "addJobModal.html", gin.H{
			"Error": "Error: An internal error occurred. Please try again.",
		})
		return
	}

	var customFields CustomFieldDefList
	customFields.fetchCurrentCustomFields(c, jobTypeId)

	formData := gin.H{
		"title":              "",
		"primary_contact_id": "",
		"custom_fields":      make(map[string]string),
	}

	c.HTML(http.StatusOK, "addJobModal.html", gin.H{
		"JobTypeName":     jobTypeName,
		"CustomFieldDefs": customFields,
		"FormData":        formData,
		"JobTypeId":       jobTypeId,
	})

}

func AddNewJob(c *gin.Context) {
	var err error
	var ticket string

	loggedInUserID, ok := c.Get("userID")
	if !ok {
		c.HTML(http.StatusOK, "addJobModal.html", gin.H{
			"Error": "Failed to get userID",
		})
		return

	}
	jobTypeId := c.Param("jobTypeId")
	title := c.PostForm("title")

	contactID := c.PostForm("primary_contact_id")

	customFields := c.PostFormMap("custom_fields")

	file, err := c.FormFile("thumbnail_image")

	if err != nil {
		if err == http.ErrMissingFile {
			c.HTML(http.StatusOK, "addJobModal.html", gin.H{
				"Error": "A thumbail is must",
			})
			return
		} else {
			c.HTML(http.StatusOK, "addJobModal.html", gin.H{
				"Error": "Error processing uploaded file.",
			})
			return
		}
	}

	maxAttempts := 10
	foundUnique := false

	for i := 0; i < maxAttempts; i++ {
		ticket, err = generateTicketID()
		if err != nil {
			c.HTML(http.StatusOK, "addJobModal.html", gin.H{
				"Error": "Error creating job ticket (generation failed).",
			})
			return
		}

		exist, checkErr := checkTicketExists(c, ticket)
		if checkErr != nil {
			c.HTML(http.StatusOK, "addJobModal.html", gin.H{
				"Error": "Error verifying job ticket (database error).",
			})
			return
		}

		if !exist {
			foundUnique = true
			break
		}

		log.Printf("Ticket ID collision detected: %s. Retrying... (Attempt %d/%d)", ticket, i+1, maxAttempts)
	}

	if !foundUnique {
		log.Printf("Failed to generate unique ticket ID after %d attempts.", maxAttempts)
		c.HTML(http.StatusOK, "addJobModal.html", gin.H{
			"Error": "Could not generate a unique ticket ID. Please try again later.",
		})
		return
	}

	if file != nil {
		var allowedImageTypes = map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".webp": true,
		}
		ext := strings.ToLower(filepath.Ext(file.Filename))

		if !allowedImageTypes[ext] {
			log.Printf("Tipo de arquivo inválido recebido: %s", ext)

			c.HTML(http.StatusOK, "addJobModal.html", gin.H{
				"Error": "Invalid file type. Only .jpg, .jpeg, .png, .webp are allowed.",
			})
			return
		}

		newFileName := ticket + ext

		destinationPath := filepath.Join("./uploads/thumbnails/", newFileName)

		src, err := file.Open()
		if err != nil {
			log.Printf("Erro ao abrir arquivo enviado: %v", err)
			return
		}
		defer src.Close()

		dst, err := os.Create(destinationPath)
		if err != nil {
			log.Printf("Erro ao criar arquivo de destino: %v", err)
			return
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			log.Printf("Erro ao salvar arquivo: %v", err)
			return
		}

		customFields["thumbnail_url"] = "/uploads/thumbnails/" + newFileName
	}

	query := `INSERT INTO jobs (ticket_id, title, job_type_id, primary_contact_id, assigned_to_user_id, custom_fields) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`

	var jobId string
	err = conn.QueryRow(c.Request.Context(), query, ticket, title, jobTypeId, contactID, loggedInUserID, customFields).Scan(&jobId)
	if err != nil {
		c.HTML(http.StatusOK, "addContactModal.html", gin.H{
			"Error": "Failed to save contact. Please try again.",
		})
		return
	}

	jsonDataMap := gin.H{
		"title": "This job was added",
	}

	jobUpdateMessage, err := json.Marshal(jsonDataMap)
	if err != nil {
		c.HTML(http.StatusOK, "addContactModal.html", gin.H{
			"Error": "An internal error occurred while creating the job. Please try again.",
		})
		return
	}

	jobUpdateQuery := `INSERT INTO job_updates (job_id, author_user_id, content) VALUES ($1, $2, $3)`
	_, err = conn.Exec(c.Request.Context(), jobUpdateQuery, jobId, loggedInUserID, jobUpdateMessage)
	if err != nil {
		c.HTML(http.StatusOK, "addJobModal.html", gin.H{
			"Error": "Failed to save the initial job note. The job was created, but the note was not. Please add the note manually.",
		})
		return
	}
	successHTML := `
        <h3>Success!</h3>
        <p>The job "<strong>` + title + `</strong>" was created successfully.</p>
        <hr>
        <button type="button" 
                onclick="document.getElementById('modal-placeholder').innerHTML = ''">
            Close
        </button>
    `
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(successHTML))

}

func checkTicketExists(ctx *gin.Context, ticketID string) (bool, error) {
	var exists bool

	query := `SELECT EXISTS(SELECT 1 FROM jobs WHERE ticket_id = $1)`

	err := conn.QueryRow(ctx.Request.Context(), query, ticketID).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("failed to verify ticket in DB: %w", err)
	}

	return exists, nil
}

func DeleteJob(c *gin.Context) {
	jobID := c.Param("id")
	var assignedToUserId string
	query := `SELECT assigned_to_user_id FROM jobs WHERE id = $1`
	err := conn.QueryRow(c.Request.Context(), query, jobID).Scan(&assignedToUserId)
	if err != nil {
		c.Header("HX-Retarget", "#global-notification-placeholder")
		c.Header("HX-Reswap", "innerHTML")
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "An internal error occurred while creating the job. Please try again.",
		})
		return

	}

	loggedInUserID, ok := c.Get("userID")
	if !ok {
		c.Header("HX-Retarget", "#global-notification-placeholder")
		c.Header("HX-Reswap", "innerHTML")
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "An internal error occurred while creating the job. Please try again.",
		})
		return

	}

	loggedInUserRole, ok := c.Get("role")
	if !ok {
		c.Header("HX-Retarget", "#global-notification-placeholder")
		c.Header("HX-Reswap", "innerHTML")
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "An internal error occurred while creating the job. Please try again.",
		})
		return

	}

	if loggedInUserID.(string) != assignedToUserId && loggedInUserRole != "admin" {
		c.Header("HX-Retarget", "#global-notification-placeholder")
		c.Header("HX-Reswap", "innerHTML")
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "You do not have permission to delete this job.",
		})
		return

	}

	deleteQuery := `DELETE FROM jobs WHERE id = $1`
	_, err = conn.Exec(c.Request.Context(), deleteQuery, jobID)
	if err != nil {
		c.Header("HX-Retarget", "#global-notification-placeholder")
		c.Header("HX-Reswap", "innerHTML")
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "Failed to delete job. Please try again.",
		})
		return

	}

	c.Status(http.StatusOK)

}

func JobsList(c *gin.Context) {
	jobTypeId := c.Param("id")
	var pagination Pagination
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pagination.Limit <= 0 || pagination.Limit > 50 {
		pagination.Limit = 10
	}

	query := `
	    WITH latest_updates AS (
	        SELECT
	        ranked_updates.id,
	            ranked_updates.author_user_id,
	            ranked_updates.content,
	            ranked_updates.created_at,
	            ranked_updates.job_id,
	            u.username AS author_name
	        FROM (
	            SELECT
	                id, author_user_id, content, created_at, job_id,
	                ROW_NUMBER() OVER(PARTITION BY job_id ORDER BY created_at DESC) as rn
	            FROM
	                job_updates
	        ) ranked_updates
	        LEFT JOIN users u ON ranked_updates.author_user_id = u.id
	        WHERE
	            ranked_updates.rn = 1
	    )
	    SELECT
	        j.id, j.title, j.status, j.ticket_id, j.custom_fields, -- Campos da 'jobs'

	        lu.id AS last_update_id,
	        lu.author_user_id AS last_update_author_id,
	        lu.content AS last_update_content,
	        lu.created_at AS last_update_created_at,
	        lu.author_name AS last_update_author_name

	    FROM
	        jobs j
	    LEFT JOIN
	        latest_updates lu ON j.id = lu.job_id
	    WHERE
	        j.id > $1 AND j.job_type_id = $2
	    ORDER BY
	        j.id ASC
	    LIMIT $3;`

	rows, err := conn.Query(c.Request.Context(), query, pagination.After, jobTypeId, pagination.Limit)
	if err != nil {
		log.Printf("failed to query items: %v", err)
		return
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var job Job
		var lastUpdateID sql.NullString
		var lastUpdateAuthorID sql.NullInt64
		var lastUpdateAuthorName sql.NullString
		var lastUpdateContent map[string]any
		var lastUpdateCreatedAt sql.NullTime

		if err := rows.Scan(&job.ID, &job.Title, &job.Status, &job.Ticket, &job.CustomFields, &lastUpdateID, &lastUpdateAuthorID, &lastUpdateContent, &lastUpdateCreatedAt, &lastUpdateAuthorName); err != nil {
			log.Printf("failed to scan row: %v", err)
			return
		}

		if lastUpdateID.Valid {
			job.LastUpdate = &JobUpdate{
				ID:           lastUpdateID.String,
				AuthorUserID: int(lastUpdateAuthorID.Int64),
				Content:      lastUpdateContent,
				CreatedAt:    lastUpdateCreatedAt.Time,
				AuthorName:   lastUpdateAuthorName.String,
			}
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		log.Printf("error iterating rows: %v", err)
		c.String(http.StatusInternalServerError, "Error reading jobs list.")
		return
	}

	var nextCursor int
	if len(jobs) > 0 {
		nextCursor = jobs[len(jobs)-1].ID
	}

	c.HTML(http.StatusOK, "jobCardFragment.html", gin.H{
		"Jobs":       jobs,
		"NextCursor": nextCursor,
		"JobTypeId":  jobTypeId,
	})
}

func generateTicketID() (string, error) {
	now := time.Now()

	datePrefix := now.Format("T0102")

	numBytes := 2
	randomBytes := make([]byte, numBytes)

	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Printf("Critical error generating random bytes:: %v", err)
		return "", fmt.Errorf("failed to generate random part of ticket: %w", err)
	}

	hexSuffix := fmt.Sprintf("%x", randomBytes)

	ticketID := datePrefix + hexSuffix

	return ticketID, nil
}

func EditJobModal(c *gin.Context) {
	jobID := c.Param("id")
	var jobData struct {
		ID                  int
		Title               string
		Status              string
		Ticket              string
		JobTypeID           string
		JobTypeName         string
		PrimaryContactID    sql.NullInt64
		CustomFields        []byte
		SelectedContactName sql.NullString
	}

	query := `
        SELECT 
            j.id, j.title, j.status, j.ticket_id, j.job_type_id, 
            jt.name AS job_type_name, 
            j.primary_contact_id, 
            j.custom_fields, 
            COALESCE(c.name, '') AS selected_contact_name
        FROM 
            jobs j
        JOIN 
            job_types jt ON j.job_type_id = jt.id
        LEFT JOIN 
            contacts c ON j.primary_contact_id = c.id
        WHERE 
            j.id = $1`

	err := conn.QueryRow(c.Request.Context(), query, jobID).Scan(
		&jobData.ID,
		&jobData.Title,
		&jobData.Status,
		&jobData.Ticket,
		&jobData.JobTypeID,
		&jobData.JobTypeName,
		&jobData.PrimaryContactID,
		&jobData.CustomFields,
		&jobData.SelectedContactName,
	)

	if err != nil {
		c.String(http.StatusNotFound, "Job not found.")
		return
	}

	var customFieldDefs CustomFieldDefList
	customFieldDefs.fetchCurrentCustomFields(c, jobData.JobTypeID)

	var customFieldsMap map[string]any
	if err := json.Unmarshal(jobData.CustomFields, &customFieldsMap); err != nil {
		c.String(http.StatusInternalServerError, "An internal error occurred while creating the job. Please try again.")
		return
	}

	formData := gin.H{
		"title":              jobData.Title,
		"primary_contact_id": jobData.PrimaryContactID.Int64,
		"status":             jobData.Status,
		"custom_fields":      customFieldsMap,
	}

	jobPayload := gin.H{
		"ID":           jobData.ID,
		"Title":        jobData.Title,
		"Ticket":       jobData.Ticket,
		"JobTypeName":  jobData.JobTypeName,
		"CustomFields": customFieldsMap,
	}

	c.HTML(http.StatusOK, "editJobModal.html", gin.H{
		"Job":                 jobPayload,
		"CustomFieldDefs":     customFieldDefs,
		"FormData":            formData,
		"SelectedContactName": jobData.SelectedContactName.String,
		"Error":               nil,
	})
}

func EditJob(c *gin.Context) {
	imgUUID := uuid.New()
	loggedInUserID, ok := c.Get("userID")
	if !ok {
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "An internal error occurred while editing the job. Please try again.",
		})
		return
	}

	jobID := c.Param("id")
	jobTitle := c.PostForm("title")
	jobStatus := c.PostForm("status")
	jobThumbnail, err := c.FormFile("thumbnail_image")
	primaryContactID := c.PostForm("primary_contact_id")
	customFields := c.PostFormMap("custom_fields")
	jobUpdateTitle := c.PostForm("update_title")
	jobUpdateDescription := c.PostForm("update_description")

	if jobTitle == "" || primaryContactID == "" || jobUpdateTitle == "" {
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "Title, Contanct and Update Title is must.",
		})
		return
	}

	if err != nil {
		if err != http.ErrMissingFile {
			c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
				"Message": "Error processing uploaded file.",
			})
			return
		}
	}

	if jobThumbnail != nil {
		var allowedImageTypes = map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".webp": true,
		}
		ext := strings.ToLower(filepath.Ext(jobThumbnail.Filename))

		if !allowedImageTypes[ext] {
			c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
				"Message": "Invalid file type. Only .jpg, .jpeg, .png, .webp are allowed.",
			})
			return
		}

		newFileName := imgUUID.String() + ext

		destinationPath := filepath.Join("./uploads/thumbnails/", newFileName)

		if err := c.SaveUploadedFile(jobThumbnail, destinationPath); err != nil {
			c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
				"Message": "Error saving the new thumbanil",
			})
			return
		}

		customFields["thumbnail_url"] = "/uploads/thumbnails/" + newFileName
	}

	contentMap := gin.H{
		"title":       jobUpdateTitle,
		"description": jobUpdateDescription,
	}

	jobUpdateContent, err := json.Marshal(contentMap)
	if err != nil {
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "Error processing the job update.",
		})
		return
	}

	query := `
	WITH updated_job AS (
	        UPDATE jobs
	        SET 
	            title = $2,
		    status = $3,
	            primary_contact_id = $4,
	            custom_fields = custom_fields || $5
	        WHERE 
	            id = $1
	    )
	    INSERT INTO job_updates (job_id, author_user_id, content)
	    VALUES ($1, $6, $7); -- Inserção direta
	`

	_, err = conn.Exec(c.Request.Context(), query, jobID, jobTitle, jobStatus, primaryContactID, customFields, loggedInUserID, jobUpdateContent)
	if err != nil {
		log.Println(err)
		c.HTML(http.StatusOK, "errorFeedback.html", gin.H{
			"Message": "An internal error occurred while editing the job. Please try again.",
		})
		return
	}

	c.Header("HX-Redirect", "/jobs/"+jobID)
	c.Status(http.StatusOK)
}

func NewJobUpdateModal(c *gin.Context) {
	jobID := c.Param("id")
	var job Job
	query := `SELECT id, ticket_id, title FROM jobs WHERE id = $1;`

	err := conn.QueryRow(c.Request.Context(), query, jobID).Scan(&job.ID, &job.Ticket, &job.Title)
	if err != nil {
		c.String(http.StatusNotFound, "Job not found")
		return
	}

	c.HTML(http.StatusOK, "newJobUpdate.html", gin.H{
		"Job": job,
		"FormData": gin.H{
			"update_title":       "",
			"update_description": "",
		},

		"Error": nil,
	})
}

func getJobForTemplate(c *gin.Context, jobID string) (gin.H, error) {
	var jobData struct {
		ID     int
		Title  string
		Ticket string
	}
	query := `SELECT id, title, ticket_id FROM jobs WHERE id = $1`
	err := conn.QueryRow(c.Request.Context(), query, jobID).Scan(
		&jobData.ID,
		&jobData.Title,
		&jobData.Ticket,
	)
	if err != nil {
		log.Printf("Erro ao buscar job %s para renderizar erro: %v", jobID, err)
		return nil, err
	}
	return gin.H{
		"ID":     jobData.ID,
		"Title":  jobData.Title,
		"Ticket": jobData.Ticket,
	}, nil
}

func NewJobUpdate(c *gin.Context) {
	jobID := c.Param("id")

	loggedInUserID, ok := c.Get("userID")
	if !ok {
		c.HTML(http.StatusOK, "newJobUpdate.html", gin.H{
			"Error": "An internal error occurred while creating the job update. Please try again.",
		})
		return
	}

	updateTitle := c.PostForm("update_title")
	updateDescription := c.PostForm("update_description")

	formData := gin.H{
		"update_title":       updateTitle,
		"update_description": updateDescription,
	}

	renderError := func(errMsg string) {
		jobData, err := getJobForTemplate(c, jobID)
		if err != nil {
			c.String(http.StatusNotFound, "Job not found or internal error.")
			return
		}

		c.HTML(http.StatusUnprocessableEntity, "newJobUpdate.html", gin.H{
			"Job":      jobData,
			"FormData": formData,
			"Error":    errMsg,
		})
	}

	if updateTitle == "" {
		renderError("Update Title cannot be empty.")
		return
	}

	updateImgForm, err := c.FormFile("update_image")
	if err != nil {
		if err != http.ErrMissingFile {
			renderError("Error processing uploaded file.")
			return
		}
	}

	var updateImage string
	if updateImgForm != nil {
		imgUUID := uuid.New()
		var allowedImageTypes = map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".webp": true,
		}
		ext := strings.ToLower(filepath.Ext(updateImgForm.Filename))

		if !allowedImageTypes[ext] {
			renderError("Invalid file type. Only .jpg, .jpeg, .png, .webp are allowed.")
			return
		}

		newFileName := imgUUID.String() + ext

		destinationPath := filepath.Join("./uploads/thumbnails/", newFileName)

		src, err := updateImgForm.Open()
		if err != nil {
			log.Printf("Erro ao abrir arquivo enviado: %v", err)
			return
		}
		defer src.Close()

		dst, err := os.Create(destinationPath)
		if err != nil {
			log.Printf("Erro ao criar arquivo de destino: %v", err)
			return
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			log.Printf("Erro ao salvar arquivo: %v", err)
			return
		}

		updateImage = "/uploads/thumbnails/" + newFileName
	}

	contentMap := gin.H{
		"title":       updateTitle,
		"description": updateDescription,
	}
	if updateImage != "" {
		contentMap["img"] = []string{updateImage}
	}

	jobUpdateContent, err := json.Marshal(contentMap)
	if err != nil {
		renderError("An internal error occurred while creating the job update.")
		return
	}

	jobUpdateQuery := `INSERT INTO job_updates (job_id, author_user_id, content) VALUES ($1, $2, $3)`
	_, err = conn.Exec(c.Request.Context(), jobUpdateQuery, jobID, loggedInUserID, jobUpdateContent)
	if err != nil {
		renderError("Failed to save the job update. Please try again.")
		return
	}

	c.Header("HX-Redirect", "/jobs/"+jobID)
	c.Status(http.StatusOK)
}

func JobView(c *gin.Context) {
	jobID := c.Param("id")
	var jobData struct {
		ID               int
		Title            string
		Status           string
		Ticket           string
		JobTypeName      string
		ContactName      string
		AssignedUserName string
		CustomFields     map[string]any
		CreatedAt        time.Time
		UpdatedAt        time.Time
	}

	query := `
	SELECT
	    j.id,
	    j.title,
	    j.ticket_id,
	    j.status,
	    j.custom_fields,
	    jt.name AS job_type_name,
	    COALESCE(c.name, '') AS contact_name,
	    COALESCE(u.username, '') AS assigned_user_name,
	    j.created_at,
	    j.updated_at
	FROM
	    jobs j
	JOIN
	    job_types jt ON j.job_type_id = jt.id
	LEFT JOIN
	    contacts c ON j.primary_contact_id = c.id
	LEFT JOIN
	    users u ON j.assigned_to_user_id = u.id
	WHERE
	    j.id = $1;
	`

	err := conn.QueryRow(c.Request.Context(), query, jobID).Scan(&jobData.ID, &jobData.Title, &jobData.Ticket, &jobData.Status, &jobData.CustomFields, &jobData.JobTypeName, &jobData.ContactName, &jobData.AssignedUserName, &jobData.CreatedAt, &jobData.UpdatedAt)
	if err != nil {
		c.String(http.StatusNotFound, "Job not found")
		return
	}

	c.HTML(http.StatusOK, "viewJob.html", gin.H{
		"Job": jobData,
	})

}

func JobUpdateHistory(c *gin.Context) {
	jobID := c.Param("id")
	var pagination PaginationUpdates
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var beforeTimestamp time.Time
	var err error
	if pagination.Before == "" || pagination.Before == "now" {
		beforeTimestamp = time.Now()
	} else {
		beforeTimestamp, err = time.Parse(time.RFC3339Nano, pagination.Before)
		if err != nil {
			log.Printf("Invalid 'before' timestamp format: %s", pagination.Before)
			c.String(http.StatusBadRequest, "Invalid 'before' parameter format.")
			return
		}
	}

	if pagination.Limit <= 0 || pagination.Limit > 50 {
		pagination.Limit = 10
	}

	query := `
	SELECT
	    ju.id,
	    COALESCE(u.username, 'Unknown User') AS author_name, -- Return 'Unknown User' if user deleted
	    ju.created_at,
	    ju.content
	FROM
	    job_updates ju
	LEFT JOIN
	    users u ON ju.author_user_id = u.id
	WHERE
	    ju.job_id = $1 -- The specific job ID
	    AND ju.created_at < $2 -- The 'before' cursor timestamp
	ORDER BY
	    ju.created_at DESC -- Newest updates first
	LIMIT
	    $3; -- The number of updates per page
	`

	rows, err := conn.Query(c.Request.Context(), query, jobID, beforeTimestamp, pagination.Limit)
	if err != nil {
		log.Printf("failed to query items: %v", err)
		c.String(http.StatusInternalServerError, "Error fetching updates.")
		return
	}
	defer rows.Close()

	var jobUpdates []JobUpdate
	for rows.Next() {
		var job JobUpdate
		if err := rows.Scan(&job.ID, &job.AuthorName, &job.CreatedAt, &job.Content); err != nil {
			log.Printf("failed to scan row: %v", err)
			c.String(http.StatusInternalServerError, "Error processing updates.")
			return
		}

		jobUpdates = append(jobUpdates, job)
	}

	if err := rows.Err(); err != nil {
		log.Printf("error iterating rows: %v", err)
		c.String(http.StatusInternalServerError, "Error reading updates list.")
		return
	}

	var nextCursorStr string
	if len(jobUpdates) == pagination.Limit {
		nextCursorTime := jobUpdates[len(jobUpdates)-1].CreatedAt
		nextCursorStr = nextCursorTime.Format(time.RFC3339Nano)
	}

	c.HTML(http.StatusOK, "_jobUpdateFragmentJobView.html", gin.H{
		"Updates":          jobUpdates,
		"NextUpdateCursor": nextCursorStr,
		"JobID":            jobID,
	})

}
