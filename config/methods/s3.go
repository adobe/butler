package methods

import (
	//"github.com/prometheus/common/log"
	"github.com/spf13/viper"

)

type S3Method struct {

	Manager string `json::-"`
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

	return &Response{}, nil

}
