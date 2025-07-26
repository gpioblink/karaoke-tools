package ngrok

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type NgrokService struct {
	cmd        *exec.Cmd
	publicURL  string
	localPort  int
	isRunning  bool
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
		isRunning: false,
	}
}

func (n *NgrokService) Start() error {
	if n.isRunning {
		return fmt.Errorf("ngrok is already running")
	}

	// Start ngrok tunnel
	n.cmd = exec.Command("ngrok", "http", fmt.Sprintf("%d", n.localPort))
	
	err := n.cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start ngrok: %v", err)
	}

	// Wait for ngrok to initialize
	time.Sleep(3 * time.Second)

	// Get public URL from ngrok API
	publicURL, err := n.getPublicURL()
	if err != nil {
		n.Stop()
		return fmt.Errorf("failed to get ngrok public URL: %v", err)
	}

	n.publicURL = publicURL
	n.isRunning = true

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
	n.isRunning = false
	n.publicURL = ""
	return nil
}

func (n *NgrokService) GetPublicURL() string {
	return n.publicURL
}

func (n *NgrokService) IsRunning() bool {
	return n.isRunning
}

func (n *NgrokService) getPublicURL() (string, error) {
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
	localAddr := fmt.Sprintf("localhost:%d", n.localPort)
	for _, tunnel := range ngrokAPI.Tunnels {
		if tunnel.Config.Addr == localAddr && strings.HasPrefix(tunnel.PublicURL, "https://") {
			return tunnel.PublicURL, nil
		}
	}

	return "", fmt.Errorf("no HTTPS tunnel found for port %d", n.localPort)
}