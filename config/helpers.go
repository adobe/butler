package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"git.corp.adobe.com/TechOps-IAO/butler/stats"

	"github.com/hoisie/mustache"
	log "github.com/sirupsen/logrus"
	"github.com/udhos/equalfile"
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

func ParseMustacheSubs(pairs []string) (map[string]string, error) {
	var (
		subs map[string]string
	)
	subs = make(map[string]string)

	for _, p := range pairs {
		p = strings.TrimSpace(p)
		keyvalpairs := strings.Split(p, "=")
		if len(keyvalpairs) != 2 {
			log.Infof("ParseMustacheSubs(): invalid key value pair \"%s\"... ignoring.", keyvalpairs)
			continue
		}
		key := strings.TrimSpace(keyvalpairs[0])
		val := strings.TrimSpace(keyvalpairs[1])
		subs[key] = val
	}
	// validate against RequiredSubKeys
	if !ValidateMustacheSubs(subs) {
		return nil, errors.New(fmt.Sprintf("could not validate required mustache subs. check your config. required subs=%s.", RequiredSubKeys))
	}
	return subs, nil
}

func ValidateMustacheSubs(Subs map[string]string) bool {
	var (
		subEntries map[string]bool
	)
	subEntries = make(map[string]bool)

	// set the default return value to false
	for _, vs := range RequiredSubKeys {
		subEntries[vs] = false
	}

	// range over the subs and see if the keys match the required list of substitution keys
	for k, _ := range Subs {
		if _, ok := subEntries[k]; ok {
			subEntries[k] = true
		}
	}

	// If any of the sub keys are false, then something is missing
	for _, v := range subEntries {
		if v == false {
			return false
		}
	}
	return true
}

// RenderConfigMustache takes a pointer to an os.File object. It reads the file
// attempts to parse the mustache
func RenderConfigMustache(f *os.File, subs map[string]string) error {
	tmpl, err := mustache.ParseFile(f.Name())
	if err != nil {
		return err
	}

	out := tmpl.Render(subs)

	f, err = os.OpenFile(f.Name(), os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write([]byte(out))
	if err != nil {
		return err
	}
	return nil
}

func CompareAndCopy(source string, dest string) bool {
	// Let's compare the source and destination files
	cmp := equalfile.New(nil, equalfile.Options{})
	equal, err := cmp.CompareFile(source, dest)
	if !equal {
		log.Infof("config.CompareAndCopy(): Found difference in \"%s.\"  Updating.", dest)
		err = CopyFile(source, dest)
		if err != nil {
			stats.SetButlerWriteVal(stats.FAILURE, stats.GetStatsLabel(dest))
			log.Infof(err.Error())
			return false
		}
		stats.SetButlerWriteVal(stats.SUCCESS, stats.GetStatsLabel(dest))
		return true
	} else {
		return false
	}
}

// CopyFile copies the src path string to the dst path string. If there is an
// error, an error is returned, otherwise nil is returned.
func CopyFile(src string, dst string) error {
	var (
		err error
		in  *os.File
		out *os.File
	)

	// open source
	in, err = os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// open destination
	if _, err = os.Stat(dst); err != nil {
		out, err = os.Create(dst)
	} else {
		out, err = os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0644)
	}
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

// CacheConfigs takes in a string of the base directory for
// the config directory and a slice of config file names and
// caches those files into memory. It returns an error
// on the event of error
func CacheConfigs(files []string) error {
	log.Infof("CacheConfig(): Storing known good configurations to cache.")
	ConfigCache = make(map[string][]byte)
	for _, file := range files {
		out, err := ioutil.ReadFile(file)
		if err != nil {
			msg := fmt.Sprintf("CacheConfig(): Could not store %s to cache. err=%s", file, err.Error())
			log.Infof(msg)
			return errors.New(msg)
		} else {
			ConfigCache[file] = out
		}
	}
	log.Infof("CacheConfig(): Done storing known good configurations to cache")
	return nil
}

// RestoreCachedConfigs takes in a strint of the base directory for
// the config directory and a slice of config file names
// and restores those files from the cache back to the
// filesystem. It returns an error on the event of an error
func RestoreCachedConfigs(files []string) error {
	// If we do not have a good configuration cache, then there's nothing for us to do.
	if ConfigCache == nil {
		log.Infof("RestoreCachedConfigFs(): No current known good configurations in cache. Cleaning configuration...")
		for _, file := range files {
			log.Infof("RestoreCachedConfigs(): Removing bad Prometheus configuration file %s.", file)
			os.Remove(file)
		}
		log.Infof("RestoreCachedConfigs(): Done cleaning broken configuration. Returning...")
		stats.SetButlerKnownGoodRestoredVal(stats.FAILURE)
		return nil
	}

	log.Infof("RestoreCachedConfigs(): Restoring known good Prometheus configurations from cache.")
	for _, file := range files {
		fileData := ConfigCache[file]

		f, err := os.OpenFile(file, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Infof("RestoreCachedConfigs(): Could not open %s for writing! err=%s.", file, err.Error())
			continue
		} else {
			count, err := f.Write(fileData)
			if err != nil {
				log.Infof("RestoreCachedConfigs(): Could write to %s! err=%s.", file, err.Error())
				continue
			} else {
				f.Close()
				log.Infof("RestoreCachedConfigs(): Wrote %d bytes for %s.", count, file)
			}
		}
	}
	log.Infof("RestoreCachedConfigs(): Done restoring known good Prometheus configurations from cache.")
	return nil
}
