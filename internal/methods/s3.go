/*
Copyright 2017 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package methods

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"

	"github.com/adobe/butler/internal/environment"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type S3Method struct {
	Bucket             string                `mapstructure:"bucket" json:"bucket"`
	AwsAccessKeyId     string                `mapstructure:"aws-access-key-id" json:"aws-access-key-id,omitempty"`
	AwsSecretAccessKey string                `mapstructure:"aws-secret-access-key" json:"-"`
	AwsSessionToken    string                `mapstructure:"aws-session-token" json:"-"`
	Manager            *string               `json:"-"`
	Region             string                `mapstructure:"region" json:"region"`
	Downloader         *s3manager.Downloader `json:"-"`
}

func NewS3Method(manager *string, entry *string) (Method, error) {
	var (
		err    error
		result S3Method
	)

	if (manager != nil) && (entry != nil) {

		err = viper.UnmarshalKey(*entry, &result)
		if err != nil {
			return result, err
		}

		result.Bucket = environment.GetVar(result.Bucket)
		result.Region = environment.GetVar(result.Region)

		// We should have something for both of these
		if (result.Bucket == "") || (result.Region == "") {
			return S3Method{}, errors.New("s3 bucket or region is not defined in config")
		}
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

func NewS3MethodWithRegionAndBucket(region string, bucket string) (Method, error) {
	var result S3Method

	sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		return S3Method{}, errors.New("could not start s3 session")
	}
	downloader := s3manager.NewDownloader(sess)

	result.Downloader = downloader
	result.Manager = nil
	result.Region = region
	result.Bucket = bucket
	return result, err
}

func (s S3Method) Get(u *url.URL) (*Response, error) {
	var (
		response Response
	)

	tmpFile, err := ioutil.TempFile("/tmp", "s3pcmsfile")
	if err != nil {
		return &Response{}, fmt.Errorf("S3Method::Get(): could not create temp file err=%v", err)
	}

	log.Debugf("S3Method::Get(): going to download s3 region=%v, bucket=%v, key=%v", s.Region, s.Bucket, u.Path)
	_, err = s.Downloader.Download(tmpFile,
		&s3.GetObjectInput{
			Bucket: aws.String(s.Bucket),
			Key:    aws.String(u.Path),
		})
	if err != nil {
		var code int
		if e, ok := err.(awserr.RequestFailure); ok {
			code = e.StatusCode()
		}
		if e, ok := err.(awserr.Error); ok {
			err2 := e.OrigErr()
			if err2 != nil {
				err = err2
			}
			// actually couldn't fulfill the reqeust since the host
			// probably doesn't exist. code = 504 is probably wrong but
			// whatever... gateway timeout will have to be good enough ;)
			code = 504
		}
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return &Response{statusCode: code}, fmt.Errorf("S3Method::Get(): caught error for download err=%v", err.Error())
	}

	fileData, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return &Response{statusCode: 500}, fmt.Errorf("S3Method::Get(): caught error read file err=%v", err.Error())
	}

	// Clean up the tmpfile
	tmpFile.Close()
	os.Remove(tmpFile.Name())

	response.statusCode = 200
	response.body = ioutil.NopCloser(bytes.NewReader(fileData))

	// Perhaps we need to do more stuff here
	return &response, nil
}
