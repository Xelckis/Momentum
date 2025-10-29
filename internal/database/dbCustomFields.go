package database

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type CustomFieldDef struct {
	FieldName  string   `json:"field_name"`
	FieldLabel string   `json:"field_label"`
	FieldType  string   `json:"field_type"`
	IsRequired bool     `json:"is_required"`
	Options    []string `json:"options"`
}
type CustomFieldDefList []CustomFieldDef

func GetCustomFieldsHandler(c *gin.Context) {
	id := c.Param("id")

	var jt jobType
	queryName := "SELECT name FROM job_types WHERE id = $1"
	err := conn.QueryRow(c.Request.Context(), queryName, id).Scan(&jt.Name)
	if err != nil {
		c.String(http.StatusNotFound, "Job Type not found")
		return
	}
	jt.ID, _ = strconv.Atoi(id)

	var currentFields CustomFieldDefList
	currentFields.fetchCurrentCustomFields(c, id)

	c.HTML(http.StatusOK, "manageCustomFieldsModal.html", gin.H{
		"JobType":      jt,
		"CustomFields": currentFields,
	})
}

func (currentFields *CustomFieldDefList) fetchCurrentCustomFields(c *gin.Context, id string) {
	var definitionsJSON []byte
	queryFields := "SELECT custom_field_definitions FROM job_types WHERE id = $1"
	err := conn.QueryRow(c.Request.Context(), queryFields, id).Scan(&definitionsJSON)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error fetching custom fields for job type %s: %v", id, err)
	} else if definitionsJSON != nil {
		err = json.Unmarshal(definitionsJSON, currentFields)
		if err != nil {
			log.Printf("Error unmarshaling custom fields for job type %s: %v", id, err)
			c.String(http.StatusInternalServerError, "Could not parse field definitions")
			return
		}
	}

}

func updateCustomFieldsDB(c *gin.Context, currentFields CustomFieldDefList, id string) {
	newCustomFieldsJSON, err := json.Marshal(currentFields)
	if err != nil {
		log.Printf("Error marshaling custom fields for job type %s: %v", id, err)
		return
	}

	query := `
		UPDATE job_types
		SET custom_field_definitions = $1
		WHERE id = $2;
	`

	cmdTag, err := conn.Exec(c.Request.Context(), query, newCustomFieldsJSON, id)
	if err != nil {
		log.Printf("Error updating custom fields for job type %s: %v", id, err)
		return
	}

	if cmdTag.RowsAffected() == 0 {
		log.Printf("Error finding custom fields for job type %s: %v", id, err)
		return
	}

}

func DeleteCustomFields(c *gin.Context) {
	id := c.Param("id")
	fieldName := c.Param("fieldName")
	var currentFields CustomFieldDefList
	currentFields.fetchCurrentCustomFields(c, id)

	found := false
	updatedFields := slices.DeleteFunc(currentFields, func(cf CustomFieldDef) bool {
		if cf.FieldName == fieldName {
			found = true
			return true
		}
		return false
	})

	if !found {
		log.Printf("Field '%s' not found for job type %s, nothing to delete.", fieldName, id)
	}

	updateCustomFieldsDB(c, updatedFields, id)

	c.Status(http.StatusOK)

}

func AddNewCustomFields(c *gin.Context) {
	id := c.Param("id")
	newFieldLabel := c.PostForm("field_label")
	newFieldName := c.PostForm("field_name")
	newFieldType := c.PostForm("field_type")
	newIsRequiredStr := c.PostForm("is_required")
	var newIsRequired bool = true
	var optionsSlice []string

	if newFieldName == "" || newFieldLabel == "" || newFieldType == "" {
		return
	}

	if newIsRequiredStr == "" {
		newIsRequired = false
	}
	isValidFormat, _ := regexp.MatchString(`^[a-z0-9_]+$`, newFieldName)
	if !isValidFormat {
		return
	}

	fieldNameExists, _ := customFieldNameExist(c, id, newFieldName)
	if fieldNameExists {
		log.Println("Field alredy exist")
		return
	}

	if newFieldType == "select" {
		optionsStr := c.PostForm("select_options")
		if optionsStr == "" {
			c.Header("HX-Retarget", "#add-field-feedback")
			c.HTML(http.StatusUnprocessableEntity, "errorFeedback.html", gin.H{"Message": "Error: Select field type requires options."})
			return
		}

		scanner := bufio.NewScanner(strings.NewReader(optionsStr))

		for scanner.Scan() {
			trimmedOpt := strings.TrimSpace(scanner.Text())

			if trimmedOpt != "" {
				optionsSlice = append(optionsSlice, trimmedOpt)
			}
		}

		if len(optionsSlice) == 0 {
			c.Header("HX-Retarget", "#add-field-feedback")
			c.HTML(http.StatusUnprocessableEntity, "errorFeedback.html", gin.H{"Message": "Error: Select field options cannot be empty."})
			return
		}
	}

	var currentFields CustomFieldDefList
	currentFields.fetchCurrentCustomFields(c, id)

	newCustomFields := CustomFieldDef{
		FieldName:  newFieldName,
		FieldLabel: newFieldLabel,
		FieldType:  newFieldType,
		IsRequired: newIsRequired,
		Options:    optionsSlice,
	}

	currentFields = append(currentFields, newCustomFields)

	updateCustomFieldsDB(c, currentFields, id)

	c.HTML(http.StatusOK, "manageCustomFieldsModal.html", gin.H{
		"CustomFields": currentFields,
		"JobType":      gin.H{"ID": id},
	})
}

func customFieldNameExist(c *gin.Context, id, fieldName string) (bool, error) {
	var exists bool
	query := `SELECT COALESCE(custom_field_definitions @> $2::jsonb, false)
              FROM job_types
              WHERE id = $1;`
	searchJSON := fmt.Sprintf(`[{"field_name": "%s"}]`, fieldName)

	err := conn.QueryRow(c.Request.Context(), query, id, searchJSON).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		log.Printf("Error checking custom field existence for job type %s: %v", id, err)
		return false, err
	}
	return exists, nil
}
