package pdf

import (
	"bytes"
	"fmt"

	"github.com/shishir/go-service/models"

	"github.com/jung-kurt/gofpdf"
)

func GenerateStudentReport(s *models.Student) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 10, "Student Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	pairs := []string{
		fmt.Sprintf("Name: %s", s.Name),
		fmt.Sprintf("Email: %s", s.Email),
		fmt.Sprintf("Class: %s-%s", s.Class, s.Section),
		fmt.Sprintf("Roll No: %d", s.Roll),
		fmt.Sprintf("DOB: %s", s.Dob),
		fmt.Sprintf("Phone: %s", s.Phone),
		fmt.Sprintf("Father: %s (%s)", s.FatherName, s.FatherPhone),
		fmt.Sprintf("Mother: %s (%s)", s.MotherName, s.MotherPhone),
		fmt.Sprintf("Address: %s", s.CurrentAddress),
		fmt.Sprintf("Admission Date: %s", s.AdmissionDate),
	}

	for _, line := range pairs {
		pdf.Cell(0, 10, line)
		pdf.Ln(8)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
