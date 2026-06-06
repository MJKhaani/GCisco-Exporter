package ssh

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/MJKhaani/GCiscoExporter/internal/config"
	"golang.org/x/crypto/ssh"
)

type Client struct {
	conn     *ssh.Client
	device   *config.Device
	cred     *config.Credential
	timeout  time.Duration
}

func NewClient(device *config.Device, cred *config.Credential, timeout time.Duration) (*Client, error) {
	sshConfig := &ssh.ClientConfig{
		User:            cred.Username,
		Timeout:         timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
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
	defer session.Close()

	done := make(chan struct{})
	var output []byte
	var execErr error

	go func() {
		output, execErr = session.Output(command)
		close(done)
	}()

	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGTERM)
		return "", ctx.Err()
	case <-done:
		if execErr != nil {
			return "", fmt.Errorf("executing command %q: %w", command, execErr)
		}
		return string(output), nil
	}
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
