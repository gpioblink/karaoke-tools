package application

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type NgrokService struct {
	cmd       *exec.Cmd
	localPort int
}

type NgrokAPI struct {
	Tunnels []struct {
		PublicURL string `json:"public_url"`
		Config    struct {
			Addr string `json:"addr"`
		} `json:"config"`
	} `json:"tunnels"`
}

func NewNgrokService(localPort int) *NgrokService {
	return &NgrokService{
		localPort: localPort,
	}
}

func (n *NgrokService) Start() error {
	// Start ngrok tunnel
	n.cmd = exec.Command("ngrok", "http", fmt.Sprintf("%d", n.localPort))

	err := n.cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start ngrok: %v", err)
	}

	// Wait for ngrok to initialize
	time.Sleep(3 * time.Second)

	return nil
}

func (n *NgrokService) Stop() error {
	if n.cmd != nil && n.cmd.Process != nil {
		err := n.cmd.Process.Kill()
		if err != nil {
			return fmt.Errorf("failed to stop ngrok: %v", err)
		}
		n.cmd.Wait()
	}
	return nil
}

func (n *NgrokService) GetPublicURL() (string, error) {
	// Query ngrok local API
	resp, err := http.Get("http://localhost:4040/api/tunnels")
	if err != nil {
		return "", fmt.Errorf("failed to query ngrok API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read ngrok API response: %v", err)
	}

	var ngrokAPI NgrokAPI
	err = json.Unmarshal(body, &ngrokAPI)
	if err != nil {
		return "", fmt.Errorf("failed to parse ngrok API response: %v", err)
	}

	// Find HTTP tunnel for our port
	localAddr := fmt.Sprintf("http://localhost:%d", n.localPort)
	for _, tunnel := range ngrokAPI.Tunnels {
		if tunnel.Config.Addr == localAddr && strings.HasPrefix(tunnel.PublicURL, "https://") {
			return tunnel.PublicURL, nil
		}
	}

	return "", fmt.Errorf("no HTTPS tunnel found for port %d", n.localPort)
}

func (n *NgrokService) ConfigureAuthToken(token string) error {
	if !isValidNgrokToken(token) {
		return fmt.Errorf("invalid ngrok token format")
	}

	cmd := exec.Command("ngrok", "config", "add-authtoken", token)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to configure ngrok auth token: %v", err)
	}

	return nil
}

func isValidNgrokToken(token string) bool {
	if len(token) == 0 {
		return false
	}

	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, token)
	return matched && len(token) >= 10 && len(token) <= 200
}
