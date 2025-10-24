package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Logger  *zap.Logger
	String  = zap.String
	Any     = zap.Any
	Int     = zap.Int
	Float32 = zap.Float32
)

func InitLogger(logpath string, loglevel string) {

	writer := getLogWriter(logpath) // 日志分割

	var level zapcore.Level

	switch loglevel {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "error":
		level = zap.ErrorLevel
	case "warn":
		level = zap.WarnLevel
	default:
		level = zap.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "linenum",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,  // 小写编码器
		EncodeTime:     zapcore.ISO8601TimeEncoder,     // ISO8601 UTC 时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder, //
		EncodeCaller:   zapcore.FullCallerEncoder,      // 全路径编码器
		EncodeName:     zapcore.FullNameEncoder,
	}

	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(level)

	var writes = []zapcore.WriteSyncer{writer}
	// 如果是开发环境，同时在控制台上也输出
	if level == zap.DebugLevel {
		writes = append(writes, zapcore.AddSync(os.Stdout))
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		// zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(writes...), // 打印到控制台和文件
		// write,
		level,
	)

	// 设置初始化字段,如：添加一个服务器名称
	filed := zap.Fields(zap.String("application", "MyGoChat"))
	// 构造日志
	Logger = zap.New(core, zap.AddCaller(), zap.Development(), filed)
	Logger.Info("Logger init success")
}

func getLogWriter(logpath string) zapcore.WriteSyncer {
	lumberJackLogger := lumberjack.Logger{
		Filename:   logpath,
		MaxSize:    100,  // 每个日志文件保存2M
		MaxBackups: 30,   // 保留30个备份
		MaxAge:     7,    // 保留7天
		Compress:   true, // 是否压缩 disabled by default
		LocalTime:  true,
	}

	return zapcore.AddSync(&lumberJackLogger)
}
