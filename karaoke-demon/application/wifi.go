package application

import (
	"fmt"
	"os/exec"
	"strings"
)

type WiFiConfig struct {
	SSID     string
	Password string
}

type WiFiService interface {
	ConfigureWiFi(config WiFiConfig) error
	GetCurrentConnection() (string, error)
}

type SystemWiFiService struct{}

func NewSystemWiFiService() *SystemWiFiService {
	return &SystemWiFiService{}
}

// ConfigureWiFi sets up WiFi connection using NetworkManager (nmcli)
func (s *SystemWiFiService) ConfigureWiFi(config WiFiConfig) error {
	// First, check if connection already exists and delete it
	connectionName := fmt.Sprintf("karaoke-wifi-%s", config.SSID)

	// Delete existing connection if it exists (ignore errors)
	exec.Command("nmcli", "connection", "delete", connectionName).Run()

	// Create new WiFi connection
	cmd := exec.Command("nmcli", "device", "wifi", "connect", config.SSID,
		"password", config.Password, "name", connectionName)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to configure WiFi: %v, output: %s", err, string(output))
	}

	return nil
}

// GetCurrentConnection returns the currently connected WiFi SSID
func (s *SystemWiFiService) GetCurrentConnection() (string, error) {
	cmd := exec.Command("nmcli", "-t", "-f", "ACTIVE,SSID", "dev", "wifi")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current WiFi connection: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 && parts[0] == "yes" {
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("no active WiFi connection found")
}
