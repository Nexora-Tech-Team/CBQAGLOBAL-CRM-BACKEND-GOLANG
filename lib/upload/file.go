package upload

import (
	"encoding/base64"
	"errors"
	"fmt"
	"google.golang.org/api/option"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/h2non/filetype"
	"golang.org/x/net/context"
)

// Upload is function for create folder if not exist and upload file to the folder
func UploadBase64(initial, file string) (string, error) {
	dec, err := base64.StdEncoding.DecodeString(file[strings.IndexByte(file, ',')+1:])
	if err != nil {
		return "", errors.New("can't decode file with error " + err.Error())
	}

	ext2, err := filetype.Get(dec)
	if err != nil {
		return "", errors.New("can't get buf with error " + err.Error())
	}

	var extension string
	if ext2.Extension != "unknown" {
		extension = ext2.Extension
	} else {
		ext := strings.ReplaceAll(file[:strings.IndexByte(file, ';')+1], "data:image/", "")
		if ext == "data:text/csv;" || ext == "data:text/plain;" {
			ext = strings.ReplaceAll(file[:strings.IndexByte(file, ';')+1], "data:text/", "")
			if ext == "plain;" {
				extension = "txt"
			} else if ext == "csv;" {
				extension = "csv"
			} else {
				extension = "unknown"
			}

		} else if ext == "data:application/vnd.ms-powerpoint;" ||
			ext == "data:application/vnd.openxmlformats-officedocument.presentationml.presentation" {

			ext = strings.ReplaceAll(file[:strings.IndexByte(file, ';')+1], "data:application/", "")

			if ext == "vnd.ms-powerpoint;" {
				extension = "ppt"
			} else if ext == "vnd.openxmlformats-officedocument.presentationml.presentation;" {
				extension = "pptx"
			} else {
				extension = "unknown"
			}

		} else if ext == "svg+xml;" {
			extension = "svg"
		} else {
			extension = "unknown"
		}
	}

	if extension == "unknown" {
		return "", errors.New("can't get extension file: unsupported file extension")
	}

	fName := initial + time.Now().Format("2006-01-02T15:04:05") + strconv.Itoa(rand.Intn(200)) + "." + extension

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", errors.New("can't create GCS client with error " + err.Error())
	}
	defer client.Close()

	bucket := client.Bucket(os.Getenv("GOOGLE_BUCKET"))

	obj := bucket.Object(fName)

	wc := obj.NewWriter(ctx)

	_, err = wc.Write(dec)
	if err != nil {
		return "", errors.New("can't write file to GCS with error " + err.Error())
	}

	if err := wc.Close(); err != nil {
		return "", errors.New("can't close GCS WriteCloser with error " + err.Error())
	}

	return fName, nil
}

func UploadFileToGCS(bucket, object, filePath, credentialsFile string) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return err
	}
	defer client.Close()

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	wc := client.Bucket(bucket).Object(object).NewWriter(ctx)
	wc.ContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if _, err = io.Copy(wc, f); err != nil {
		return err
	}
	if errClose := wc.Close(); errClose != nil {
		return errClose
	}
	return nil
}

func GenerateSignedURL(bucket, object string) (string, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		return "", err
	}
	defer client.Close()
	privateKey := os.Getenv("GOOGLE_PRIVATE_KEY")
	if privateKey == "" {
		return "", fmt.Errorf("GOOGLE_PRIVATE_KEY environment variable not set")
	}
	privateKey = strings.ReplaceAll(privateKey, "\\n", "\n")
	url, err := client.Bucket(bucket).SignedURL(object, &storage.SignedURLOptions{
		GoogleAccessID: os.Getenv("GOOGLE_ACCESS_ID"),
		PrivateKey:     []byte(privateKey),
		Method:         "GET",
		Expires:        time.Now().Add(100 * 365 * 24 * time.Hour),
	})
	if err != nil {
		return "", err
	}
	return url, nil
}

func UploadFilePdfToGCS(bucket, object, filePath, credentialsFile string) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return err
	}
	defer client.Close()

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	wc := client.Bucket(bucket).Object(object).NewWriter(ctx)
	wc.ContentType = "application/pdf"
	if _, err = io.Copy(wc, f); err != nil {
		return err
	}
	if errClose := wc.Close(); errClose != nil {
		return errClose
	}
	return nil
}
