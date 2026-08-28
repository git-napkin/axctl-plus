package server

import (
	"fmt"
	"os/exec"
	"strings"
)

func gsettingsBin() (string, error) {
	bin, err := exec.LookPath("gsettings")
	if err != nil {
		return "", fmt.Errorf("gsettings not found")
	}
	return bin, nil
}

func preferredColorScheme() (string, error) {
	bin, err := gsettingsBin()
	if err != nil {
		return "", err
	}
	out, err := exec.Command(bin, "get", "org.gnome.desktop.interface", "color-scheme").Output()
	if err != nil {
		return "", fmt.Errorf("gsettings get failed: %v", err)
	}
	return strings.Trim(strings.TrimSpace(string(out)), "'"), nil
}

func IsDarkMode() (bool, error) {
	scheme, err := preferredColorScheme()
	if err != nil {
		return false, err
	}
	return scheme == "prefer-dark", nil
}

func SetDarkMode(on bool) error {
	bin, err := gsettingsBin()
	if err != nil {
		return err
	}
	scheme := "prefer-light"
	if on {
		scheme = "prefer-dark"
	}
	if err := exec.Command(bin, "set", "org.gnome.desktop.interface", "color-scheme", scheme).Run(); err != nil {
		return fmt.Errorf("gsettings set failed: %v", err)
	}
	return nil
}
