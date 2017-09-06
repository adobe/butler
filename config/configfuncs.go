package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	//log "github.com/sirupsen/logrus"
)

func IsValidScheme(s string) bool {
	var (
		Found = false
	)
	for _, i := range ValidSchemes {
		if strings.ToLower(s) == i {
			Found = true
		}

	}
	return Found
}

// ValidateButlerConfig takes a pointer to an os.File object. It scans over the
// file and ensures that it begins with the proper header, and ends with the
// proper footer. If it does not begin or end with the proper header/footer,
// then an error is returned. If the file passes the checks, a nil is returned.
//func ValidateButlerConfig(f *os.File) error {
func ValidateButlerConfig(f interface{}) error {
	var (
		configLine    string
		isFirstLine   bool
		isValidHeader bool
		isValidFooter bool
		scanner       *bufio.Scanner
	)
	isFirstLine = true
	isValidHeader = true
	isValidFooter = true

	switch t := f.(type) {
	case *os.File:
		newf := f.(*os.File)
		file, err := os.Open(newf.Name())
		if err != nil {
			return err
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	case []byte:
		newf := f.([]byte)
		file := bytes.NewReader(newf)
		scanner = bufio.NewScanner(file)
	default:
		return errors.New(fmt.Sprintf("ValidateButlerConfig(): unknown file type %s for %s", t, f))
	}

	/*
	   file, err := os.Open(f.Name())
	   if err != nil {
	           return err
	   }
	   defer file.Close()
	*/

	//scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		configLine = scanner.Text()
		// Check that the header is valid
		if isFirstLine {
			if configLine != butlerHeader {
				isValidHeader = false
			}
			isFirstLine = false
		}
	}
	// Check that the footer is valid
	if configLine != butlerFooter {
		if configLine != butlerFooter {
			isValidFooter = false
		}
	}

	if !isValidHeader && !isValidFooter {
		return errors.New("Invalid butler header and footer")
	} else if !isValidHeader {
		return errors.New("Invalid butler header")
	} else if !isValidFooter {
		return errors.New("Invalid butler footer")
	} else {
		return nil
	}
}
