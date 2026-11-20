package errreport

import (
	"regexp"

	"github.com/getsentry/sentry-go"
	"github.com/sopandgo/sopandgo/backend/internal/errkind"
)

// Validation errors. The API answers each with its own JSON code.
var (
	ErrDSNRequired        = errkind.New(errkind.Invalid, "a DSN is required to turn error reporting on")
	ErrDSNInvalid         = errkind.New(errkind.Invalid, "the DSN must look like https://<key>@<host>/<project>")
	ErrEnvironmentInvalid = errkind.New(errkind.Invalid, "the environment must be 1-64 letters, digits, '.', '_' or '-'")
)

var environmentPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)

// validate checks st as Save will store it (already trimmed).
func validate(st Settings) error {
	if st.DSN == "" {
		if st.Enabled {
			return ErrDSNRequired
		}
	} else if _, err := parseDSN(st.DSN); err != nil {
		return err
	}
	if !environmentPattern.MatchString(st.Environment) {
		return ErrEnvironmentInvalid
	}
	return nil
}

// parseDSN accepts http (self-hosted GlitchTip on a lab network) and https,
// like the Gotify URL in notify.
// DSNTarget returns the host and project ID a DSN sends to, or empty strings
// for an empty or invalid DSN. The audit log records these instead of the DSN.
func DSNTarget(dsn string) (host, project string) {
	if dsn == "" {
		return "", ""
	}
	d, err := parseDSN(dsn)
	if err != nil {
		return "", ""
	}
	return d.GetHost(), d.GetProjectID()
}

func parseDSN(raw string) (*sentry.Dsn, error) {
	dsn, err := sentry.NewDsn(raw)
	if err != nil {
		return nil, ErrDSNInvalid
	}
	if s := dsn.GetScheme(); s != "https" && s != "http" {
		return nil, ErrDSNInvalid
	}
	if dsn.GetPublicKey() == "" || dsn.GetProjectID() == "" || dsn.GetHost() == "" {
		return nil, ErrDSNInvalid
	}
	return dsn, nil
}
