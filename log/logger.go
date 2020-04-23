package log

import (
	"os"

	"github.com/sirupsen/logrus"
)

const timestampFormat = "2006-01-02T15:04:05.000"

func SetLogger(logLevel uint32, jsonFormat, colorFormat bool) {
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.Level(logLevel))
	if jsonFormat {
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
			DisableSorting:  true,
		})
	}
}

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

func Trace(msg string, ctx ...interface{}) {
	WithFields(ctx...).Trace(msg)
}

func Tracef(format string, args ...interface{}) {
	logrus.Tracef(format, args...)
}

func Traceln(msg string, ctx ...interface{}) {
	WithFields(ctx...).Traceln(msg)
}

func Debug(msg string, ctx ...interface{}) {
	WithFields(ctx...).Debug(msg)
}

func Debugf(format string, args ...interface{}) {
	logrus.Debugf(format, args...)
}

func Debugln(msg string, ctx ...interface{}) {
	WithFields(ctx...).Debugln(msg)
}

func Info(msg string, ctx ...interface{}) {
	WithFields(ctx...).Info(msg)
}

func Infof(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}

func Infoln(msg string, ctx ...interface{}) {
	WithFields(ctx...).Infoln(msg)
}

func Print(msg ...interface{}) {
	logrus.Print(msg...)
}

func Printf(format string, args ...interface{}) {
	logrus.Printf(format, args...)
}

func Println(msg ...interface{}) {
	logrus.Println(msg...)
}

func Warn(msg string, ctx ...interface{}) {
	WithFields(ctx...).Warn(msg)
}

func Warnf(format string, args ...interface{}) {
	logrus.Warnf(format, args...)
}

func Warnln(msg string, ctx ...interface{}) {
	WithFields(ctx...).Warnln(msg)
}

func Error(msg string, ctx ...interface{}) {
	WithFields(ctx...).Error(msg)
}

func Errorf(format string, args ...interface{}) {
	logrus.Errorf(format, args...)
}

func Errorln(msg string, ctx ...interface{}) {
	WithFields(ctx...).Errorln(msg)
}

func Fatal(msg string, ctx ...interface{}) {
	WithFields(ctx...).Fatal(msg)
}

func Fatalf(format string, args ...interface{}) {
	logrus.Fatalf(format, args...)
}

func Fatalln(msg string, ctx ...interface{}) {
	WithFields(ctx...).Fatalln(msg)
}

// alias of `Fatal`
func Crit(msg string, ctx ...interface{}) {
	Fatal(msg, ctx...)
}

// alias of `Fatalf`
func Critf(format string, args ...interface{}) {
	Fatalf(format, args...)
}

// alias of `Fatalln`
func Critln(msg string, ctx ...interface{}) {
	Fatalln(msg, ctx...)
}

func Panic(msg string, ctx ...interface{}) {
	WithFields(ctx...).Panic(msg)
}

func Panicf(format string, args ...interface{}) {
	logrus.Panicf(format, args...)
}

func Panicln(msg string, ctx ...interface{}) {
	WithFields(ctx...).Panicln(msg)
}
