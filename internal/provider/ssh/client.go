package ssh

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/MJKhaani/GCiscoExporter/internal/config"
	"golang.org/x/crypto/ssh"
)

type Client struct {
	conn    *ssh.Client
	device  *config.Device
	cred    *config.Credential
	timeout time.Duration
}

func NewClient(device *config.Device, cred *config.Credential, timeout time.Duration, legacyCiphers bool) (*Client, error) {
	sshConfig := &ssh.ClientConfig{
		User:            cred.Username,
		Timeout:         timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if legacyCiphers {
		sshConfig.Ciphers = append(sshConfig.Ciphers,
			"aes128-cbc", "aes192-cbc", "aes256-cbc",
			"3des-cbc", "aes128-ctr", "aes192-ctr", "aes256-ctr",
		)
		sshConfig.KeyExchanges = append(sshConfig.KeyExchanges,
			"diffie-hellman-group-exchange-sha1",
			"diffie-hellman-group14-sha1",
			"diffie-hellman-group1-sha1",
		)
	}

	if cred.KeyFile != "" {
		key, err := readKey(cred.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("reading SSH key: %w", err)
		}
		sshConfig.Auth = []ssh.AuthMethod{ssh.PublicKeys(key)}
	} else if cred.Password != "" {
		sshConfig.Auth = []ssh.AuthMethod{ssh.Password(cred.Password)}
	} else {
		return nil, fmt.Errorf("no authentication method provided")
	}

	addr := fmt.Sprintf("%s:%d", device.Host, device.Port)
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH dial %s: %w", addr, err)
	}

	return &Client{
		conn:    conn,
		device:  device,
		cred:    cred,
		timeout: timeout,
	}, nil
}

func (c *Client) Execute(ctx context.Context, command string) (string, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("creating SSH session: %w", err)
	}

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
		session.Close()
	}()

	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGTERM)
		session.Close()
		return "", ctx.Err()
	case err := <-done:
		if err != nil {
			errStr := stderr.String()
			if errStr != "" {
				return stdout.String(), fmt.Errorf("command %q failed: %w (stderr: %s)", command, err, errStr)
			}
			return stdout.String(), fmt.Errorf("command %q failed: %w", command, err)
		}
		return stdout.String(), nil
	}
}

func (c *Client) ExecuteAll(ctx context.Context, commands []string) (map[string]string, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("creating SSH session: %w", err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("vt100", 200, 500, modes); err != nil {
		return nil, fmt.Errorf("request pty: %w", err)
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	if err := session.Shell(); err != nil {
		return nil, fmt.Errorf("shell: %w", err)
	}

	results := make(map[string]string)
	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Split(bufio.ScanLines)

	prompt := "#"
	var outputBuf strings.Builder
	currentCmd := ""
	cmdIdx := 0
	cmdDone := make(chan bool, 1)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			outputBuf.WriteString(line + "\n")

			if strings.HasSuffix(strings.TrimSpace(line), prompt) || strings.Contains(line, prompt+" ") {
				if currentCmd != "" {
					output := outputBuf.String()
					output = cleanOutput(output, currentCmd, prompt)
					results[currentCmd] = output
					outputBuf.Reset()
					currentCmd = ""
					cmdDone <- true
				}
			}
		}
	}()

	time.Sleep(1 * time.Second)

	for cmdIdx < len(commands) {
		currentCmd = commands[cmdIdx]
		outputBuf.Reset()

		if _, err := fmt.Fprintf(stdinPipe, "%s\n", currentCmd); err != nil {
			return results, fmt.Errorf("writing command: %w", err)
		}

		select {
		case <-ctx.Done():
			return results, ctx.Err()
		case <-cmdDone:
			cmdIdx++
		case <-time.After(c.timeout):
			output := outputBuf.String()
			output = cleanOutput(output, currentCmd, prompt)
			results[currentCmd] = output
			cmdIdx++
		}
	}

	fmt.Fprintln(stdinPipe, "exit")
	time.Sleep(500 * time.Millisecond)
	session.Close()

	return results, nil
}

func cleanOutput(output, command, prompt string) string {
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.ReplaceAll(output, "\r", "\n")

	lines := strings.Split(output, "\n")
	var clean []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == command {
			continue
		}
		if strings.HasPrefix(trimmed, prompt) || strings.HasSuffix(trimmed, prompt) {
			continue
		}
		clean = append(clean, line)
	}
	return strings.Join(clean, "\n")
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func readKey(path string) (ssh.Signer, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(key)
}
