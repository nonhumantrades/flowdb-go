package cli

import (
	"fmt"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
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
