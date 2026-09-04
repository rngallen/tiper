package logs

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

var (
	dbLogMu sync.RWMutex
	dbLog   zerolog.Logger
	dbReady bool
)

// InitDBZerolog configures the GORM/SQL logger (logs/db.log). Call together
// with InitZerolog so SQL lines are JSON zerolog, not stdlib text on stderr.
func InitDBZerolog(w io.Writer) {
	if w == nil {
		w = os.Stderr
	}
	l := zerolog.New(w).With().
		Timestamp().
		Str("service", serviceName).
		Str("log_type", "db").
		Logger()
	dbLogMu.Lock()
	dbLog = l
	dbReady = true
	dbLogMu.Unlock()
}

// GormWriter satisfies gorm.io/gorm/logger.Writer (Printf) and emits JSON
// through the db zerolog logger.
type GormWriter struct{}

// Printf implements gorm logger.Writer.
func (GormWriter) Printf(format string, args ...any) {
	dbLogMu.RLock()
	l := dbLog
	ok := dbReady
	dbLogMu.RUnlock()
	if !ok {
		l = zerolog.New(os.Stderr).With().
			Timestamp().
			Str("service", serviceName).
			Str("log_type", "db").
			Logger()
	}
	msg := strings.TrimRight(fmt.Sprintf(format, args...), "\r\n")
	l.Info().Msg(msg)
}
