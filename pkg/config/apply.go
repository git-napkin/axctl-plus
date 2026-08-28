package config

import (
	"fmt"

	"axctl/pkg/ipc"
	"axctl/pkg/server"
)

func ApplyConfig(cfg *TOMLConfig, compositor ipc.Compositor) error {
	ipcCfg := cfg.ToIPCConfig()
	handler := server.NewConfigHandler(compositor)
	if err := handler.ApplyConfig(ipcCfg); err != nil {
		return fmt.Errorf("failed to apply config: %w", err)
	}
	return nil
}
