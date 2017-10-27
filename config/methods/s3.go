package methods

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	//"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/spf13/viper"
	"net/http"
)

type S3Method struct {
	Bucket     string `json:"bucket-name"`
	Item       string `json:"file-name"`
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

	result.Sess = session.Must(session.NewSession(&aws.Config{Region: aws.String("us-west-1")}))
	result.Downloader = s3manager.NewDownloader(result.Sess)

	return result, err
}

func (s S3Method) Get(file string) (*http.Response, error) {

	w := byte[]

	_, err := s.Downloader.Download(w,
	&s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(s.Item),
	})
	return &http.Response{}, nil

}
