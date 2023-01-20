package util

import (
	"log"
	"os"
)

type Level struct {
	level  uint8
	prefix string
	name   string
}

var (
	LevelTrace = Level{level: 50, prefix: "TRACE: ", name: "trace"}
	LevelDebug = Level{level: 40, prefix: "DEBUG: ", name: "debug"}
	LevelInfo  = Level{level: 30, prefix: "INFO:  ", name: "info"}
	LevelWarn  = Level{level: 20, prefix: "WARN:  ", name: "warn"}
	LevelError = Level{level: 10, prefix: "ERROR: ", name: "error"}
)

var logLevels = []*Level{
	&LevelTrace, &LevelDebug, &LevelInfo, &LevelWarn, &LevelError,
}

type Logger struct {
	logger *log.Logger
	level  Level
}

func (log *Logger) Trace(format string, v ...any) {
	log.logAtLevel(LevelTrace, format, v...)
}

func (log *Logger) Debug(format string, v ...any) {
	log.logAtLevel(LevelDebug, format, v...)
}

func (log *Logger) Info(format string, v ...any) {
	log.logAtLevel(LevelInfo, format, v...)
}

func (log *Logger) Warn(format string, v ...any) {
	log.logAtLevel(LevelWarn, format, v...)
}

func (log *Logger) Error(format string, v ...any) {
	log.logAtLevel(LevelError, format, v...)
}

func (log *Logger) SetLevel(levelName string) {
	level := LevelInfo
	for _, next := range logLevels {
		if next.name == levelName {
			level = *next
		}
	}
	log.level = level
	log.Info("Log Level set to: %v", level.name)
}

func (log *Logger) logAtLevel(level Level, messageFormat string, v ...any) {
	if log.levelEnabled(level) {
		log.logger.Printf(level.prefix+messageFormat+"\n", v...)
		// fmt.Printf(level.prefix+messageFormat+"\n", v...)
	}
}

func (log *Logger) levelEnabled(messageLevel Level) bool {
	return log.level.level >= messageLevel.level
}

var Log = &Logger{
	logger: log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile),
	level:  LevelInfo,
}
