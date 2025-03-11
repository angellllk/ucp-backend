package service

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const (
	Off = iota
	Error
	Warn
	Info
	Debug
	Trace
)

type message struct {
	severity string
	string   string
	source   string
	logLevel int
}

type LoggerService struct {
	messages  chan message
	waitGroup *sync.WaitGroup
	logPath   string
	mu        *sync.Mutex
	version   string
}

var globalLogger *LoggerService

func NewLoggerService(path string, version string) (*LoggerService, error) {
	file, errLog := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if errLog != nil {
		return nil, errLog
	}

	l := LoggerService{
		messages:  make(chan message),
		waitGroup: &sync.WaitGroup{},
		logPath:   path,
		mu:        &sync.Mutex{},
		version:   "v" + version,
	}

	globalLogger = &l

	l.waitGroup.Add(1)
	go l.run(file)

	return &l, nil
}

func (l *LoggerService) run(file *os.File) {
	mw := io.MultiWriter(os.Stdout, file)
	InfoLogger := log.New(mw, "[INFO] ", log.Ldate|log.Ltime)
	WarningLogger := log.New(mw, "[WARNING] ", log.Ldate|log.Ltime)
	ErrorLogger := log.New(mw, "[ERROR] ", log.Ldate|log.Ltime)

	defer l.waitGroup.Done()
	defer file.Close()

	for msg := range l.messages {
		switch msg.logLevel {
		case 0:
			InfoLogger.Println(l.version + " " + msg.source + " " + msg.string)
		case 1:
			WarningLogger.Println(l.version + " " + msg.source + " " + msg.string)
		case 2:
			ErrorLogger.Println(l.version + " " + msg.source + " " + msg.string)
		}
	}
}

func (l *LoggerService) Shutdown() {
	close(l.messages)
	l.waitGroup.Wait()
}

func getModuleName() string {
	_, fileName, line, ok := runtime.Caller(2)
	if ok {
		s := fmt.Sprintf("%s:%d", fileName, line)
		return s
	}
	return "<unknown>"
}

func (l *LoggerService) SendLog(level int, tar string, mp string, f string, ln int, m string) {
	logMsg := fmt.Sprintf("Level %d Target %s, Modulel Path %s, File %s, Line %d -- Message %s", level, tar, mp, f, ln, m)
	switch level {
	case Debug:
		l.Debug(logMsg)
	case Error:
		l.Exception(logMsg)
	case Warn:
		l.Warning(logMsg)
	case Info, Trace:
		l.Info(logMsg)
	default:

	}
}

func (l *LoggerService) Debug(msg string) {
	l.messages <- message{
		severity: "debug",
		string:   msg,
		source:   getModuleName(),
		logLevel: 0,
	}
}

func (l *LoggerService) Info(msg string) {
	l.messages <- message{
		severity: "info",
		string:   msg,
		source:   getModuleName(),
		logLevel: 0,
	}
}

func (l *LoggerService) Warning(msg string) {
	l.messages <- message{
		severity: "warning",
		string:   msg,
		source:   getModuleName(),
		logLevel: 1,
	}
}

func (l *LoggerService) Exception(msg string) {
	l.messages <- message{
		severity: "exception",
		string:   msg,
		source:   getModuleName(),
		logLevel: 2,
	}
}

func (l *LoggerService) ClearOldLogs(retentionPeriod time.Duration) error {
	dir := filepath.Dir(l.logPath)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		normalizedPath, err := filepath.Abs(filepath.Clean(path))
		if err != nil {
			return fmt.Errorf("error normalizing path: %w", err)
		}
		normalizedLogPath, err := filepath.Abs(filepath.Clean(l.logPath))
		if err != nil {
			return fmt.Errorf("error normalizing log path: %w", err)
		}

		if normalizedPath == normalizedLogPath {
			return nil
		}

		if !info.IsDir() && filepath.Ext(info.Name()) == ".log" {
			if time.Since(info.ModTime()) > retentionPeriod {
				l.Info(fmt.Sprintf("Deleting old log: %s\n", path))
				return os.Remove(path)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error deleting old logs: %w", err)
	}

	return nil
}
