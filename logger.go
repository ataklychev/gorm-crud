package gorm_crud

type LoggerInterface interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Print(args ...interface{})
	Printf(format string, args ...interface{})
}

func NewLoggerWithMeta(logger LoggerInterface, meta ...interface{}) LoggerInterface {
	return &LoggerWithMeta{meta: meta, logger: logger}
}

type LoggerWithMeta struct {
	meta   []interface{}
	logger LoggerInterface
}

func (l *LoggerWithMeta) Debug(args ...interface{}) {
	l.logger.Debug(append(l.meta, args))
}

func (l *LoggerWithMeta) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, append(l.meta, args))
}

func (l *LoggerWithMeta) Info(args ...interface{}) {
	l.logger.Info(append(l.meta, args))
}

func (l *LoggerWithMeta) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, append(l.meta, args))
}

func (l *LoggerWithMeta) Warn(args ...interface{}) {
	l.logger.Warn(append(l.meta, args))
}

func (l *LoggerWithMeta) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, append(l.meta, args))
}

func (l *LoggerWithMeta) Error(args ...interface{}) {
	l.logger.Error(append(l.meta, args))
}

func (l *LoggerWithMeta) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, append(l.meta, args))
}

func (l *LoggerWithMeta) Fatal(args ...interface{}) {
	l.logger.Fatal(append(l.meta, args))
}

func (l *LoggerWithMeta) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, append(l.meta, args))
}

func (l *LoggerWithMeta) Print(args ...interface{}) {
	l.logger.Debug(append(l.meta, args))
}

func (l *LoggerWithMeta) Printf(format string, args ...interface{}) {
	l.logger.Debugf(format, append(l.meta, args))
}
