package middleware

import (
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// maskedPrefixes hide the bearer values that travel in public URLs: a share
// token opens a link, a session id opens a stream, a ticket opens a file. They
// are credentials for as long as they live, so the request log must not keep a
// readable copy.
var maskedPrefixes = []struct {
	prefix      string
	placeholder string
}{
	{"/api/public/shares/", "[share]"},
	{"/api/public/images/", "[share]"},
	{"/api/public/stream/", "[session]"},
	{"/api/public/subtitles/", "[session]"},
	{"/api/public/downloads/", "[ticket]"},
}

// Logger is chi's request logger with public bearer tokens masked.
func Logger(next http.Handler) http.Handler {
	return chimiddleware.RequestLogger(maskingFormatter{
		// Same settings as chi's own default logger, so the log looks as before.
		next: &chimiddleware.DefaultLogFormatter{Logger: log.New(os.Stdout, "", log.LstdFlags), NoColor: runtime.GOOS == "windows"},
	})(next)
}

type maskingFormatter struct {
	next chimiddleware.LogFormatter
}

func (f maskingFormatter) NewLogEntry(r *http.Request) chimiddleware.LogEntry {
	if masked := maskSensitivePath(r.URL.Path); masked != r.URL.Path {
		// A shallow copy is enough: the formatter only reads the request line,
		// and the handler keeps the original.
		copied := *r
		copied.RequestURI = masked
		if r.URL.RawQuery != "" {
			copied.RequestURI += "?" + r.URL.RawQuery
		}
		return f.next.NewLogEntry(&copied)
	}
	return f.next.NewLogEntry(r)
}

func maskSensitivePath(path string) string {
	for _, entry := range maskedPrefixes {
		if !strings.HasPrefix(path, entry.prefix) {
			continue
		}
		rest := strings.TrimPrefix(path, entry.prefix)
		tail := ""
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			tail = rest[i:]
		}
		return entry.prefix + entry.placeholder + tail
	}
	return path
}
