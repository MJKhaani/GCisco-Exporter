package ssh

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/MJKhaani/GCiscoExporter/internal/config"
	"golang.org/x/crypto/ssh"
)

type Client struct {
	host         string
	client       *ssh.Client
	stdin        io.WriteCloser
	stdout       io.Reader
	session      *ssh.Session
	timeout      time.Duration
	clientConfig *ssh.ClientConfig
}

func NewClient(device *config.Device, cred *config.Credential, timeout time.Duration, legacyCiphers bool) (*Client, error) {
	sshConfig := &ssh.ClientConfig{
		User:            cred.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	if legacyCiphers {
		sshConfig.SetDefaults()
		sshConfig.Ciphers = append(sshConfig.Ciphers, "aes128-cbc", "3des-cbc")
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

	if os.Getenv("GODEBUG") == "" {
		os.Setenv("GODEBUG", "rsa1024minkeys=0")
	}

	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH dial %s: %w", addr, err)
	}

	c := &Client{
		host:         addr,
		client:       conn,
		timeout:      timeout,
		clientConfig: sshConfig,
	}

	if err := c.setupSession(); err != nil {
		conn.Close()
		return nil, err
	}

	return c, nil
}

func (c *Client) setupSession() error {
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("creating SSH session: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		return fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return fmt.Errorf("stdout pipe: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:  0,
		ssh.OCRNL: 0,
	}

	if err := session.RequestPty("vt100", 0, 2000, modes); err != nil {
		session.Close()
		return fmt.Errorf("request pty: %w", err)
	}

	if err := session.Shell(); err != nil {
		session.Close()
		return fmt.Errorf("shell: %w", err)
	}

	c.stdin = stdin
	c.stdout = stdout
	c.session = session

	c.runRaw("")
	c.runRaw("terminal length 0")

	time.Sleep(500 * time.Millisecond)

	return nil
}

func (c *Client) runRaw(cmd string) {
	io.WriteString(c.stdin, cmd+"\n")
}

func (c *Client) Execute(ctx context.Context, command string) (string, error) {
	outputCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		output, err := c.readOutput(command)
		if err != nil {
			errCh <- err
			return
		}
		outputCh <- output
	}()

	c.runRaw(command)

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-errCh:
		return "", fmt.Errorf("command %q failed: %w", command, err)
	case output := <-outputCh:
		return output, nil
	case <-time.After(c.timeout):
		return "", fmt.Errorf("command %q timed out after %v", command, c.timeout)
	}
}

func (c *Client) readOutput(cmd string) (string, error) {
	promptRe := regexp.MustCompile(`.+#\s?$`)
	var buf bytes.Buffer
	tmp := make([]byte, 4096)

	for {
		n, err := c.stdout.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
			s := buf.String()
			if strings.Contains(s, cmd) && promptRe.MatchString(s) {
				return cleanOutput(s, cmd), nil
			}
		}
		if err != nil {
			s := buf.String()
			if strings.Contains(s, cmd) && promptRe.MatchString(s) {
				return cleanOutput(s, cmd), nil
			}
			return "", err
		}
	}
}

func cleanOutput(s, cmd string) string {
	s = strings.ReplaceAll(s, "\r", "")
	lines := strings.Split(s, "\n")
	var clean []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == cmd {
			continue
		}
		if strings.HasSuffix(trimmed, "#") || strings.Contains(trimmed, "# ") {
			continue
		}
		clean = append(clean, line)
	}
	return strings.Join(clean, "\n")
}

func (c *Client) Close() {
	if c.client == nil {
		return
	}
	if c.session != nil {
		c.session.Close()
	}
	c.client.Close()
}

func readKey(path string) (ssh.Signer, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(key)
}
