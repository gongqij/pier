package http

import "github.com/sirupsen/logrus"

// 由于使用 msp 相关仓库需要传入的logger接口与pier仓库使用的log接口不一样，因此需要包装一层

type loggerWrapper struct {
	logger logrus.FieldLogger
}

func (lw *loggerWrapper) Debug(v ...interface{}) {
	lw.logger.Debug(v)
}

func (lw *loggerWrapper) Debugf(format string, v ...interface{}) {
	lw.logger.Debugf(format, v)
}

func (lw *loggerWrapper) Info(v ...interface{}) {
	lw.logger.Info(v)
}

func (lw *loggerWrapper) Infof(format string, v ...interface{}) {
	lw.logger.Infof(format, v)
}

func (lw *loggerWrapper) Notice(v ...interface{}) {
	lw.logger.Info(v)
}

func (lw *loggerWrapper) Noticef(format string, v ...interface{}) {
	lw.logger.Infof(format, v)
}

func (lw *loggerWrapper) Warning(v ...interface{}) {
	lw.logger.Warning(v)
}

func (lw *loggerWrapper) Warningf(format string, v ...interface{}) {
	lw.logger.Warningf(format, v)
}

func (lw *loggerWrapper) Error(v ...interface{}) {
	lw.logger.Error(v)
}

func (lw *loggerWrapper) Errorf(format string, v ...interface{}) {
	lw.logger.Errorf(format, v)
}

func (lw *loggerWrapper) Critical(v ...interface{}) {
	lw.logger.Info(v)
}

func (lw *loggerWrapper) Criticalf(format string, v ...interface{}) {
	lw.logger.Infof(format, v)
}
