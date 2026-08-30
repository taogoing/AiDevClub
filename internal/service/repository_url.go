package service

import (
	"net/url"
	"strings"
)

func validRepositoryURL(raw string) bool {
	u, err := url.ParseRequestURI(strings.TrimSpace(raw))
	return err == nil && u.Scheme == "https" && u.Host != ""
}
