package pdf

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"fmt"
	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"google.golang.org/api/option"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func GeneratePdf(template, fileName string) (map[string]string, error) {
	pdfGen, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, err
	}
	// Set global options
	pdfGen.Dpi.Set(300)
	pdfGen.MarginTop.Set(5)
	pdfGen.MarginBottom.Set(5)
	pdfGen.MarginLeft.Set(15)
	pdfGen.MarginRight.Set(5)
	// Add to document
	pdfGen.AddPage(wkhtmltopdf.NewPageReader(strings.NewReader(template)))
	// Create PDF document in internal buffer
	err = pdfGen.Create()
	if err != nil {
		return nil, err
	}

	// Define the path and filename for the PDF file
	path := "assets/"

	// check if folder doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, os.ModePerm); err != nil {
			return nil, errors.New("can't create folder with error " + err.Error())
		}
	}

	// save file
	if err := pdfGen.WriteFile(path + "/" + fileName); err != nil {
		return nil, errors.New("can't save file with error " + err.Error())
	}
	var result = map[string]string{
		"name": fileName,
		"path": os.Getenv("BASE_URL") + "/" + path,
	}
	return result, nil
}

func UploadPDFToGCS(bucketName, objectName, filePath, keyFilePath string) error {
	// Read the file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Set up Google Cloud Storage client with credentials
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(keyFilePath))
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}
	defer client.Close()

	// Open a handle to your bucket
	bucket := client.Bucket(bucketName)

	// Get current month and year
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")

	// Construct the object name with the folder structure
	objectNameWithFolder := fmt.Sprintf("%s%s/%s", year, month, objectName)

	// Create a new object in the bucket
	obj := bucket.Object(objectNameWithFolder)

	// Write the file's data to the object
	wc := obj.NewWriter(ctx)
	defer wc.Close()

	if _, err := wc.Write(data); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	// Print success message
	fmt.Printf("File %s uploaded to %s\n", filePath, bucketName)

	return nil
}
