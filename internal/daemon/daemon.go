package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Config struct {
	Active  string       `json:"active"`
	Configs []APIConfig  `json:"configs"`
}

type APIConfig struct {
	Alias     string `json:"alias"`
	APIKey    string `json:"api_key"`
	AuthToken string `json:"auth_token"`
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
}

type Daemon struct {
	configPath    string
	socketPath    string
	pidPath       string
	config        *Config
	configVersion int64
	mu            sync.RWMutex
	watcher       *fsnotify.Watcher
	listener      net.Listener
	ctx           context.Context
	cancel        context.CancelFunc
	debouncer     *time.Timer
	debounceMu    sync.Mutex
}

func New(configDir, runtimeDir string) (*Daemon, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Daemon{
		configPath:    filepath.Join(configDir, "config.json"),
		socketPath:    filepath.Join(runtimeDir, "apimgr.sock"),
		pidPath:       filepath.Join(runtimeDir, "daemon.pid"),
		configVersion: time.Now().Unix(),
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

func (d *Daemon) Start() error {
	// Clean up old socket if exists
	if err := d.cleanupSocket(); err != nil {
		return fmt.Errorf("failed to cleanup socket: %w", err)
	}
	
	// Write PID file
	if err := d.writePIDFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}
	
	// Load initial config
	if err := d.loadConfig(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	// Start config watcher
	if err := d.startWatcher(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}
	
	// Start Unix socket server
	if err := d.startSocketServer(); err != nil {
		return fmt.Errorf("failed to start socket server: %w", err)
	}
	
	// Handle signals
	d.handleSignals()
	
	return nil
}

func (d *Daemon) Stop() {
	log.Println("Stopping daemon...")
	d.cancel()
	
	if d.listener != nil {
		d.listener.Close()
	}
	
	if d.watcher != nil {
		d.watcher.Close()
	}
	
	d.cleanup()
	log.Println("Daemon stopped")
}

func (d *Daemon) cleanupSocket() error {
	if _, err := os.Stat(d.socketPath); err == nil {
		if err := os.Remove(d.socketPath); err != nil {
			return err
		}
	}
	
	// Ensure socket directory exists
	socketDir := filepath.Dir(d.socketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return err
	}
	
	return nil
}

func (d *Daemon) writePIDFile() error {
	pidDir := filepath.Dir(d.pidPath)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return err
	}
	
	pid := os.Getpid()
	return os.WriteFile(d.pidPath, []byte(fmt.Sprintf("%d", pid)), 0644)
}

func (d *Daemon) loadConfig() error {
	data, err := os.ReadFile(d.configPath)
	if err != nil {
		return err
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	
	d.mu.Lock()
	d.config = &config
	d.configVersion = time.Now().Unix()
	d.mu.Unlock()
	
	log.Printf("Config loaded: active=%s, version=%d", config.Active, d.configVersion)
	return nil
}

func (d *Daemon) startWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	
	d.watcher = watcher
	
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					d.debouncedReload()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Watcher error:", err)
			case <-d.ctx.Done():
				return
			}
		}
	}()
	
	// Watch config file
	configDir := filepath.Dir(d.configPath)
	if err := watcher.Add(configDir); err != nil {
		return err
	}
	
	log.Printf("Watching config directory: %s", configDir)
	return nil
}

func (d *Daemon) debouncedReload() {
	d.debounceMu.Lock()
	defer d.debounceMu.Unlock()
	
	if d.debouncer != nil {
		d.debouncer.Stop()
	}
	
	d.debouncer = time.AfterFunc(100*time.Millisecond, func() {
		if err := d.loadConfig(); err != nil {
			log.Printf("Failed to reload config: %v", err)
		}
	})
}

func (d *Daemon) startSocketServer() error {
	listener, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return err
	}
	
	d.listener = listener
	
	// Set socket permissions
	if err := os.Chmod(d.socketPath, 0600); err != nil {
		return err
	}
	
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-d.ctx.Done():
					return
				default:
					log.Printf("Accept error: %v", err)
					continue
				}
			}
			
			go d.handleConnection(conn)
		}
	}()
	
	log.Printf("Socket server listening on: %s", d.socketPath)
	return nil
}

func (d *Daemon) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	// Set timeout
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Printf("Read error: %v", err)
		return
	}
	
	command := strings.TrimSpace(string(buf[:n]))
	response := d.processCommand(command)
	
	if _, err := conn.Write([]byte(response)); err != nil {
		log.Printf("Write error: %v", err)
	}
}

func (d *Daemon) processCommand(command string) string {
	parts := strings.Split(command, " ")
	if len(parts) == 0 {
		return "ERROR: empty command"
	}
	
	switch parts[0] {
	case "GET":
		return d.handleGet()
	case "VERSION":
		return d.handleVersion()
	case "PING":
		return "PONG"
	case "RELOAD":
		if err := d.loadConfig(); err != nil {
			return fmt.Sprintf("ERROR: %v", err)
		}
		return "OK"
	default:
		return fmt.Sprintf("ERROR: unknown command: %s", parts[0])
	}
}

func (d *Daemon) handleGet() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	if d.config == nil {
		return "ERROR: config not loaded"
	}
	
	// Find active config
	for _, cfg := range d.config.Configs {
		if cfg.Alias == d.config.Active {
			result := make(map[string]string)
			
			// Use API key or auth token
			if cfg.APIKey != "" {
				result["ANTHROPIC_API_KEY"] = cfg.APIKey
			} else if cfg.AuthToken != "" {
				result["ANTHROPIC_API_KEY"] = cfg.AuthToken
			}
			
			if cfg.BaseURL != "" {
				result["ANTHROPIC_BASE_URL"] = cfg.BaseURL
			}
			
			data, _ := json.Marshal(result)
			return string(data)
		}
	}
	
	return "{}"
}

func (d *Daemon) handleVersion() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	return fmt.Sprintf("%d", d.configVersion)
}

func (d *Daemon) handleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	
	go func() {
		for {
			select {
			case sig := <-sigChan:
				switch sig {
				case syscall.SIGHUP:
					log.Println("Received SIGHUP, reloading config...")
					if err := d.loadConfig(); err != nil {
						log.Printf("Failed to reload config: %v", err)
					}
				case syscall.SIGINT, syscall.SIGTERM:
					log.Printf("Received %v, shutting down...", sig)
					d.Stop()
					return
				}
			case <-d.ctx.Done():
				return
			}
		}
	}()
}

func (d *Daemon) cleanup() {
	// Remove socket file
	if _, err := os.Stat(d.socketPath); err == nil {
		os.Remove(d.socketPath)
	}
	
	// Remove PID file
	if _, err := os.Stat(d.pidPath); err == nil {
		os.Remove(d.pidPath)
	}
}

func (d *Daemon) IsRunning() bool {
	// Check if PID file exists
	data, err := os.ReadFile(d.pidPath)
	if err != nil {
		return false
	}
	
	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return false
	}
	
	// Check if process is running
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	
	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
