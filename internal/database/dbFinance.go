package database

import (
	"Momentum/internal/logger"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type PaginationFinance struct {
	Limit  int    `form:"limit,default=10"`
	Before string `form:"before,default=now"`
}

type Finance struct {
	ID              int
	Description     string
	TransactionDate time.Time
	Type            string
	Amount          decimal.Decimal
	RelatedJobID    sql.NullInt64
	CreatedAt       time.Time
}

func FinanceList(c *gin.Context) {
	var pagination PaginationFinance
	if err := c.ShouldBindQuery(&pagination); err != nil {
		logger.LogToLogFile(c, fmt.Sprintf("Finance List [Bind Query]: Error while binding query to pagination struct `%v`", err))
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
			logger.LogToLogFile(c, fmt.Sprintf("Finance List [Before Time]: Invalid 'before' timestamp format: %s", pagination.Before))
			c.String(http.StatusBadRequest, "Invalid 'before' parameter format.")
			return
		}
	}

	if pagination.Limit <= 0 || pagination.Limit > 50 {
		pagination.Limit = 10
	}

	query := `
	SELECT
	    id,
	    description,
	    amount,
	    type,
	    transaction_date,
	    related_job_id,
	    created_at 
	FROM
	    financial_transactions
	WHERE
	    created_at < $1 
	ORDER BY
	    created_at DESC 
	LIMIT
	    $2; 
	`

	rows, err := conn.Query(c.Request.Context(), query, beforeTimestamp, pagination.Limit)
	if err != nil {
		logger.LogToLogFile(c, fmt.Sprintf("Finance List [SQL]: Error querying financial_transactions table `%v`", err))
		c.String(http.StatusInternalServerError, "An internal server error occurred, Try again.")
		return
	}
	defer rows.Close()

	var finances []Finance
	for rows.Next() {
		var record Finance
		if err := rows.Scan(&record.ID, &record.Description, &record.Amount, &record.Type, &record.TransactionDate, &record.RelatedJobID, &record.CreatedAt); err != nil {
			logger.LogToLogFile(c, fmt.Sprintf("Finance List [SQL]: Failed to scan row `%v`", err))
			c.String(http.StatusInternalServerError, "Error processing financial records.")
			return
		}

		finances = append(finances, record)
	}

	if err := rows.Err(); err != nil {
		logger.LogToLogFile(c, fmt.Sprintf("Finance List [SQL]: Error interating rows `%v`", err))
		c.String(http.StatusInternalServerError, "Error reading updates list.")
		return
	}

	var nextCursorStr string
	if len(finances) == pagination.Limit {
		nextCursorTime := finances[len(finances)-1].CreatedAt
		nextCursorStr = nextCursorTime.Format(time.RFC3339Nano)
	}

	c.HTML(http.StatusOK, "_financialTransactionFragment.html", gin.H{
		"Transactions": finances,
		"NextCursor":   nextCursorStr,
	})

}

func AddNewFinancialRecord(c *gin.Context) {
	description := c.PostForm("description")
	amountStr := c.PostForm("amount")
	typeRecord := c.PostForm("type")
	transactionDateStr := c.PostForm("transaction_date")
	relatedJobID := c.PostForm("related_job_id")
	relatedJobIDNull := sql.NullString{String: relatedJobID}

	formData := gin.H{
		"description":      description,
		"amount":           amountStr,
		"type":             typeRecord,
		"transaction_date": transactionDateStr,
		"related_job_id":   relatedJobID,
	}

	if description == "" || amountStr == "" || typeRecord == "" {
		c.HTML(http.StatusOK, "newFinancialRecordForm.html", gin.H{
			"FormData":           formData,
			"SelectedJobDisplay": "",
			"Error":              "Description, Amount, and Type are required.",
		})
		return
	}

	if typeRecord != "income" && typeRecord != "expense" {
		c.HTML(http.StatusOK, "newFinancialRecordForm.html", gin.H{
			"FormData":           formData,
			"SelectedJobDisplay": "",
			"Error":              "Type must be income or expanse",
		})
		return
	}

	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		c.HTML(http.StatusOK, "newFinancialRecordForm.html", gin.H{
			"FormData":           formData,
			"SelectedJobDisplay": "",
			"Error":              "An internal server error occurred while converting the amount, Try again.",
		})
		return

	}

	var transactionDate time.Time
	if transactionDateStr == "" {
		transactionDate = time.Now()
	} else {
		parsedDate, err := time.Parse("2006-01-02", transactionDateStr)
		if err != nil {
			c.HTML(http.StatusOK, "newFinancialRecordForm.html", gin.H{
				"FormData":           formData,
				"SelectedJobDisplay": "",
				"Error":              "Invalid date format. Use YYYY-MM-DD.",
			})
			return
		}
		transactionDate = parsedDate
	}

	query := `INSERT INTO financial_transactions (description, amount, type, transaction_date, related_job_id) VALUES ($1, $2, $3, $4, $5)`
	_, err = conn.Exec(c.Request.Context(), query, description, amount, typeRecord, transactionDate, relatedJobIDNull)

	if err != nil {
		logger.LogToLogFile(c, fmt.Sprintf("Add New Financial Record [SQL]: Error while inserting record into financial_transactions `%v`", err))
		c.HTML(http.StatusOK, "newFinancialRecordForm.html", gin.H{
			"FormData":           formData,
			"SelectedJobDisplay": "",
			"Error":              "An internal server error occurred, Try again.",
		})
		return
	}

	c.Status(http.StatusOK)

}
