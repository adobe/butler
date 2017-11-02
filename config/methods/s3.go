package methods

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/viper"
	//"net/http"
	"fmt"
	"github.com/prometheus/common/log"
	"os"
)

type S3Method struct {
	Manager string
	Bucket  string `mapstructure:"bucket-name" json:"bucket-name"`
	Region  string `mapstructure:"region" json:"region"`
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

	result.Manager = manager

	return result, err
}

func (s S3Method) GetBucket() (string){
	return s.Bucket
}

func (s S3Method) GetRegion() (string){
	return s.Region
}

func (s S3Method) Get(file string) (*Response, error) {

	f, err := os.Create(file)

	if err != nil {
		msg := fmt.Sprintf("Unable to open file. err=%v", err)
		log.Error(msg)
		return nil, err
	}
	_, err = s.Downloader.Download(f,
		&s3.GetObjectInput{
			Bucket: aws.String(s.Bucket),
			Key:    aws.String(file),
		})

	if err != nil {
		msg := fmt.Sprintf("unable to download item %q, %v", file, err)
		log.Error(msg)
		return nil, err
	}
	return &Response{}, nil

}
