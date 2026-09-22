package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Record map[string]any
type Records []Record

func ExecuteArgs(c *Client, args []string) (Records, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("command is required")
	}
	for _, a := range args {
		if a == "|" || strings.Contains(a, "|") {
			var b strings.Builder
			for i, v := range args {
				if i > 0 {
					b.WriteByte(' ')
				}
				if strings.ContainsAny(v, " 	|\\\"'") {
					b.WriteByte('"')
					b.WriteString(strings.ReplaceAll(v, "\\", "\\\\"))
					b.WriteString(strings.ReplaceAll(v, """, "\\""))
					b.WriteByte('"')
				} else {
					b.WriteString(v)
				}
			}
			return ExecuteLine(c, b.String())
		}
	}
	return runCommand(c, args, nil, false)
}

func ExecuteLine(c *Client, line string) (Records, error) {
	parts, err := splitPipeline(line)
	if err != nil {
		return nil, err
	}
	var data Records
	for i, part := range parts {
		args, err := tokenize(part)
		if err != nil {
			return nil, err
		}
		if len(args) == 0 {
			return nil, fmt.Errorf("empty pipeline command")
		}
		data, err = runCommand(c, args, data, i > 0)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func splitPipeline(s string) ([]string, error) {
	var out []string
	start := 0
	var quote byte
	escape := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escape {
			escape = false
			continue
		}
		if ch == '\\' {
			escape = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}
		if ch == '|' {
			out = append(out, strings.TrimSpace(s[start:i]))
			start = i + 1
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	out = append(out, strings.TrimSpace(s[start:]))
	return out, nil
}

func tokenize(s string) ([]string, error) {
	var out []string
	var b strings.Builder
	var quote byte
	escape := false
	flush := func() {
		if b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
		}
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escape {
			b.WriteByte(ch)
			escape = false
			continue
		}
		if ch == '\\' && quote != '\'' {
			escape = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
			} else {
				b.WriteByte(ch)
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}
		if ch == ' ' || ch == '\t' {
			flush()
			continue
		}
		b.WriteByte(ch)
	}
	if escape {
		b.WriteByte('\\')
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	flush()
	return out, nil
}

func runCommand(c *Client, a []string, input Records, piped bool) (Records, error) {
	if piped {
		switch a[0] {
		case "where":
			return pipeWhere(a[1:], input)
		case "select":
			return pipeSelect(a[1:], input)
		case "sort":
			return pipeSort(a[1:], input)
		case "count":
			return Records{{"count": len(input)}}, nil
		case "head":
			return pipeLimit(a[1:], input, true)
		case "tail":
			return pipeLimit(a[1:], input, false)
		}
	}
	switch a[0] {
	case "help":
		return helpRecords(), nil
	case "clear":
		fmt.Print("\033[2J\033[H")
		return nil, nil
	case "users":
		return userCommand(c, a[1:])
	case "chat":
		return chatCommand(c, a[1:])
	case "events":
		return eventCommand(c, a[1:])
	case "audit":
		return auditCommand(c, a[1:])
	case "presence":
		return presenceCommand(c, a[1:])
	case "where", "select", "sort", "count", "head", "tail":
		return nil, fmt.Errorf("%s requires pipeline input", a[0])
	default:
		return nil, fmt.Errorf("unknown command %q; type help", a[0])
	}
}

func helpRecords() Records {
	return Records{
		{"command": "users list|get USER|update USER --display-name X --email X --status active|disabled --role user|admin|password USER|revoke USER|delete USER"},
		{"command": "chat list|create NAME --allow-ops|delete ID|role ID USER mod|op|user"},
		{"command": "events list --year YYYY --month MM|create|update ID|delete ID"},
		{"command": "audit list"},
		{"command": "presence list"},
		{"command": "pipeline", "examples": "users list | where status=active | select username,role; users list | count"},
		{"command": "shell", "examples": "help; clear; exit; quit"},
	}
}

func PrintOutput(r Records, asJSON bool) {
	if asJSON {
		b, _ := json.MarshalIndent(r, "", "  ")
		fmt.Println(string(b))
		return
	}
	if len(r) == 0 {
		return
	}
	keys := []string{}
	seen := map[string]bool{}
	for _, row := range r {
		for k := range row {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)
	width := map[string]int{}
	for _, k := range keys {
		width[k] = len(k)
	}
	for _, row := range r {
		for _, k := range keys {
			v := fmt.Sprint(row[k])
			if len(v) > width[k] {
				width[k] = len(v)
			}
		}
	}
	for i, k := range keys {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Printf("%-*s", width[k], k)
	}
	fmt.Println()
	for i, k := range keys {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Printf("%-*s", width[k], strings.Repeat("-", width[k]))
	}
	fmt.Println()
	for _, row := range r {
		for i, k := range keys {
			if i > 0 {
				fmt.Print("  ")
			}
			fmt.Printf("%-*s", width[k], fmt.Sprint(row[k]))
		}
		fmt.Println()
	}
}

func value(row Record, key string) string {
	return fmt.Sprint(row[key])
}

func pipeWhere(args []string, in Records) (Records, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("usage: where field=value")
	}
	p := strings.SplitN(args[0], "=", 2)
	if len(p) != 2 {
		return nil, fmt.Errorf("usage: where field=value")
	}
	out := Records{}
	for _, r := range in {
		if strings.EqualFold(value(r, p[0]), p[1]) {
			out = append(out, r)
		}
	}
	return out, nil
}

func pipeSelect(args []string, in Records) (Records, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("usage: select field,field,...")
	}
	fields := strings.Split(args[0], ",")
	out := Records{}
	for _, r := range in {
		n := Record{}
		for _, f := range fields {
			f = strings.TrimSpace(f)
			if f != "" {
				n[f] = r[f]
			}
		}
		out = append(out, n)
	}
	return out, nil
}

func pipeSort(args []string, in Records) (Records, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("usage: sort field or sort -field")
	}
	field := args[0]
	desc := strings.HasPrefix(field, "-")
	field = strings.TrimPrefix(field, "-")
	out := append(Records(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := value(out[i], field), value(out[j], field)
		if desc {
			return a > b
		}
		return a < b
	})
	return out, nil
}

func pipeLimit(args []string, in Records, head bool) (Records, error) {
	n := 10
	if len(args) > 1 {
		return nil, fmt.Errorf("usage: head [N]")
	}
	if len(args) == 1 {
		v, e := strconv.Atoi(args[0])
		if e != nil || v < 0 {
			return nil, fmt.Errorf("invalid count %q", args[0])
		}
		n = v
	}
	if n > len(in) {
		n = len(in)
	}
	if head {
		return append(Records(nil), in[:n]...), nil
	}
	return append(Records(nil), in[len(in)-n:]...), nil
}
