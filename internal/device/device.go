package device

import (
	"os"
	"runtime"
	"time"

	"github.com/JPlanken/metarepo-cli/internal/config"
)

// Info holds information about the current device
type Info struct {
	Serial   string
	Name     string
	Platform string
	Hostname string
	Username string
	Arch     string
}

// GetCurrentDevice returns information about the current device
func GetCurrentDevice() (*Info, error) {
	serial, err := GetSerialNumber()
	if err != nil {
		return nil, err
	}

	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}

	return &Info{
		Serial:   serial,
		Platform: runtime.GOOS,
		Hostname: hostname,
		Username: username,
		Arch:     runtime.GOARCH,
	}, nil
}

// ToConfigDevice converts device Info to a config.Device
func (i *Info) ToConfigDevice(name string) config.Device {
	return config.Device{
		Serial:     i.Serial,
		Name:       name,
		Platform:   i.Platform,
		Hostname:   i.Hostname,
		Registered: time.Now(),
	}
}
