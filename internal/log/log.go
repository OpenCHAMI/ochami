// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/term"

	"github.com/openchami/ochami/internal/version"
)

// NewDefault returns the logger used before logging is configured: plain
// "<prog>: <message>" lines (plus any fields) written to w at warning level.
func NewDefault(w io.Writer) zerolog.Logger {
	cw := zerolog.ConsoleWriter{
		Out:        w,
		NoColor:    true,
		PartsOrder: []string{zerolog.LevelFieldName, zerolog.MessageFieldName},
		FormatLevel: func(interface{}) string {
			return version.ProgName + ":"
		},
	}
	return zerolog.New(cw).Level(zerolog.WarnLevel)
}

// New constructs a logger that writes to writer. The returned logger is an
// independent value and can safely be owned by a single CLI invocation.
func New(writer io.Writer, ll, lf, lc string) (zerolog.Logger, error) {
	if writer == nil {
		writer = io.Discard
	}

	var loggerLevel zerolog.Level
	switch ll {
	case "error":
		loggerLevel = zerolog.ErrorLevel
	case "warning":
		loggerLevel = zerolog.WarnLevel
	case "info":
		loggerLevel = zerolog.InfoLevel
	case "debug":
		loggerLevel = zerolog.DebugLevel
	default:
		return zerolog.Logger{}, fmt.Errorf("unknown log level: %s", ll)
	}

	cw := zerolog.ConsoleWriter{Out: writer}

	switch lc {
	case "", "auto":
		file, ok := writer.(*os.File)
		cw.NoColor = !ok || !term.IsTerminal(int(file.Fd()))
	case "on":
		cw.NoColor = false
	case "off":
		cw.NoColor = true
	default:
		return zerolog.Logger{}, fmt.Errorf("invalid log-color: %s", lc)
	}

	switch lf {
	case "rfc3339":
		cw.TimeFormat = time.RFC3339
		cw.FormatCaller = getFormatCaller(cw.NoColor)
		return zerolog.New(cw).Level(loggerLevel).With().Timestamp().Caller().Logger(), nil
	case "basic":
		cw.FormatTimestamp = func(i interface{}) string { return "" }
		cw.FormatLevel = func(i interface{}) string { return strings.ToUpper(fmt.Sprintf("%-6s|", i)) }
		cw.FormatCaller = getFormatCaller(cw.NoColor)
		return zerolog.New(cw).Level(loggerLevel).With().Caller().Logger(), nil
	case "json":
		return zerolog.New(cw).Level(loggerLevel).With().Timestamp().Logger(), nil
	default:
		return zerolog.Logger{}, fmt.Errorf("unknown log format: %s", lf)
	}
}

// getFormatCaller is a wrapper that generates a Formatter for the
// ConsoleWriter.FormatCaller field. The Formatter generated uses the base name
// of the source file where the log message originated from and ensures that it
// is still colorized, if enabled.
func getFormatCaller(noColor bool) zerolog.Formatter {
	return func(i interface{}) string {
		re := regexp.MustCompile(`(?P<path>.*):(?P<line>\d+)`)
		path := re.ReplaceAllString(i.(string), "${path}")
		line := re.ReplaceAllString(i.(string), "${line}")

		var out string
		_, f, l, ok := runtime.Caller(7)
		if ok {
			out = fmt.Sprintf("%s:%d", filepath.Base(f), l)
		} else {
			out = fmt.Sprintf("%s:%s", path, line)
		}

		return colorize(out, colorBold, noColor) + colorize(" >", colorCyan, noColor)
	}
}

// BasicLogger stores an io.Writer and a prefix for early logging. The io.Writer
// is where the earlyLog functions will write to and prefix is an optional
// prefix to use in log messages. This abstraction exists to make unit testing
// earlyLog functions easier.
type BasicLogger struct {
	// Since logging isn't set up until after config is read, this variable
	// allows more verbose printing if true for more verbose logging
	// pre-config parsing.
	EarlyVerbose bool
	out          io.Writer
	prefix       string
}

// NewBasicLogger creates a new BasicLogger with the specified io.Writer and
// prefix string (which can ge left blank to disable the prefix).
func NewBasicLogger(out io.Writer, on bool, prefix string) BasicLogger {
	return BasicLogger{
		EarlyVerbose: on,
		out:          out,
		prefix:       prefix,
	}
}

// BasicLog writes a string to the BasicLogger's io.Writer, prepending its
// prefix (e.g. "prefix: <msg>") if it is not empty.
func (el BasicLogger) BasicLog(arg ...interface{}) {
	if el.EarlyVerbose {
		if strings.Trim(el.prefix, " ") != "" {
			fmt.Fprintf(el.out, "%s: ", el.prefix)
		}
		fmt.Fprintln(el.out, arg...)
	}
}

// BasicLogf is like BasicLogger.earlyLof except that it behaves like Printf in
// that it accepts a format string.
func (el BasicLogger) BasicLogf(fstr string, arg ...interface{}) {
	if el.EarlyVerbose {
		if strings.Trim(el.prefix, " ") != "" {
			fmt.Fprintf(el.out, "%s: ", el.prefix)
		}
		fmt.Fprintf(el.out, fstr+"\n", arg...)
	}
}
