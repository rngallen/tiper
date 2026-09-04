package logs

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// These thin wrappers expose the process-global zerolog logger configured by
// InitZerolog so callers don't import rs/zerolog directly everywhere.

const callerSkip = 1 // skip this wrapper so file:line is the call site

// Debug logs at debug level (variadic args formatted like fmt.Sprint).
func Debug(args ...any) {
	log.Debug().Msg(formatArgs(args...))
}

// Debugf logs at debug level with formatting. Callers that emit sensitive
// payloads must still gate on DFMS.DEBUG themselves.
func Debugf(format string, args ...any) {
	log.Debug().Msgf(format, args...)
}

// Info logs at info level (variadic args formatted like fmt.Sprint).
func Info(args ...any) {
	log.Info().Msg(formatArgs(args...))
}

// Infof logs at info level with formatting.
func Infof(format string, args ...any) {
	log.Info().Msgf(format, args...)
}

// Warn logs at warn level.
func Warn(args ...any) {
	log.Warn().Msg(formatArgs(args...))
}

// Warnf logs at warn level with formatting.
func Warnf(format string, args ...any) {
	log.Warn().Msgf(format, args...)
}

// Error logs at error level and records the caller's file:line (same idea as
// Fiber's access logger, but for application code). One extra stack frame is
// cheap; the path is a Go source location, not a user secret.
func Error(args ...any) {
	e := log.Error().Caller(callerSkip)
	if len(args) == 1 {
		if err, ok := args[0].(error); ok {
			e.Err(err).Send()
			return
		}
	}
	e.Msg(formatArgs(args...))
}

// Errorf logs at error level with formatting and the caller's file:line.
func Errorf(format string, args ...any) {
	log.Error().Caller(callerSkip).Msgf(format, args...)
}

func formatArgs(args ...any) string {
	if len(args) == 0 {
		return ""
	}
	if len(args) == 1 {
		if s, ok := args[0].(string); ok {
			return s
		}
	}
	return fmt.Sprint(args...)
}
