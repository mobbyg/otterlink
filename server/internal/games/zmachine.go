package games

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type ZMachineRuntime struct {
	binary string
	stdbuf string
}

func NewZMachineRuntime() *ZMachineRuntime {
	binary := strings.TrimSpace(os.Getenv("OTTERLINK_DFROTZ"))
	if binary != "" {
		if resolved, err := exec.LookPath(binary); err == nil {
			binary = resolved
		} else {
			binary = ""
		}
	}
	if binary == "" {
		binary, _ = exec.LookPath("dfrotz")
	}
	if binary == "" {
		if info, err := os.Stat("/usr/games/dfrotz"); err == nil && info.Mode().IsRegular() && info.Mode()&0111 != 0 {
			binary = "/usr/games/dfrotz"
		}
	}
	stdbuf, _ := exec.LookPath("stdbuf")
	return &ZMachineRuntime{binary: binary, stdbuf: stdbuf}
}

func (r *ZMachineRuntime) Name() string { return "Z-machine (Frotz)" }

func (r *ZMachineRuntime) Available() bool {
	return r != nil && r.binary != ""
}

func (r *ZMachineRuntime) Start(parent context.Context, storyPath, saveDir string) (Process, error) {
	if !r.Available() {
		return nil, ErrRuntimeUnavailable
	}
	ctx, cancel := context.WithCancel(parent)
	args := []string{"-m", "-q", "-R", saveDir, storyPath}
	var cmd *exec.Cmd
	if r.stdbuf != "" {
		wrapped := append([]string{"-o0", "-e0", r.binary}, args...)
		cmd = exec.CommandContext(ctx, r.stdbuf, wrapped...)
	} else {
		cmd = exec.CommandContext(ctx, r.binary, args...)
	}
	cmd.Dir = saveDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}

	process := &zMachineProcess{
		cmd: cmd, stdin: stdin, cancel: cancel,
		output: make(chan string, 64), done: make(chan error, 1),
	}
	var readers sync.WaitGroup
	readers.Add(2)
	go pumpOutput(stdout, process.output, &readers)
	go pumpOutput(stderr, process.output, &readers)
	go func() {
		readers.Wait()
		close(process.output)
		err := cmd.Wait()
		process.done <- err
		close(process.done)
	}()
	return process, nil
}

type zMachineProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	cancel context.CancelFunc
	output chan string
	done   chan error
	writeMu sync.Mutex
}

func (p *zMachineProcess) WriteLine(line string) error {
	select {
	case <-p.done:
		return ErrSessionEnded
	default:
	}
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	_, err := io.WriteString(p.stdin, line+"\n")
	if err != nil {
		return err
	}
	return nil
}

func (p *zMachineProcess) Output() <-chan string { return p.output }
func (p *zMachineProcess) Done() <-chan error    { return p.done }

func (p *zMachineProcess) Stop() error {
	p.cancel()
	_ = p.stdin.Close()
	if p.cmd.Process == nil {
		return nil
	}
	return nil
}

func pumpOutput(reader io.Reader, output chan<- string, readers *sync.WaitGroup) {
	defer readers.Done()
	buffer := make([]byte, 4096)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			chunk := string(append([]byte(nil), buffer[:n]...))
			output <- chunk
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				output <- "\n[Interpreter output error: " + strings.TrimSpace(err.Error()) + "]\n"
			}
			return
		}
	}
}
