package daemon

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func RunLauncher(sentinelDir string) error {
	daemonsDir := filepath.Join(sentinelDir, "daemons")
	registry, err := LoadRegistry(daemonsDir)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	manager, err := NewManager(registry)
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	server, err := NewServer(sentinelDir, registry, manager)
	if err != nil {
		return fmt.Errorf("create ipc server: %w", err)
	}

	pidPath := filepath.Join(sentinelDir, "daemon.pid")
	if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		return fmt.Errorf("write pid file: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	fmt.Printf("gdc-sentinel daemon started (pid %d)\n", os.Getpid())

	if err := server.Start(); err != nil {
		return fmt.Errorf("start ipc server: %w", err)
	}

	<-sigCh
	fmt.Println("shutting down daemon...")

	manager.StopAll()
	server.Stop()
	os.Remove(pidPath)

	return nil
}
