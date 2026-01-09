package cli

import (
	"fmt"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

var relativeTimeRegex = regexp.MustCompile(`^-(\d+)([smhdw])$`)

func getCredentialsPath() string {
	u, _ := user.Current()
	return filepath.Join(u.HomeDir, ".flowdbcli.json")
}

func (c *Cli) readLine(prompt string) (string, error) {
	fmt.Print(prompt)
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (c *Cli) promptWithDefault(label, def string) (string, error) {
	prompt := label
	if def != "" {
		prompt += fmt.Sprintf(" [%s]", def)
	}
	prompt += ": "

	line, err := c.readLine(prompt)
	if err != nil {
		return "", err
	}
	if line == "" {
		return def, nil
	}
	return line, nil
}

func (c *Cli) promptNonEmpty(label string) (string, error) {
	for {
		val, err := c.promptWithDefault(label, "")
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(val) == "" {
			fmt.Println("value is required")
			continue
		}
		return strings.TrimSpace(val), nil
	}
}

func maskKey(k string) string {
	if k == "" {
		return ""
	}
	if len(k) <= 4 {
		return k
	}
	if len(k) <= 8 {
		return k[:2] + "..."
	}
	return k[:4] + "..." + k[len(k)-4:]
}

func (c *Cli) findS3ProfileIndex(name string) int {
	for i, p := range c.state.S3Profiles {
		if strings.EqualFold(p.Name, name) {
			return i
		}
	}
	return -1
}

// parseTime parses time strings in three formats:
// - Unix timestamp (seconds): "1704067200"
// - ISO 8601: "2024-01-01T00:00:00Z" or "2024-01-01"
// - Relative: "now", "-1h", "-30m", "-7d", "-2w"
// Empty string returns zero time.
func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}

	// Handle "now"
	if s == "now" {
		return time.Now().UTC(), nil
	}

	// Handle relative time (-Ns, -Nm, -Nh, -Nd, -Nw)
	if matches := relativeTimeRegex.FindStringSubmatch(s); matches != nil {
		n, _ := strconv.Atoi(matches[1])
		var d time.Duration
		switch matches[2] {
		case "s":
			d = time.Duration(n) * time.Second
		case "m":
			d = time.Duration(n) * time.Minute
		case "h":
			d = time.Duration(n) * time.Hour
		case "d":
			d = time.Duration(n) * 24 * time.Hour
		case "w":
			d = time.Duration(n) * 7 * 24 * time.Hour
		}
		return time.Now().UTC().Add(-d), nil
	}

	// Handle Unix timestamp (all digits)
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(ts, 0).UTC(), nil
	}

	// Handle ISO 8601 formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid time format: %q (use unix timestamp, ISO 8601, or relative like -1h, -7d, now)", s)
}

func formatBytes(b uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.1f TB", float64(b)/TB)
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/GB)
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/MB)
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/KB)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm %ds", m, s)
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh %dm", h, m)
	}
	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	if h == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd %dh", days, h)
}

func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return "-"
	}
	return ts.AsTime().UTC().Format("2006-01-02 15:04:05 UTC")
}

func formatTimestampShort(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return "-"
	}
	return ts.AsTime().UTC().Format("2006-01-02 15:04:05")
}

func formatNumber(n uint64) string {
	s := strconv.FormatUint(n, 10)
	if len(s) <= 3 {
		return s
	}

	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func printProgressBar(current, total uint64, suffix string) {
	width := 30
	var percent float64
	var filled int

	if total > 0 {
		percent = float64(current) / float64(total) * 100
		filled = int(float64(width) * float64(current) / float64(total))
	}

	bar := strings.Repeat("=", filled)
	if filled < width {
		bar += ">"
		bar += strings.Repeat(" ", width-filled-1)
	}

	fmt.Printf("\r[%s] %5.1f%% | %s / %s | %s",
		bar,
		percent,
		formatBytes(current),
		formatBytes(total),
		suffix,
	)
}

func clearProgressBar() {
	fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")
}
