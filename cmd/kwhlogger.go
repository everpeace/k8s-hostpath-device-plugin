package cmd

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
)

var (
	_ kwhlog.Logger = &zerologKwhLogger{}
)

type zerologKwhLogger struct {
	zerolog.Logger
}

func newZerologKubeWebhookLogger(log zerolog.Logger) *zerologKwhLogger {
	return &zerologKwhLogger{Logger: log}
}

func (l *zerologKwhLogger) Infof(format string, args ...interface{}) {
	if e := l.Info(); e.Enabled() {
		e.CallerSkipFrame(1).Msg(fmt.Sprintf(format, args...))
	}
}

func (l *zerologKwhLogger) Warningf(format string, args ...interface{}) {
	if e := l.Warn(); e.Enabled() {
		e.CallerSkipFrame(1).Msg(fmt.Sprintf(format, args...))
	}
}
func (l *zerologKwhLogger) Errorf(format string, args ...interface{}) {
	if e := l.Error(); e.Enabled() {
		e.CallerSkipFrame(1).Msg(fmt.Sprintf(format, args...))
	}
}

func (l *zerologKwhLogger) Debugf(format string, args ...interface{}) {
	if e := l.Debug(); e.Enabled() {
		e.CallerSkipFrame(1).Msg(fmt.Sprintf(format, args...))
	}
}

func (l *zerologKwhLogger) WithValues(values map[string]interface{}) kwhlog.Logger {
	zlogger := l.Logger
	for k, v := range values {
		zlogger = zlogger.With().Interface(k, v).Logger()
	}
	return newZerologKubeWebhookLogger(zlogger)
}

func (l *zerologKwhLogger) WithCtxValues(ctx context.Context) kwhlog.Logger {
	return l.WithValues(kwhlog.ValuesFromCtx(ctx))
}

func (l *zerologKwhLogger) SetValuesOnCtx(parent context.Context, values map[string]interface{}) context.Context {
	return kwhlog.CtxWithValues(parent, values)
}
