// Package log is a wrapper of logrus.
package log

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
)

const timestampFormat = "2006-01-02T15:04:05.000"

// JSONFormat print log in json format
var JSONFormat bool

// SetLogger set log level and format etc
func SetLogger(logLevel uint32, jsonFormat, colorFormat bool) {
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.Level(logLevel))
	JSONFormat = jsonFormat
	if JSONFormat {
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: timestampFormat,
		})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			ForceColors:     colorFormat,
			DisableColors:   !colorFormat,
			ForceQuote:      true,
			FullTimestamp:   true,
			TimestampFormat: timestampFormat,
			DisableSorting:  false,
		})
	}
}

// SetLogFile set log file path and rotation
func SetLogFile(logFile string, logRotation, logMaxAge uint64) {
	if logFile == "" {
		return
	}
	// always write in json format to log file
	if !JSONFormat {
		JSONFormat = true
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: timestampFormat,
		})
	}
	var (
		logRotateSuffix = "%Y%m%d%H"
		logRotationTime = time.Duration(logRotation) * time.Hour
		logMaxAgeTime   = time.Duration(logMaxAge) * time.Hour
	)
	logFile, _ = filepath.Abs(logFile)
	writer, err := rotatelogs.New(
		fmt.Sprintf("%s.%s", logFile, logRotateSuffix),
		rotatelogs.WithLinkName(logFile),
		rotatelogs.WithMaxAge(logMaxAgeTime),
		rotatelogs.WithRotationTime(logRotationTime),
	)
	if err != nil {
		logrus.Fatalf("Failed to Initialize Log File %s", err)
	}
	logrus.SetOutput(writer)
}

// WithFields encapsulate logrus.WithFields
func WithFields(ctx ...interface{}) *logrus.Entry {
	length := len(ctx)
	if length%2 != 0 {
		Debugf("log fileds number %v is not even", length)
	}
	fields := make(logrus.Fields)
	for k := 0; k+2 <= length; k += 2 {
		key, ok := ctx[k].(string)
		if ok {
			fields[key] = ctx[k+1]
		} else {
			Debugf("log field key '%v' is not string", ctx[k])
		}
	}
	return logrus.WithFields(fields)
}

// PrintFunc print function prototype
type PrintFunc func(msg string, ctx ...interface{})

// GetPrintFuncOr get log func of default
func GetPrintFuncOr(predicate func() bool, targetFunc, otherFunc PrintFunc) PrintFunc {
	if predicate() {
		return targetFunc
	}
	return otherFunc
}

// Null don't output anything
func Null(string, ...interface{}) {
}

// Trace trace
func Trace(msg string, ctx ...interface{}) {
	WithFields(ctx...).Trace(msg)
}

// Tracef tracef
func Tracef(format string, args ...interface{}) {
	logrus.Tracef(format, args...)
}

// Traceln traceln
func Traceln(msg string, ctx ...interface{}) {
	WithFields(ctx...).Traceln(msg)
}

// Debug debug
func Debug(msg string, ctx ...interface{}) {
	WithFields(ctx...).Debug(msg)
}

// Debugf debugf
func Debugf(format string, args ...interface{}) {
	logrus.Debugf(format, args...)
}

// Debugln debugln
func Debugln(msg string, ctx ...interface{}) {
	WithFields(ctx...).Debugln(msg)
}

// Info info
func Info(msg string, ctx ...interface{}) {
	WithFields(ctx...).Info(msg)
}

// Infof infof
func Infof(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}

// Infoln infoln
func Infoln(msg string, ctx ...interface{}) {
	WithFields(ctx...).Infoln(msg)
}

// Print print
func Print(msg ...interface{}) {
	logrus.Print(msg...)
}

// Printf printf
func Printf(format string, args ...interface{}) {
	logrus.Printf(format, args...)
}

// Println println
func Println(msg ...interface{}) {
	logrus.Println(msg...)
}

// Warn warn
func Warn(msg string, ctx ...interface{}) {
	WithFields(ctx...).Warn(msg)
}

// Warnf warnf
func Warnf(format string, args ...interface{}) {
	logrus.Warnf(format, args...)
}

// Warnln warnln
func Warnln(msg string, ctx ...interface{}) {
	WithFields(ctx...).Warnln(msg)
}

// Error error
func Error(msg string, ctx ...interface{}) {
	WithFields(ctx...).Error(msg)
}

// Errorf errorf
func Errorf(format string, args ...interface{}) {
	logrus.Errorf(format, args...)
}

// Errorln errorln
func Errorln(msg string, ctx ...interface{}) {
	WithFields(ctx...).Errorln(msg)
}

// Fatal fatal
func Fatal(msg string, ctx ...interface{}) {
	WithFields(ctx...).Fatal(msg)
}

// Fatalf fatalf
func Fatalf(format string, args ...interface{}) {
	logrus.Fatalf(format, args...)
}

// Fatalln fatalln
func Fatalln(msg string, ctx ...interface{}) {
	WithFields(ctx...).Fatalln(msg)
}

// Crit alias of `Fatal`
func Crit(msg string, ctx ...interface{}) {
	Fatal(msg, ctx...)
}

// Critf alias of `Fatalf`
func Critf(format string, args ...interface{}) {
	Fatalf(format, args...)
}

// Critln alias of `Fatalln`
func Critln(msg string, ctx ...interface{}) {
	Fatalln(msg, ctx...)
}

// Panic panic
func Panic(msg string, ctx ...interface{}) {
	WithFields(ctx...).Panic(msg)
}

// Panicf panicf
func Panicf(format string, args ...interface{}) {
	logrus.Panicf(format, args...)
}

// Panicln panicln
func Panicln(msg string, ctx ...interface{}) {
	WithFields(ctx...).Panicln(msg)
}
