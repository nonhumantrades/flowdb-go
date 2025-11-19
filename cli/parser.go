package cli

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Command struct {
	Path []string
	New  func() any
}

type ParsedCommand struct {
	Def   *Command
	Value any
}

type Parser struct {
	commands []*Command
}

func NewParser() *Parser {
	return &Parser{commands: []*Command{}}
}

func (p *Parser) Register(cmd string, fn func() any) {
	parts := strings.Fields(strings.ToLower(cmd))
	if len(parts) == 0 {
		panic("empty command path")
	}
	p.commands = append(p.commands, &Command{
		Path: parts,
		New:  fn,
	})
}

func (p *Parser) RegisterMany(fn func() any, cmds ...string) {
	for _, cmd := range cmds {
		p.Register(cmd, fn)
	}
}

func (p *Parser) Parse(line string) (*ParsedCommand, error) {
	tokens := tokenize(line)
	if len(tokens) == 0 {
		return nil, errors.New("empty input")
	}

	var best *Command
	var bestLen int
	for _, def := range p.commands {
		if len(tokens) < len(def.Path) {
			continue
		}
		match := true
		for i := range def.Path {
			if strings.ToLower(tokens[i]) != def.Path[i] {
				match = false
				break
			}
		}
		if match && len(def.Path) > bestLen {
			best = def
			bestLen = len(def.Path)
		}
	}

	if best == nil {
		return nil, fmt.Errorf("unknown command: %s", line)
	}

	argTokens := tokens[bestLen:]
	kv := parseArgs(argTokens)

	cmdVal := best.New()
	if err := bindArgs(cmdVal, kv); err != nil {
		return nil, err
	}

	return &ParsedCommand{Def: best, Value: cmdVal}, nil
}

func tokenize(line string) []string {
	var tokens []string
	var cur strings.Builder
	inQuotes := false
	var quoteChar rune

	for _, r := range line {
		switch {
		case r == ' ' || r == '\t':
			if inQuotes {
				cur.WriteRune(r)
			} else if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		case r == '"' || r == '\'':
			if inQuotes {
				if r == quoteChar {
					inQuotes = false
				} else {
					cur.WriteRune(r)
				}
			} else {
				inQuotes = true
				quoteChar = r
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func parseArgs(tokens []string) map[string]string {
	m := make(map[string]string)
	for _, t := range tokens {
		if t == "" {
			continue
		}
		if strings.Contains(t, "=") {
			parts := strings.SplitN(t, "=", 2)
			key := strings.ToLower(parts[0])
			val := parts[1]
			val = strings.Trim(val, `"'`)
			m[key] = val
		} else {
			m[strings.ToLower(t)] = "true"
		}
	}
	return m
}

func bindArgs(target any, kv map[string]string) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target must be pointer to struct, got %T", target)
	}
	v = v.Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("cli")
		if tag == "" {
			tag = strings.ToLower(f.Name)
		} else {
			tag = strings.ToLower(tag)
		}

		raw, ok := kv[tag]
		if !ok {
			continue
		}

		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}

		switch fv.Kind() {
		case reflect.String:
			fv.SetString(raw)
		case reflect.Bool:
			b, err := strconv.ParseBool(raw)
			if err != nil {
				return fmt.Errorf("invalid bool for %s: %q", tag, raw)
			}
			fv.SetBool(b)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid int for %s: %q", tag, raw)
			}
			fv.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := strconv.ParseUint(raw, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid uint for %s: %q", tag, raw)
			}
			fv.SetUint(n)
		default:
		}
	}

	return nil
}
