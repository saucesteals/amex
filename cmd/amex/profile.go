package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	http "github.com/saucesteals/fhttp"

	"github.com/saucesteals/amex"
)

var (
	ErrResourceMissing = errors.New("resource missing")
)

type Resource[T any] struct {
	mu       sync.Mutex
	data     T
	isLoaded bool
	path     string
}

func NewResource[T any](path string) *Resource[T] {
	return &Resource[T]{
		path: path,
	}
}

func (r *Resource[T]) load() error {
	contents, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: %w", r.path, ErrResourceMissing)
		}

		return err
	}

	err = json.Unmarshal(contents, &r.data)
	if err != nil {
		return err
	}

	return nil
}

func (r *Resource[T]) save() error {
	contents, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(r.path, contents, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (r *Resource[T]) Set(data T) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data = data
	r.isLoaded = true
	return r.save()
}

func (r *Resource[T]) Get() (T, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isLoaded {
		err := r.load()
		if err != nil {
			return r.data, err
		}

		r.isLoaded = true
	}

	return r.data, nil
}

type Profile struct {
	path string

	Credentials *Resource[amex.Credentials]
	Cookies     *Resource[[]*http.Cookie]
}

func GetProgramDir(subfolders ...string) (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	base := filepath.Join(userHome, "amex")
	for _, subfolder := range subfolders {
		base = filepath.Join(base, subfolder)
	}

	if _, err := os.Stat(base); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}

		err = os.MkdirAll(base, 0700)
		if err != nil {
			return "", err
		}
	}

	return base, nil
}

func ImportProfile(username string) (*Profile, error) {
	dir, err := GetProgramDir("profiles", username)
	if err != nil {
		return nil, err
	}

	return &Profile{
		path:        dir,
		Credentials: NewResource[amex.Credentials](filepath.Join(dir, "credentials.json")),
		Cookies:     NewResource[[]*http.Cookie](filepath.Join(dir, "cookies.json")),
	}, nil
}

func (p *Profile) GetDirectory(parts ...string) (string, error) {
	dir := filepath.Join(p.path, filepath.Join(parts...))
	if _, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}

		err = os.MkdirAll(dir, 0700)
		if err != nil {
			return "", err
		}
	}

	return dir, nil
}
