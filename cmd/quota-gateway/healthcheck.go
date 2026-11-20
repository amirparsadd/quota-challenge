package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

func runHealthcheck(live bool) int {
	path := "/readyz"
	if live {
		path = "/livez"
	}
	url := fmt.Sprintf("http://%s%s", opsLoopbackHost(), path)
	ctx, cancel := context.WithTimeout(context.Background(), healthcheckTimeoutFromEnv())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 1
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func runHealthcheckCmd(live bool) {
	os.Exit(runHealthcheck(live))
}