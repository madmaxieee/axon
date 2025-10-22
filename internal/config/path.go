package config

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

func GetConfigHome() string {
	return filepath.Join(xdg.ConfigHome, "axon")
}
