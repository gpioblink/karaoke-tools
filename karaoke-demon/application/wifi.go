package application

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode/utf8"
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

// validateWiFiConfig validates SSID and password inputs
func validateWiFiConfig(config WiFiConfig) error {
	// SSID validation
	if config.SSID == "" {
		return fmt.Errorf("SSID cannot be empty")
	}
	if len(config.SSID) > 32 {
		return fmt.Errorf("SSID length cannot exceed 32 characters")
	}
	if !utf8.ValidString(config.SSID) {
		return fmt.Errorf("SSID contains invalid UTF-8 characters")
	}
	// Check for control characters and newlines that could break config format
	if regexp.MustCompile(`[\x00-\x1f\x7f]`).MatchString(config.SSID) {
		return fmt.Errorf("SSID contains invalid control characters")
	}

	// Password validation
	if len(config.Password) < 8 || len(config.Password) > 63 {
		return fmt.Errorf("password length must be between 8 and 63 characters")
	}
	if !utf8.ValidString(config.Password) {
		return fmt.Errorf("password contains invalid UTF-8 characters")
	}
	// Check for control characters that could break config format
	if regexp.MustCompile(`[\x00-\x1f\x7f]`).MatchString(config.Password) {
		return fmt.Errorf("password contains invalid control characters")
	}

	return nil
}

// ConfigureWiFi sets up WiFi connection using ConnMan (connmanctl)
func (s *SystemWiFiService) ConfigureWiFi(config WiFiConfig) error {
	// Validate input
	if err := validateWiFiConfig(config); err != nil {
		return fmt.Errorf("invalid WiFi configuration: %v", err)
	}

	// Create SHA256 hash of SSID to prevent path traversal
	hash := sha256.Sum256([]byte(config.SSID))
	hashStr := hex.EncodeToString(hash[:])
	configPath := fmt.Sprintf("/var/lib/connman/karaoke-wifi-%s.config", hashStr)

	// Create ConnMan service configuration file
	configContent := fmt.Sprintf(`[service_wifi]
Type = wifi
Name = %s
Passphrase = %s
AutoConnect = true
`, config.SSID, config.Password)

	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		return fmt.Errorf("failed to create WiFi config: %v", err)
	}

	// Enable WiFi and scan
	exec.Command("connmanctl", "enable", "wifi").Run()
	exec.Command("connmanctl", "scan", "wifi").Run()

	return nil
}

// GetCurrentConnection returns the currently connected WiFi SSID
func (s *SystemWiFiService) GetCurrentConnection() (string, error) {
	cmd := exec.Command("connmanctl", "state")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get connection state: %v", err)
	}

	// Check if we're connected to WiFi
	if !strings.Contains(string(output), "State = online") {
		return "", fmt.Errorf("no active connection found")
	}

	// Get services to find the connected WiFi
	cmd = exec.Command("connmanctl", "services")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get services: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "*AO") { // Connected service marker
			parts := strings.Fields(line)
			if len(parts) >= 3 && strings.HasPrefix(parts[2], "wifi_") {
				// Extract SSID from service name
				serviceParts := strings.Split(parts[1], " ")
				if len(serviceParts) > 0 {
					return serviceParts[0], nil
				}
			}
		}
	}

	return "", fmt.Errorf("no active WiFi connection found")
}
