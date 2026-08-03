//go:build localstack

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

// This file exercises S3Method against a real S3-compatible API (LocalStack)
// instead of a hand-rolled httptest.Server. It validates things the fake
// server in s3_test.go cannot: real request signing/auth being accepted,
// real bucket/key existence semantics, and real S3 error responses.
//
// It is excluded from normal `go test ./...` runs via the `localstack`
// build tag, since it requires a running LocalStack container. To run:
//
//	docker run -d --name butler-localstack -p 4566:4566 \
//	  -e SERVICES=s3 localstack/localstack:3.0
//	go test -tags localstack -v ./internal/methods/... -run LocalStack
package methods

import (
	"bytes"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const localstackEndpoint = "http://localhost:4566"

func localstackSession(t *testing.T) *session.Session {
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String("us-east-1"),
		Endpoint:         aws.String(localstackEndpoint),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(true),
		// LocalStack accepts any non-empty static credentials for its
		// community S3 emulation -- there's no real AWS account behind it.
		Credentials: credentials.NewStaticCredentials("test", "test", ""),
	})
	if err != nil {
		t.Fatalf("could not create localstack session: %v", err)
	}
	return sess
}

func ensureLocalstackBucketAndObject(t *testing.T, sess *session.Session, bucket, key, body string) {
	svc := s3.New(sess)
	_, err := svc.CreateBucket(&s3.CreateBucketInput{Bucket: aws.String(bucket)})
	if err != nil {
		if !bucketAlreadyOwned(err) {
			t.Fatalf("could not create bucket: %v", err)
		}
	}
	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader([]byte(body)),
	})
	if err != nil {
		t.Fatalf("could not put object: %v", err)
	}
}

func bucketAlreadyOwned(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") || strings.Contains(err.Error(), "BucketAlreadyExists"))
}

// TestLocalStackGetSucceeds exercises the real construction path
// (NewS3MethodWithOpts) and Get() against a real S3-compatible server:
// real SigV4 signing, real bucket lookup, real object retrieval.
func TestLocalStackGetSucceeds(t *testing.T) {
	os.Setenv("AWS_ACCESS_KEY_ID", "test")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	defer os.Unsetenv("AWS_ACCESS_KEY_ID")
	defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	sess := localstackSession(t)
	ensureLocalstackBucketAndObject(t, sess, "butler-test-bucket", "config/prometheus.yml", "hello from localstack")

	opts := S3MethodOpts{
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		Bucket:          "butler-test-bucket",
		Region:          "us-east-1",
		Retries:         3,
	}
	method, err := NewS3MethodWithOpts(opts)
	if err != nil {
		t.Fatalf("NewS3MethodWithOpts failed: %v", err)
	}

	// Point the constructed method's downloader at LocalStack. This is the
	// one seam NewS3MethodWithOpts doesn't expose via opts (there's no
	// -s3.endpoint flag, real butler always talks to real AWS), so the test
	// rebuilds the downloader the same way production code does, just
	// against the LocalStack endpoint.
	m := method.(S3Method)
	m.Downloader = newDownloaderForTest(t, "test", "test", "us-east-1", 3)

	u, err := url.Parse("/config/prometheus.yml")
	if err != nil {
		t.Fatalf("url.Parse failed: %v", err)
	}

	resp, err := m.Get(u)
	if err != nil {
		t.Fatalf("Get() failed against real S3-compatible API: %v", err)
	}
	if resp.GetResponseStatusCode() != 200 {
		t.Fatalf("expected status 200, got %v", resp.GetResponseStatusCode())
	}
	out, err := ioutil.ReadAll(resp.GetResponseBody())
	if err != nil {
		t.Fatalf("could not read response body: %v", err)
	}
	if string(out) != "hello from localstack" {
		t.Fatalf("expected body %q, got %q", "hello from localstack", string(out))
	}
}

// TestLocalStackGetMissingKeyReturns404 confirms a real S3-compatible
// NoSuchKey error surfaces as a 404 through Get(), validating the
// RequestFailure status-code fix against real S3 XML error responses
// rather than a hand-crafted httptest.Server response.
func TestLocalStackGetMissingKeyReturns404(t *testing.T) {
	sess := localstackSession(t)
	ensureLocalstackBucketAndObject(t, sess, "butler-test-bucket", "config/exists.yml", "present")

	m := S3Method{
		Bucket:     "butler-test-bucket",
		Region:     "us-east-1",
		Downloader: newDownloaderForTest(t, "test", "test", "us-east-1", 3),
	}

	u, err := url.Parse("/config/does-not-exist.yml")
	if err != nil {
		t.Fatalf("url.Parse failed: %v", err)
	}

	resp, err := m.Get(u)
	if err == nil {
		t.Fatalf("expected error for missing key, got nil")
	}
	if resp.GetResponseStatusCode() != 404 {
		t.Fatalf("expected status 404 for missing key, got %v (err=%v)", resp.GetResponseStatusCode(), err)
	}
}

func newDownloaderForTest(t *testing.T, accessKey, secretKey, region string, maxRetries int) *s3manager.Downloader {
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(region),
		Endpoint:         aws.String(localstackEndpoint),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(true),
		MaxRetries:       aws.Int(maxRetries),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		t.Fatalf("could not create session: %v", err)
	}
	return s3manager.NewDownloader(sess)
}
