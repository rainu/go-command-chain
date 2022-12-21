package cmdchain

import "os/exec"

// ErrorChecker is a function which will receive the command's error. His purposes is to check if the given error can
// be ignored. If the function return true the given error is a "real" error and will NOT be ignored!
type ErrorChecker func(index int, command *exec.Cmd, err error) bool

// IgnoreExitCode will return an ErrorChecker. This will ignore all exec.ExitError which have any of the given exit codes.
func IgnoreExitCode(allowedCodes ...int) ErrorChecker {
	return func(_ int, _ *exec.Cmd, err error) bool {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()

			for _, allowedCode := range allowedCodes {
				if allowedCode == exitCode {
					return false
				}
			}
		}

		// its a "true" error
		return true
	}
}

// IgnoreExitErrors will return an ErrorChecker. This will ignore all exec.ExitError.
func IgnoreExitErrors() ErrorChecker {
	return func(_ int, _ *exec.Cmd, err error) bool {
		_, isExitError := err.(*exec.ExitError)

		return !isExitError
	}
}

// IgnoreAll will return an ErrorChecker. This will ignore all error.
func IgnoreAll() ErrorChecker {
	return func(_ int, _ *exec.Cmd, _ error) bool {
		return false
	}
}

// IgnoreNothing will return an ErrorChecker. This will ignore no error.
func IgnoreNothing() ErrorChecker {
	return func(_ int, _ *exec.Cmd, _ error) bool {
		return true
	}
}
