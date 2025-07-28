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

	// Configure NTP server before WiFi setup
	if err := configureNTPServer(); err != nil {
		return fmt.Errorf("failed to configure NTP server: %v", err)
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

// ResetWiFiConfig removes all WiFi configurations
func (s *SystemWiFiService) ResetWiFiConfig() error {
	// Remove all karaoke-wifi-*.config files
	configDir := "/var/lib/connman"
	files, err := os.ReadDir(configDir)
	if err != nil {
		return fmt.Errorf("failed to read connman directory: %v", err)
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "karaoke-wifi-") && strings.HasSuffix(file.Name(), ".config") {
			configPath := fmt.Sprintf("%s/%s", configDir, file.Name())
			if err := os.Remove(configPath); err != nil {
				return fmt.Errorf("failed to remove config file %s: %v", file.Name(), err)
			}
		}
	}

	// Disconnect from WiFi and restart connman
	exec.Command("connmanctl", "disable", "wifi").Run()
	exec.Command("connmanctl", "enable", "wifi").Run()

	return nil
}

// configureNTPServer adds NTP server to ConnMan settings using regex
func configureNTPServer() error {
	settingsPath := "/var/lib/connman/settings"

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create entire settings file with NTP configuration
			defaultContent := `[global]
OfflineMode=false
Timeservers = ntp.nict.jp

[WiFi]
Enable=true
Tethering=false
Tethering.Freq=2412

`
			return os.WriteFile(settingsPath, []byte(defaultContent), 0644)
		}
		return fmt.Errorf("failed to read settings: %v", err)
	}

	text := string(content)

	// Check if NTP is already configured
	if strings.Contains(text, "Timeservers = ntp.nict.jp") {
		return nil
	}

	// If [global] section exists, add NTP server after OfflineMode
	re := regexp.MustCompile(`(\[global\][^[]*)OfflineMode=false`)
	if re.MatchString(text) {
		newText := re.ReplaceAllString(text, `$1OfflineMode=false
Timeservers = ntp.nict.jp`)
		return os.WriteFile(settingsPath, []byte(newText), 0644)
	}

	// If [global] section doesn't exist, add it at the beginning
	globalSection := `[global]
OfflineMode=false
Timeservers = ntp.nict.jp

`
	newText := globalSection + text
	return os.WriteFile(settingsPath, []byte(newText), 0644)
}
