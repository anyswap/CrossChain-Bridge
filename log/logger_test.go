package log

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	now = time.Now().Unix()
	err = fmt.Errorf("error message")
)

// Fatal Fatalf Fatalln is not test
func TestLogger(t *testing.T) {
	SetLogger(6, false, true)

	WithFields("timestamp", now, "err", err).Tracef("test WithFields Tracef at %v", now)
	WithFields("timestamp", now, "err", err).Debugf("test WithFields Debugf at %v", now)
	WithFields("timestamp", now, "err", err).Infof("test WithFields Infof at %v", now)
	WithFields("timestamp", now, "err", err).Printf("test WithFields Printf at %v", now)
	WithFields("timestamp", now, "err", err).Warnf("test WithFields Warnf at %v", now)
	WithFields("timestamp", now, "err", err).Errorf("test WithFields Errorf at %v", now)
	assert.Panics(t, func() { WithFields("timestamp", now, "err", err).Panicf("test WithFields Panicf at %v", now) }, "not panic")

	Trace("test Trace", "timestamp", now, "err", err)
	Tracef("test Tracef, timestamp=%v err=%v", now, err)
	Traceln("test Traceln", "timestamp", now, "err", err)

	Debug("test Debug", "timestamp", now, "err", err)
	Debugf("test Debugf, timestamp=%v err=%v", now, err)
	Debugln("test Debugln", "timestamp", now, "err", err)

	Info("test Info", "timestamp", now, "err", err)
	Infof("test Infof, timestamp=%v err=%v", now, err)
	Infoln("test Infoln", "timestamp", now, "err", err)

	Print("test Print ", "timestamp", now, " err ", err)
	Printf("test Printf, timestamp=%v err=%v", now, err)
	Println("test Println", "timestamp", now, "err", err)

	Warn("test Warn", "timestamp", now, "err", err)
	Warnf("test Warnf, timestamp=%v err=%v", now, err)
	Warnln("test Warnln", "timestamp", now, "err", err)

	Error("test Error", "timestamp", now, "err", err)
	Errorf("test Errorf, timestamp=%v err=%v", now, err)
	Errorln("test Errorln", "timestamp", now, "err", err)

	assert.Panics(t, func() { Panic("test Panic", "timestamp", now, "err", err) }, "not panic")
	assert.Panics(t, func() { Panicf("test Panicf, timestamp=%v err=%v", now, err) }, "not panic")
	assert.Panics(t, func() { Panicln("test Panicln", "timestamp", now, "err", err) }, "not panic")
}
