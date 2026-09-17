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
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/term"

	"github.com/openchami/ochami/internal/version"
)

var (
	Logger = NewConcurrentLogger(zerolog.Nop())

	// A BasicLogger that is turned off until turned on by the
	// --verbose flag.
	EarlyLogger = NewBasicLogger(os.Stderr, false, version.ProgName)
)

// ConcurrentLogger provides race-safe replacement and access to the process
// logger. Each log event uses an immutable snapshot, so separate CLI
// invocations may initialize logging concurrently without racing with active
// commands.
type ConcurrentLogger struct {
	logger atomic.Pointer[zerolog.Logger]
}

// NewConcurrentLogger creates a logger initialized with logger.
func NewConcurrentLogger(logger zerolog.Logger) *ConcurrentLogger {
	cl := &ConcurrentLogger{}
	cl.Set(logger)
	return cl
}

// Set atomically replaces the logger used for future events.
func (cl *ConcurrentLogger) Set(logger zerolog.Logger) {
	cl.logger.Store(&logger)
}

// Get returns an immutable snapshot of the current logger.
func (cl *ConcurrentLogger) Get() zerolog.Logger {
	return *cl.logger.Load()
}

func (cl *ConcurrentLogger) Debug() *zerolog.Event {
	logger := cl.Get()
	return logger.Debug()
}

func (cl *ConcurrentLogger) Info() *zerolog.Event {
	logger := cl.Get()
	return logger.Info()
}

func (cl *ConcurrentLogger) Warn() *zerolog.Event {
	logger := cl.Get()
	return logger.Warn()
}

func (cl *ConcurrentLogger) Error() *zerolog.Event {
	logger := cl.Get()
	return logger.Error()
}

// Init() initializes the global logging object so it can be used for logging by
// any package that imports this internal log package.
func Init(ll, lf, lc string) error {
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
		return fmt.Errorf("unknown log level: %s", ll)
	}

	cw := zerolog.ConsoleWriter{Out: os.Stderr}

	switch lc {
	case "", "auto":
		cw.NoColor = !term.IsTerminal(int(os.Stderr.Fd()))
	case "on":
		cw.NoColor = false
	case "off":
		cw.NoColor = true
	default:
		return fmt.Errorf("invalid log-color: %s", lc)
	}

	switch lf {
	case "rfc3339":
		cw.TimeFormat = time.RFC3339
		cw.FormatCaller = getFormatCaller(cw.NoColor)
		Logger.Set(zerolog.New(cw).Level(loggerLevel).With().Timestamp().Caller().Logger())
	case "basic":
		cw.FormatTimestamp = func(i interface{}) string { return "" }
		cw.FormatLevel = func(i interface{}) string { return strings.ToUpper(fmt.Sprintf("%-6s|", i)) }
		cw.FormatCaller = getFormatCaller(cw.NoColor)
		Logger.Set(zerolog.New(cw).Level(loggerLevel).With().Caller().Logger())
	case "json":
		Logger.Set(zerolog.New(cw).Level(loggerLevel).With().Timestamp().Logger())
	default:
		return fmt.Errorf("unknown log format: %s", lf)
	}

	return nil
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
