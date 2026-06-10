package contract

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RequireFlag returns a VALIDATION_ERROR if value is empty.
func RequireFlag(name, value string) error {
	if value == "" {
		return NewError(ValidationError, fmt.Sprintf("--%s is required", name))
	}
	return nil
}

// RequireExactlyOne returns a VALIDATION_ERROR if not exactly one flag has a non-empty value.
func RequireExactlyOne(flags map[string]string) error {
	var set []string
	var all []string
	for name, value := range flags {
		all = append(all, "--"+name)
		if value != "" {
			set = append(set, "--"+name)
		}
	}
	sort.Strings(all)
	if len(set) == 1 {
		return nil
	}
	return NewError(ValidationError, fmt.Sprintf("exactly one of %s is required", strings.Join(all, ", ")))
}

// RequireAtLeastOne returns a VALIDATION_ERROR if no flags have a non-empty value.
func RequireAtLeastOne(flags map[string]string) error {
	var all []string
	for name, value := range flags {
		all = append(all, "--"+name)
		if value != "" {
			return nil
		}
	}
	sort.Strings(all)
	return NewError(ValidationError, fmt.Sprintf("at least one of %s is required", strings.Join(all, ", ")))
}

// ValidateISO8601 returns a VALIDATION_ERROR if the value is not a valid ISO-8601 date.
func ValidateISO8601(value string) error {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
	}
	for _, f := range formats {
		if _, err := time.Parse(f, value); err == nil {
			return nil
		}
	}
	return NewError(ValidationError, fmt.Sprintf("invalid ISO-8601 date: %q", value))
}

// ValidateISO8601Optional validates an ISO-8601 date only if non-empty.
func ValidateISO8601Optional(value string) error {
	if value == "" {
		return nil
	}
	return ValidateISO8601(value)
}

// ValidateURL returns a VALIDATION_ERROR if the value is not a valid HTTP(S) URL.
func ValidateURL(value string) error {
	if value == "" {
		return NewError(ValidationError, "URL is required")
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return NewError(ValidationError, fmt.Sprintf("invalid URL: %q", value))
	}
	return nil
}

// ValidateFilePath returns a FILE_NOT_FOUND if the path is empty or does not exist.
func ValidateFilePath(path string) error {
	if path == "" {
		return NewError(FileNotFound, "file path is required")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return NewError(FileNotFound, fmt.Sprintf("file not found: %s", path))
	}
	return nil
}

// ValidateOutputPath returns an error if a download output path is unusable.
func ValidateOutputPath(path string, force bool) error {
	if path == "" {
		return NewError(ValidationError, "output path is required")
	}
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() || force {
			return nil
		}
		return NewError(Conflict, fmt.Sprintf("output file already exists: %s", path))
	}
	if !os.IsNotExist(err) {
		return NewError(UnknownError, fmt.Sprintf("cannot inspect output path: %v", err))
	}
	parent := filepath.Dir(path)
	if parent == "." || parent == "" {
		return nil
	}
	if parentInfo, err := os.Stat(parent); err != nil {
		return NewError(FileNotFound, fmt.Sprintf("output directory not found: %s", parent))
	} else if !parentInfo.IsDir() {
		return NewError(ValidationError, fmt.Sprintf("output parent is not a directory: %s", parent))
	}
	return nil
}
