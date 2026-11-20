package main

import (
	"os"
	"strings"
	"time"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func flagString(args []string, name, def string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == name && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(args[i], name+"=") {
			return strings.TrimPrefix(args[i], name+"=")
		}
	}
	return def
}

func flagBool(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
	}
	return false
}

func flagDuration(args []string, name string) time.Duration {
	for _, a := range args {
		if strings.HasPrefix(a, name+"=") {
			d, err := time.ParseDuration(strings.TrimPrefix(a, name+"="))
			if err != nil {
				return 0
			}
			return d
		}
	}
	return 0
}