package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func main() {
	cfg := ConfigFromEnv()
	if err := cfg.ParseFlags(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "otterlink-admin:", err)
		os.Exit(2)
	}
	client := NewClient(cfg)
	if len(cfg.Command) > 0 {
		if err := client.EnsureAuth(false); err != nil {
			fmt.Fprintln(os.Stderr, "otterlink-admin:", err)
			os.Exit(1)
		}
		out, err := ExecuteArgs(client, cfg.Command)
		if err != nil {
			fmt.Fprintln(os.Stderr, "otterlink-admin:", err)
			os.Exit(1)
		}
		PrintOutput(out, cfg.JSON)
		return
	}
	fmt.Println("Otter Link administration shell")
	fmt.Println("Type 'help' for commands. Type 'quit' or 'exit' to leave.")
	if err := client.EnsureAuth(true); err != nil {
		fmt.Fprintln(os.Stderr, "otterlink-admin:", err)
		os.Exit(1)
	}
	for {
		line, err := readLine("otter> ")
		if err == io.EOF {
			fmt.Println()
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "otterlink-admin:", err)
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "quit" || line == "exit" {
			return
		}
		out, err := ExecuteLine(client, line)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			continue
		}
		PrintOutput(out, false)
	}
}

func readLine(prompt string) (string, error) {
	fmt.Print(prompt)

	// Put the terminal into character-at-a-time mode so the admin shell can
	// provide basic line editing instead of relying on the terminal's
	// canonical input handling.
	if err := exec.Command("stty", "-icanon", "-echo").Run(); err != nil {
		// Fall back to normal terminal input if stty is unavailable.
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("")
		v, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return "", readErr
		}
		return strings.TrimSuffix(strings.TrimSuffix(v, "\n"), "\r"), nil
	}
	defer func() {
		_ = exec.Command("stty", "icanon", "echo").Run()
	}()

	var line []byte
	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}

		switch buf[0] {
		case '\r', '\n':
			fmt.Println()
			return string(line), nil
		case 3: // Ctrl-C
			fmt.Println("^C")
			return "", nil
		case 4: // Ctrl-D
			return "", io.EOF
		case 8, 127: // Backspace / Delete
			if len(line) > 0 {
				line = line[:len(line)-1]
				fmt.Print("\b \b")
			}
		default:
			if buf[0] >= 32 {
				line = append(line, buf[0])
				fmt.Printf("%c", buf[0])
			}
		}
	}
}

type Config struct {
	Server   string
	Token    string
	Username string
	Password string
	JSON     bool
	Command  []string
}

func ConfigFromEnv() Config {
	server := os.Getenv("OTTERLINK_SERVER")
	if server == "" {
		server = "http://localhost:9090"
	}
	return Config{
		Server:   server,
		Token:    os.Getenv("OTTERLINK_TOKEN"),
		Username: os.Getenv("OTTERLINK_USERNAME"),
		Password: os.Getenv("OTTERLINK_PASSWORD"),
	}
}

func (c *Config) ParseFlags(args []string) error {
	for len(args) > 0 {
		switch args[0] {
		case "--server":
			if len(args) < 2 {
				return fmt.Errorf("--server requires a value")
			}
			c.Server, args = strings.TrimRight(args[1], "/"), args[2:]
		case "--token":
			if len(args) < 2 {
				return fmt.Errorf("--token requires a value")
			}
			c.Token, args = args[1], args[2:]
		case "--username":
			if len(args) < 2 {
				return fmt.Errorf("--username requires a value")
			}
			c.Username, args = args[1], args[2:]
		case "--json":
			c.JSON, args = true, args[1:]
		case "--help", "-h":
			PrintUsage()
			os.Exit(0)
		default:
			c.Command = args
			return nil
		}
	}
	return nil
}

func PrintUsage() {
	fmt.Println("Usage: otterlink-admin [global options] [command]")
	fmt.Println()
	fmt.Println("Global options:")
	fmt.Println("  --server URL       Otter Link HTTP server (default http://localhost:9090)")
	fmt.Println("  --token TOKEN      existing session token; prefer OTTERLINK_TOKEN")
	fmt.Println("  --username NAME    login username; prefer OTTERLINK_USERNAME")
	fmt.Println("  --json             print final command output as JSON")
	fmt.Println()
	fmt.Println("Run without a command for the interactive administration shell.")
}
