package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/grpclog"
)

var Log = zap.NewNop()

type zapGrpcLogger struct {
    logger *zap.Logger
}

func InitLogger(serviceName string, env string) {
	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.Encoding = "json"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		// config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var err error
	Log, err = config.Build(zap.Fields(
		zap.String("service", serviceName),
	))

	if err != nil {
		panic(err)
	}

	grpclog.SetLoggerV2(&zapGrpcLogger{logger: Log})
}

func ForContext(ctx context.Context) *zap.Logger {
    if ctx == nil {
        return Log
    }
    
    if id, ok := ctx.Value("correlation_id").(string); ok && id != "" {
        return Log.With(zap.String("correlation_id", id))
    }
    
    return Log
}

func (l *zapGrpcLogger) Info(args ...interface{})                    { /* l.logger.Sugar().Info(args...) */}
func (l *zapGrpcLogger) Infoln(args ...interface{})                  { /* l.logger.Sugar().Info(args...) */}
func (l *zapGrpcLogger) Infof(format string, args ...interface{})    { /* l.logger.Sugar().Infof(format, args...) */}
func (l *zapGrpcLogger) Warning(args ...interface{})                 { l.logger.Sugar().Warn(args...) }
func (l *zapGrpcLogger) Warningln(args ...interface{})               { l.logger.Sugar().Warn(args...) }
func (l *zapGrpcLogger) Warningf(format string, args ...interface{}) { l.logger.Sugar().Warnf(format, args...) }
func (l *zapGrpcLogger) Error(args ...interface{})                   { l.logger.Sugar().Error(args...) }
func (l *zapGrpcLogger) Errorln(args ...interface{})                 { l.logger.Sugar().Error(args...) }
func (l *zapGrpcLogger) Errorf(format string, args ...interface{})   { l.logger.Sugar().Errorf(format, args...) }
func (l *zapGrpcLogger) Fatal(args ...interface{})                   { l.logger.Sugar().Fatal(args...) }
func (l *zapGrpcLogger) Fatalln(args ...interface{})                 { l.logger.Sugar().Fatal(args...) }
func (l *zapGrpcLogger) Fatalf(format string, args ...interface{})   { l.logger.Sugar().Fatalf(format, args...) }
func (l *zapGrpcLogger) V(v int) bool                                { return false }