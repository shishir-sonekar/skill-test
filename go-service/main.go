package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	
	"log"
	"net/http"
	"fmt"
	"os"

	"github.com/shishir/go-service/internal/client"
	"github.com/shishir/go-service/internal/pdf"
	
)

func GenerateReportHandler(c *gin.Context) {
	id := c.Param("id")

	stu, err := client.GetStudentByID(id)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	pdfBytes, err := pdf.GenerateStudentReport(stu)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "PDF generation failed"})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=student_"+id+".pdf")
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	fmt.Println("Node base URL:", os.Getenv("NODE_BASE_URL"))

	r := gin.Default()
	r.GET("/api/v1/students/:id/report", GenerateReportHandler)
	r.Run(":8080")
}
