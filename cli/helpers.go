package cli

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strings"
)

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
