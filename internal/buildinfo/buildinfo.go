// Package buildinfo holds release metadata injected by GoReleaser.
package buildinfo

var (
	Version = "dev"
	Commit  = "unknown"
)

// String returns version and Git commit for --version output.
func String() string {
	return Version + " " + Commit
}