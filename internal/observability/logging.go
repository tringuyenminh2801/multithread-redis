package observability

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// SetupLogger installs a JSON logger on stdout as the slog default.
// Docker captures stdout, and Alloy ships it to Loki.
func SetupLogger(level string) error {
	var l slog.Level

	if err := l.UnmarshalText([]byte(strings.ToUpper(level))); err != nil {
		return fmt.Errorf("invalid log level %q: %w", level, err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
	slog.SetDefault(logger)

	return nil
}
