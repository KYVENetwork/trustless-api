package utils

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

func TrustlessApiLogger(moduleName string) zerolog.Logger {
	writer := io.MultiWriter(os.Stdout)
	customConsoleWriter := zerolog.ConsoleWriter{Out: writer}
	customConsoleWriter.FormatCaller = func(i interface{}) string {
		return "\x1b[36m[API]\x1b[0m"
	}

	logger := zerolog.New(customConsoleWriter).With().Str("module", moduleName).Timestamp().Logger()
	return logger
}
