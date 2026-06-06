package debugcapture

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Capture struct {
	basePath string
}

func New(basePath string) (*Capture, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("creating capture directory: %w", err)
	}
	return &Capture{basePath: basePath}, nil
}

func (c *Capture) Save(device, command, output string) error {
	dir := filepath.Join(c.basePath, device)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating device capture dir: %w", err)
	}

	ts := time.Now().Format("20060102-150405")
	filename := filepath.Join(dir, fmt.Sprintf("%s-%s.txt", ts, sanitize(command)))
	return os.WriteFile(filename, []byte(output), 0644)
}

func sanitize(s string) string {
	result := make([]byte, 0, len(s))
	for _, b := range []byte(s) {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '-' || b == '_' {
			result = append(result, b)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}
