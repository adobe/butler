package reloaders

import (
	"errors"
)

func NewGenericReloader(manager string, method string, entry []byte) (Reloader, error) {
	return GenericReloader{}, errors.New("Generic reloader is not very useful")
}

func NewGenericReloaderWithCustomError(manager string, method string, err error) (Reloader, error) {
	return GenericReloader{}, err
}

type GenericReloader struct {
	Opts GenericReloaderOpts
}

type GenericReloaderOpts struct {
}

func (r GenericReloader) Reload() error {
	var (
		res error
	)
	return res
}
func (r GenericReloader) GetMethod() string {
	return "none"
}
func (r GenericReloader) GetOpts() ReloaderOpts {
	return r.Opts
}

func (r GenericReloader) SetOpts(opts ReloaderOpts) bool {
	r.Opts = opts.(GenericReloaderOpts)
	return true
}
