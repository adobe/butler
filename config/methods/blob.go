package methods

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"git.corp.adobe.com/TechOps-IAO/butler/environment"

	"github.com/Azure/azure-sdk-for-go/storage"
	//log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type BlobMethod struct {
	StorageAccount string                    `mapstructure:"storage-account-name" json:"storage-account-name"`
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
	authToken := os.Getenv("BUTLER_STORAGE_TOKEN")
	if (os.Getenv("ACCOUNT_KEY") == "") && (authToken == "") {
		return BlobMethod{}, errors.New("blob storage token undefined. Please set the BUTLER_STORAGE_TOKEN environment variable.")
	}
	os.Setenv("ACCOUNT_KEY", authToken)

	accountName := os.Getenv("BUTLER_STORAGE_ACCOUNT")
	if (accountName == "") && (result.StorageAccount == "") {
		return BlobMethod{}, errors.New("blob storage account name undefined")
	}

	if (result.StorageAccount == "") && (accountName != "") {
		result.StorageAccount = environment.GetVar(accountName)
	}

	os.Setenv("ACCOUNT_NAME", result.StorageAccount)

	client, err = storage.NewBasicClient(result.StorageAccount, authToken)
	if err != nil {
		return BlobMethod{}, errors.New(fmt.Sprintf("blob client error. err=%v", err))
	}

	result.AzureClient = client
	result.BlobClient = client.GetBlobService()

	return result, err
}

func NewBlobMethodWithAccount(account string) (Method, error) {
	var (
		client storage.Client
		err    error
		result BlobMethod
	)

	if account == "" {
		return BlobMethod{}, errors.New("must provide a blob account name")
	}

	result.StorageAccount = environment.GetVar(account)
	authToken := os.Getenv("BUTLER_STORAGE_TOKEN")
	if (os.Getenv("ACCOUNT_KEY") == "") && (authToken == "") {
		return BlobMethod{}, errors.New("blob storage token undefined")
	}
	os.Setenv("ACCOUNT_KEY", authToken)
	os.Setenv("ACCOUNT_NAME", result.StorageAccount)

	client, err = storage.NewBasicClient(result.StorageAccount, authToken)
	if err != nil {
		return BlobMethod{}, errors.New(fmt.Sprintf("blob client error. err=%v", err))
	}

	result.AzureClient = client
	result.BlobClient = client.GetBlobService()

	return result, err
}

func (b BlobMethod) Get(file string) (*Response, error) {
	var (
		container string
		object    string
		res       Response
	)
	fileSplit := strings.Split(file, "/")

	// this should only be /foo/bar or /foo/bar/baz
	if (len(fileSplit) < 2) || (len(fileSplit) > 3) {
		return &Response{}, errors.New("improper length for blob storage account/path")
	}

	// Check to see if the storage account is in the file name
	if strings.HasPrefix(file, b.StorageAccount) {
		// we have to worry about /account/container/file.txt
		container = fileSplit[1]
		object = fileSplit[2]
	} else {
		// we just need to worry about /container/file.txt
		container = fileSplit[0]
		object = fileSplit[1]
	}

	cnt := b.BlobClient.GetContainerReference(container)
	blob := cnt.GetBlobReference(object)
	r, err := blob.Get(nil)
	if err != nil {
		return &Response{statusCode: 504}, err
	}
	res.body = r
	res.statusCode = 200
	return &res, nil
}
