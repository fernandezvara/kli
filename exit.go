package kli

import "errors"

// Process exit codes used by the library. Command functions may return any
// other code with Exit.
const (
	ExitOK      = 0 // success, or help shown
	ExitFailure = 1 // a command failed at run time
	ExitUsage   = 2 // unknown command or flag, missing or invalid flag value
)

// ExitError is an error that carries the process exit code the program
// should end with. Execute never exits the process itself: the caller
// passes the result to ExitCode and to os.Exit.
type ExitError struct {
	Code int
	Err  error
	// Reported is true when the library already printed the details of
	// this error (for example the configuration errors of a command), so
	// the caller should not print it again.
	Reported bool
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

// Exit returns an error that makes ExitCode report code. Commands return
// it to end with a chosen exit code: return cli.Exit(3, err). err may be
// nil for a silent exit.
func Exit(code int, err error) error {
	return &ExitError{Code: code, Err: err}
}

// ExitCode is the exit code for the error Execute returned: 0 for nil,
// the code of an ExitError anywhere in the chain, 1 for any other error.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return ExitFailure
}

// IsReported reports whether the library already printed err's details.
func IsReported(err error) bool {
	var ee *ExitError
	return errors.As(err, &ee) && ee.Reported
}

func usageError(err error, reported bool) error {
	return &ExitError{Code: ExitUsage, Err: err, Reported: reported}
}
