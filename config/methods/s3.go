package methods

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"
	//"github.com/prometheus/common/log"
	"github.com/spf13/viper"
	"net/http"
	"os"
)

type S3Method struct {
	Bucket     string `json:"bucket-name"`
	Item       string `json:"file-name"`
	Manager    string `json::-"`
	Region     string `json:"region"`
	Sess       *session.Session
	Downloader *s3manager.Downloader
}

func NewS3Method(manager string, entry string) (Method, error) {
	var (
		err    error
		result S3Method
	)

	err = viper.UnmarshalKey(entry, &result)
	if err != nil {
		return result, err
	}

	result.Sess = session.Must(session.NewSession())
	result.Downloader = s3manager.NewDownloader(result.Sess)
	result.Manager = manager

	return result, err
}

func (s S3Method) Get(file string) (*http.Response, error) {

	f, err := os.Create(file)

	if err != nil {
		msg := fmt.Sprintf("Unable to open file. err=%v", err)
		log.Fatal(msg)
		return nil, err
	}
	_, err = s.Downloader.Download(f,
		&s3.GetObjectInput{
			Bucket: aws.String(s.Bucket),
			Key:    aws.String(s.Item),
		})
	return &http.Response{}, nil

}
