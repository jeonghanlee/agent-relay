package version

// Build-time variables injected via LDFLAGS.
var (
	Version   = "0.1.0-dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)
