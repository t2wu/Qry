package qrylogger

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm/logger"
)

// ErrRecordNotFound record not found error
// var ErrRecordNotFound = errors.New("record not found")

// Colors
// const (
// 	Reset       = "\033[0m"
// 	Red         = "\033[31m"
// 	Green       = "\033[32m"
// 	Yellow      = "\033[33m"
// 	Blue        = "\033[34m"
// 	Magenta     = "\033[35m"
// 	Cyan        = "\033[36m"
// 	White       = "\033[37m"
// 	BlueBold    = "\033[34;1m"
// 	MagentaBold = "\033[35;1m"
// 	RedBold     = "\033[31;1m"
// 	YellowBold  = "\033[33;1m"
// )

// // LogLevel log level
// type LogLevel int

// const (
// 	// Silent silent log level
// 	Silent LogLevel = iota + 1
// 	// Error error log level
// 	Error
// 	// Warn warn log level
// 	Warn
// 	// Info info log level
// 	Info
// )

// Writer log writer interface
// type Writer interface {
// 	Printf(string, ...interface{})
// }

// Config logger config
// type Config struct {
// 	SlowThreshold             time.Duration
// 	Colorful                  bool
// 	IgnoreRecordNotFoundError bool
// 	ParameterizedQueries      bool
// 	LogLevel                  LogLevel
// }

// Interface logger interface
// type Interface interface {
// 	LogMode(LogLevel) Interface
// 	Info(context.Context, string, ...interface{})
// 	Warn(context.Context, string, ...interface{})
// 	Error(context.Context, string, ...interface{})
// 	Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error)
// }

// var (
// 	// Discard Discard logger will print any log to io.Discard
// 	Discard = New(log.New(io.Discard, "", log.LstdFlags), logger.Config{})
// 	// Default Default logger
// 	Default = New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
// 		SlowThreshold:             200 * time.Millisecond,
// 		LogLevel:                  logger.Warn,
// 		IgnoreRecordNotFoundError: false,
// 		Colorful:                  true,
// 	})
// 	// Recorder Recorder logger records running SQL into a recorder instance
// 	Recorder = traceRecorder{Interface: Default, BeginAt: time.Now()}
// )

// New initialize logger
func New(writer logger.Writer, config logger.Config) logger.Interface {
	var (
		infoStr      = "%s\n[info] "
		warnStr      = "%s\n[warn] "
		errStr       = "%s\n[error] "
		traceStr     = "%s\n[%.3fms] [rows:%v] %s"
		traceWarnStr = "%s %s\n[%.3fms] [rows:%v] %s"
		traceErrStr  = "%s %s\n[%.3fms] [rows:%v] %s"
	)

	if config.Colorful {
		infoStr = logger.Green + "%s\n" + logger.Reset + logger.Green + "[info] " + logger.Reset
		warnStr = logger.BlueBold + "%s\n" + logger.Reset + logger.Magenta + "[warn] " + logger.Reset
		errStr = logger.Magenta + "%s\n" + logger.Reset + logger.Red + "[error] " + logger.Reset
		traceStr = logger.Green + "%s\n" + logger.Reset + logger.Yellow + "[%.3fms] " + logger.BlueBold + "[rows:%v]" + logger.Reset + " %s"
		traceWarnStr = logger.Green + "%s " + logger.Yellow + "%s\n" + logger.Reset + logger.RedBold + "[%.3fms] " + logger.Yellow + "[rows:%v]" + logger.Magenta + " %s" + logger.Reset
		traceErrStr = logger.RedBold + "%s " + logger.MagentaBold + "%s\n" + logger.Reset + logger.Yellow + "[%.3fms] " + logger.BlueBold + "[rows:%v]" + logger.Reset + " %s"
	}

	return &qrylogger{
		Writer:       writer,
		Config:       config,
		infoStr:      infoStr,
		warnStr:      warnStr,
		errStr:       errStr,
		traceStr:     traceStr,
		traceWarnStr: traceWarnStr,
		traceErrStr:  traceErrStr,
	}
}

type qrylogger struct {
	logger.Writer
	logger.Config
	infoStr, warnStr, errStr            string
	traceStr, traceErrStr, traceWarnStr string
}

// LogMode log mode
func (l *qrylogger) LogMode(level logger.LogLevel) logger.Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

// Info print info
func (l qrylogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		l.Printf(l.infoStr+msg, append([]interface{}{FileWithLineNum()}, data...)...)
	}
}

// Warn print warn messages
func (l qrylogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		l.Printf(l.warnStr+msg, append([]interface{}{FileWithLineNum()}, data...)...)
	}
}

// Error print error messages
func (l qrylogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		l.Printf(l.errStr+msg, append([]interface{}{FileWithLineNum()}, data...)...)
	}
}

// Trace print sql message
func (l qrylogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= logger.Error && (!errors.Is(err, logger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			l.Printf(l.traceErrStr, FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Printf(l.traceErrStr, FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= logger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			l.Printf(l.traceWarnStr, FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Printf(l.traceWarnStr, FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case l.LogLevel == logger.Info:
		sql, rows := fc()
		if rows == -1 {
			l.Printf(l.traceStr, FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Printf(l.traceStr, FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}

// Trace print sql message
func (l qrylogger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.Config.ParameterizedQueries {
		return sql, nil
	}
	return sql, params
}

// --------------
// The above are directly from Gorm's original logger (but removed stuffs we can still reference Gorm's original definition to),
// we want to change the
// position of the file reported to be one above qry module
// But the following code also takes a page out of Gorm source code and then modified

// FileWithLineNum return the file name and line number of the current file
func FileWithLineNum() string {
	// the second caller usually from gorm internal, so set i start from 2
	for i := 2; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && (!strings.HasPrefix(file, qrySrceDir) || !strings.HasPrefix(file, gormSrceDir) || strings.HasSuffix(file, "_test.go")) {
			return file + ":" + strconv.FormatInt(int64(line), 10)
		}
	}

	return ""
}

var qrySrceDir string
var gormSrceDir string

func init() {
	_, file, _, _ := runtime.Caller(0)
	// compatible solution to get gorm source directory with various operating systems
	qrySrceDir = qrySourceDir(file)
	gormSrceDir = gormSourceDir(file)
}

func qrySourceDir(file string) string {
	dir := filepath.Dir(file)
	dir = filepath.Dir(dir)

	s := filepath.Dir(dir)
	if filepath.Base(s) != "qry" {
		s = dir
	}
	return filepath.ToSlash(s) + "/"
}

func gormSourceDir(file string) string {
	dir := filepath.Dir(file)
	dir = filepath.Dir(dir)

	s := filepath.Dir(dir)
	if filepath.Base(s) != "gorm.io" {
		s = dir
	}
	return filepath.ToSlash(s) + "/"
}
