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
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/adobe/butler/internal/environment"

	"github.com/Azure/azure-sdk-for-go/storage"
	//log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type BlobMethod struct {
	StorageAccount string                    `mapstructure:"storage-account-name" json:"storage-account-name"`
	StorageKey     string                    `mapstructure:"storage-account-key" json:"storage-account-key"`
	AzureClient    storage.Client            `json:"-"`
	BlobClient     storage.BlobStorageClient `json:"-"`
}

func NewBlobMethod(manager *string, entry *string) (Method, error) {
	var (
		client storage.Client
		err    error
		result BlobMethod
	)

	if (manager != nil) && (entry != nil) {
		err = viper.UnmarshalKey(*entry, &result)
		if err != nil {
			return result, err
		}
	}

	result.StorageAccount = environment.GetVar(result.StorageAccount)
	result.StorageKey = environment.GetVar(result.StorageKey)

	if (result.StorageKey == "") && (environment.GetVar(os.Getenv("ACCOUNT_KEY")) == "") {
		return BlobMethod{}, errors.New("blob storage token undefined. Please set storage-account-key.")
	}
	if result.StorageKey == "" {
		result.StorageKey = environment.GetVar(os.Getenv("ACCOUNT_KEY"))
	}

	if (result.StorageAccount == "") && (environment.GetVar(os.Getenv("ACCOUNT_NAME")) == "") {
		return BlobMethod{}, errors.New("blob storage-account-name undefined.")
	}
	if result.StorageAccount == "" {
		result.StorageAccount = environment.GetVar(os.Getenv("ACCOUNT_NAME"))
	}

	client, err = storage.NewBasicClient(result.StorageAccount, result.StorageKey)
	if err != nil {
		return BlobMethod{}, fmt.Errorf("blob client error. err=%v", err)
	}

	result.AzureClient = client
	result.BlobClient = client.GetBlobService()

	return result, err
}

func NewBlobMethodWithAccountAndKey(account string, key string) (Method, error) {
	var (
		client storage.Client
		err    error
		result BlobMethod
	)

	if account == "" {
		return BlobMethod{}, errors.New("must provide a blob account name")
	}
	result.StorageAccount = environment.GetVar(account)

	if key == "" {
		return BlobMethod{}, errors.New("blob storage token undefined")
	}
	result.StorageKey = environment.GetVar(key)

	os.Setenv("ACCOUNT_KEY", result.StorageKey)
	os.Setenv("ACCOUNT_NAME", result.StorageAccount)

	client, err = storage.NewBasicClient(result.StorageAccount, result.StorageKey)
	if err != nil {
		return BlobMethod{}, fmt.Errorf("blob client error. err=%v", err)
	}

	result.AzureClient = client
	result.BlobClient = client.GetBlobService()

	return result, err
}

func (b BlobMethod) Get(u *url.URL) (*Response, error) {
	var (
		res Response
	)
	pathSplit := strings.Split(u.Path, "/")

	if len(pathSplit) < 2 {
		return &Response{}, errors.New("improper length for blob storage account/path")
	}

	container := pathSplit[1]
	blobFile := strings.Join(pathSplit[2:], "/")

	cnt := b.BlobClient.GetContainerReference(container)
	blob := cnt.GetBlobReference(blobFile)
	r, err := blob.Get(nil)
	if err != nil {
		return &Response{statusCode: 504}, err
	}
	res.body = r
	res.statusCode = 200
	return &res, nil
}

func (b *BlobMethod) SetStorageAccount(a string) {
	b.StorageAccount = environment.GetVar(a)
}

func (b *BlobMethod) SetStorageKey(k string) {
	b.StorageKey = environment.GetVar(k)
}
