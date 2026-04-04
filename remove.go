package main

import (
	"fmt"
	"os"
	"os/exec"
)

const unitFile = "/etc/systemd/system/sas_exporter.service"

func selfRemove() error {
	fmt.Println("Stopping and disabling sas_exporter service...")
	_ = exec.Command("systemctl", "disable", "--now", "sas_exporter").Run()

	if err := os.Remove(unitFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing unit file: %w", err)
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	fmt.Println("Removing binary...")
	if err := os.Remove(exePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing binary: %w", err)
	}

	fmt.Println("sas_exporter removed.")
	return nil
}

func restartService() {
	_ = exec.Command("systemctl", "restart", "sas_exporter").Run()
}
