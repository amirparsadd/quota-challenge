package main

import (
	"net"
	"os"
	"strings"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// listenHTTPAddr prefers PORT (12-factor), then HTTP_ADDR.
func listenHTTPAddr() string {
	if p := os.Getenv("PORT"); p != "" {
		if strings.HasPrefix(p, ":") {
			return p
		}
		return ":" + p
	}
	return env("HTTP_ADDR", ":8080")
}

// listenOpsAddr is the internal admin port (health + metrics); never publish to the public edge.
func listenOpsAddr() string {
	if p := os.Getenv("OPS_PORT"); p != "" {
		if strings.HasPrefix(p, ":") {
			return p
		}
		return ":" + p
	}
	return env("OPS_ADDR", ":8081")
}

// opsLoopbackHost returns a host:port suitable for local probe HTTP (e.g. healthcheck subcommand).
func opsLoopbackHost() string {
	addr := listenOpsAddr()
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr
	}
	if host, port, err := net.SplitHostPort(addr); err == nil {
		if host == "" || host == "0.0.0.0" || host == "::" {
			return net.JoinHostPort("127.0.0.1", port)
		}
	}
	return addr
}

func parseRegions(s string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		out[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return out
}