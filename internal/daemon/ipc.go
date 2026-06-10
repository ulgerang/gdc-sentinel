package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type MessageType string

const (
	MsgStart   MessageType = "start"
	MsgStop    MessageType = "stop"
	MsgRestart MessageType = "restart"
	MsgList    MessageType = "list"
	MsgStatus  MessageType = "status"
	MsgPing    MessageType = "ping"
	MsgResult  MessageType = "result"
)

type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type StartPayload struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type StopPayload struct {
	Name string `json:"name"`
}

type StatusPayload struct {
	Name string `json:"name"`
}

type ResultPayload struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Server struct {
	socketPath string
	listener   net.Listener
	registry   *Registry
	manager    *Manager
	mu         sync.Mutex
	stopCh     chan struct{}
}

func NewServer(sentinelDir string, registry *Registry, manager *Manager) (*Server, error) {
	socketPath := filepath.Join(sentinelDir, "daemon.sock")
	return &Server{
		socketPath: socketPath,
		registry:   registry,
		manager:    manager,
		stopCh:     make(chan struct{}),
	}, nil
}

func (s *Server) Start() error {
	os.Remove(s.socketPath)

	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.socketPath, err)
	}
	s.listener = l

	os.Chmod(s.socketPath, 0600)

	go s.acceptLoop()
	return nil
}

func (s *Server) Stop() {
	close(s.stopCh)
	if s.listener != nil {
		s.listener.Close()
	}
	os.Remove(s.socketPath)
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
				continue
			}
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	var msg Message
	if err := dec.Decode(&msg); err != nil {
		enc.Encode(resultMsg(false, nil, "invalid message"))
		return
	}

	resp := s.handleMessage(msg)
	enc.Encode(resp)
}

func (s *Server) handleMessage(msg Message) Message {
	switch msg.Type {
	case MsgPing:
		return resultMsg(true, "pong", "")

	case MsgStart:
		var p StartPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return resultMsg(false, nil, "invalid payload")
		}
		return s.handleStart(p)

	case MsgStop:
		var p StopPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return resultMsg(false, nil, "invalid payload")
		}
		return s.handleStop(p)

	case MsgRestart:
		var p StopPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return resultMsg(false, nil, "invalid payload")
		}
		return s.handleRestart(p)

	case MsgList:
		return s.handleList()

	case MsgStatus:
		var p StatusPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return resultMsg(false, nil, "invalid payload")
		}
		return s.handleStatus(p)

	default:
		return resultMsg(false, nil, fmt.Sprintf("unknown message type: %s", msg.Type))
	}
}

func (s *Server) handleStart(p StartPayload) Message {
	w, err := s.manager.StartWorkspace(p.Name, p.Path)
	if err != nil {
		return resultMsg(false, nil, err.Error())
	}
	return resultMsg(true, w, "")
}

func (s *Server) handleStop(p StopPayload) Message {
	if err := s.manager.StopWorkspace(p.Name); err != nil {
		return resultMsg(false, nil, err.Error())
	}
	return resultMsg(true, nil, "")
}

func (s *Server) handleRestart(p StopPayload) Message {
	if err := s.manager.RestartWorkspace(p.Name); err != nil {
		return resultMsg(false, nil, err.Error())
	}
	return resultMsg(true, nil, "")
}

func (s *Server) handleList() Message {
	workspaces := s.manager.ListWithStatus()
	return resultMsg(true, workspaces, "")
}

func (s *Server) handleStatus(p StatusPayload) Message {
	w, ok := s.manager.GetWithStatus(p.Name)
	if !ok {
		return resultMsg(false, nil, fmt.Sprintf("workspace %q not found", p.Name))
	}
	return resultMsg(true, w, "")
}

func resultMsg(success bool, data interface{}, errMsg string) Message {
	r := ResultPayload{
		Success: success,
		Data:    data,
		Error:   errMsg,
	}
	payload, _ := json.Marshal(r)
	return Message{Type: MsgResult, Payload: payload}
}

func SendIPC(socketPath string, msg Message) (Message, error) {
	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		return Message{}, fmt.Errorf("connect to daemon: %w (is the daemon running?)", err)
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	if err := enc.Encode(&msg); err != nil {
		return Message{}, fmt.Errorf("send message: %w", err)
	}

	var resp Message
	if err := dec.Decode(&resp); err != nil {
		return Message{}, fmt.Errorf("read response: %w", err)
	}
	return resp, nil
}

func ReadSocketPath(sentinelDir string) string {
	return filepath.Join(sentinelDir, "daemon.sock")
}

func IsDaemonRunning(socketPath string) bool {
	conn, err := net.DialTimeout("unix", socketPath, time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
