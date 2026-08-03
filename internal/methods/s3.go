/*
Copyright 2017-2026 Adobe. All rights reserved.
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
	"strconv"

	"github.com/adobe/butler/internal/environment"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// defaultS3Retries matches aws-sdk-go's own client.DefaultRetryer.NumMaxRetries,
// so an unset/invalid config value keeps today's implicit SDK behavior.
const defaultS3Retries = 3

type S3Method struct {
	AccessKeyID     string                `mapstructure:"access-key-id" json:"access-key-id"`
	Bucket          string                `mapstructure:"bucket" json:"bucket"`
	Downloader      *s3manager.Downloader `json:"-"`
	Manager         *string               `json:"-"`
	Region          string                `mapstructure:"region" json:"region"`
	Retries         string                `mapstructure:"retries" json:"retries"`
	SecretAccessKey string                `mapstructure:"secret-access-key" json:"-"`
	SessionToken    string                `mapstructure:"session-token" json:"-"`
}

type S3MethodOpts struct {
	AccessKeyID     string
	Bucket          string
	Region          string
	Retries         int
	Scheme          string
	SecretAccessKey string
	SessionToken    string
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

	result.AccessKeyID = environment.GetVar(result.AccessKeyID)
	if result.AccessKeyID == "" {
		result.AccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	}

	result.SecretAccessKey = environment.GetVar(result.SecretAccessKey)
	if result.SecretAccessKey == "" {
		result.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}

	result.SessionToken = environment.GetVar(result.SessionToken)
	if result.SessionToken == "" {
		result.SessionToken = os.Getenv("AWS_SESSION_TOKEN")
	}

	newRetries, _ := strconv.Atoi(environment.GetVar(result.Retries))
	if newRetries == 0 {
		log.Warnf("NewS3Method(): could not convert %v to integer for retries, defaulting to %v. This is probably undesired.", result.Retries, defaultS3Retries)
		newRetries = defaultS3Retries
	}

	sess, err := session.NewSession(&aws.Config{
		Region:     aws.String(result.Region),
		MaxRetries: aws.Int(newRetries),
		Credentials: credentials.NewStaticCredentials(result.AccessKeyID,
			result.SecretAccessKey,
			result.SessionToken),
	})
	if err != nil {
		return S3Method{}, errors.New("could not start s3 session")
	}

	downloader := s3manager.NewDownloader(sess)

	result.Downloader = downloader
	result.Manager = manager

	return result, err
}

func NewS3MethodWithOpts(opts S3MethodOpts) (Method, error) {
	var result S3Method

	newRetries := opts.Retries
	if newRetries == 0 {
		newRetries = defaultS3Retries
	}

	sess, err := session.NewSession(&aws.Config{
		Region:     aws.String(opts.Region),
		MaxRetries: aws.Int(newRetries),
		Credentials: credentials.NewStaticCredentials(opts.AccessKeyID,
			opts.SecretAccessKey,
			opts.SessionToken),
	})
	if err != nil {
		return S3Method{}, errors.New("could not start s3 session")
	}
	downloader := s3manager.NewDownloader(sess)

	result.Downloader = downloader
	result.Manager = nil
	result.Region = opts.Region
	result.Retries = strconv.Itoa(newRetries)
	result.Bucket = opts.Bucket
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
			// The request reached S3 and got back an actual HTTP error
			// (eg: 404 for a missing key, 500 for a server-side failure).
			// Surface that real status code rather than guessing.
			code = e.StatusCode()
		} else if e, ok := err.(awserr.Error); ok {
			err2 := e.OrigErr()
			if err2 != nil {
				err = err2
			}
			// The request never got a response from S3 at all (eg: the
			// host doesn't exist, or a network-level failure). code = 504
			// is probably wrong but whatever... gateway timeout will have
			// to be good enough ;)
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

func (o S3MethodOpts) GetScheme() string {
	return o.Scheme
}
