package methods

import (
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	//log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type S3Method struct {
	Manager    string                `json:"-"`
	Bucket     string                `mapstructure:"bucket" json:"bucket"`
	Region     string                `mapstructure:"region" json:"region"`
	Downloader *s3manager.Downloader `json:"-"`
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

	// We should have something for both of these
	if (result.Bucket == "") || (result.Region == "") {
		return S3Method{}, errors.New("s3 bucket or region is not defined in config")
	}

	sess, err := session.NewSession(&aws.Config{Region: aws.String(result.Region)})
	if err != nil {
		return S3Method{}, errors.New("could not start s3 session")
	}

	downloader := s3manager.NewDownloader(sess)

	result.Downloader = downloader
	result.Manager = manager

	return result, err
}

func (s S3Method) Get(file string, args ...interface{}) (*Response, error) {
	var (
		fPointer *os.File
	)

	// args should actually be a pointer to an os.File object
	if len(args) != 1 {
		return &Response{}, errors.New("S3Method::Get(): incorrect args count")
	}

	// Let's make sure it is indeed a pointer to an os.file object
	if arg, ok := args[0].(*os.File); !ok {
		return &Response{}, errors.New("S3Method::Get(): args[0] is not a *os.File object")
	} else {
		fPointer = arg
	}

	n, err := s.Downloader.Download(fPointer,
		&s3.GetObjectInput{
			Bucket: aws.String(s.Bucket),
			Key:    aws.String(file),
		})
	if err != nil {
		return &Response{}, errors.New(fmt.Sprintf("S3Method::Get(): caught error %v", err.Error()))
	}

	// Perhaps we need to do more stuff here
	_ = n
	return &Response{}, nil
}
