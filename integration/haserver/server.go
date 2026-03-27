package haserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"

	ha "codeberg.org/jhot/go-ha-sdk"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// MockHAServer is a fake Home Assistant WebSocket server for testing.
// Create with New(); inject events via InjectXxx methods; observe with ConnectionCount/PingsReceived.
type MockHAServer struct {
	Server     *httptest.Server
	Token      string
	RejectAuth bool // send auth_invalid instead of auth_ok
	DropOnPing bool // close connection when first ping is received

	mu            sync.Mutex
	allConns      []*websocket.Conn
	pingsReceived int
	eventQueue    chan []byte
}

// New creates and starts a MockHAServer backed by httptest.Server. Call Close() when done.
func New(token string) *MockHAServer {
	s := newBare(token)
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	s.Server = httptest.NewServer(mux)
	return s
}

// NewBare creates a MockHAServer without starting any HTTP server.
// Call RegisterRoutes to attach the WebSocket handler to an existing mux,
// then start your own http.Server. Addr() returns "" and Close() is a no-op.
func NewBare(token string) *MockHAServer {
	return newBare(token)
}

func newBare(token string) *MockHAServer {
	return &MockHAServer{
		Token:      token,
		eventQueue: make(chan []byte, 64),
	}
}

// RegisterRoutes registers the /api/websocket handler on mux.
func (s *MockHAServer) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/websocket", s.handle)
}

// Addr returns "127.0.0.1:PORT" (no scheme) for use in HatsConfig.HomeAssistantHost/Port.
// Returns "" for bare servers created with NewBare.
func (s *MockHAServer) Addr() string {
	if s.Server == nil {
		return ""
	}
	return s.Server.Listener.Addr().String()
}

// Close shuts down the underlying httptest.Server. No-op for bare servers.
func (s *MockHAServer) Close() {
	if s.Server != nil {
		s.Server.Close()
	}
}

// DropConnection forcibly closes the active WebSocket connection.
// Note: calling Close() concurrently with an in-progress WriteJSON in the handler goroutine
// is a known limitation; safe in practice for test use.
func (s *MockHAServer) DropConnection() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.allConns) > 0 {
		s.allConns[len(s.allConns)-1].Close()
	}
}

// ConnectionCount returns total number of WebSocket connections ever established.
func (s *MockHAServer) ConnectionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.allConns)
}

// PingsReceived returns total number of ping messages received across all connections.
func (s *MockHAServer) PingsReceived() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pingsReceived
}

// InjectStateChange queues a state_changed event for delivery to the subscriber.
func (s *MockHAServer) InjectStateChange(entityId, oldState, newState string) {
	s.inject(ha.HassMessage{
		Type: ha.MsgEvent,
		ID:   1001,
		Event: ha.Event{
			EventType: ha.EventStateChanged,
			Data: ha.EventData{
				EntityID: entityId,
				NewState: ha.StateData{State: newState},
				OldState: ha.StateData{State: oldState},
			},
		},
	})
}

// InjectZHAEvent queues a zha_event for delivery to the subscriber.
func (s *MockHAServer) InjectZHAEvent(deviceIeee, command string) {
	s.inject(ha.HassMessage{
		Type: ha.MsgEvent,
		ID:   1002,
		Event: ha.Event{
			EventType: ha.EventZHA,
			Data:      ha.EventData{DeviceIEEE: deviceIeee, Command: command},
		},
	})
}

// InjectNFCEvent queues a tag_scanned event for delivery to the subscriber.
func (s *MockHAServer) InjectNFCEvent(tagId string) {
	s.inject(ha.HassMessage{
		Type: ha.MsgEvent,
		ID:   1003,
		Event: ha.Event{
			EventType: ha.EventTagScanned,
			Data:      ha.EventData{TagID: tagId},
		},
	})
}

// InjectTimerFinished queues a timer.finished event for delivery to the subscriber.
func (s *MockHAServer) InjectTimerFinished(entityId string) {
	s.inject(ha.HassMessage{
		Type: ha.MsgEvent,
		ID:   1005,
		Event: ha.Event{
			EventType: ha.EventTimerFinished,
			Data:      ha.EventData{EntityID: entityId},
		},
	})
}

// InjectZWaveEvent queues a zwave_js_value_notification event for delivery to the subscriber.
func (s *MockHAServer) InjectZWaveEvent(deviceId string) {
	s.inject(ha.HassMessage{
		Type: ha.MsgEvent,
		ID:   1004,
		Event: ha.Event{
			EventType: ha.EventZWave,
			Data:      ha.EventData{DeviceID: deviceId},
		},
	})
}

func (s *MockHAServer) inject(msg ha.HassMessage) {
	data, _ := json.Marshal(msg)
	select {
	case s.eventQueue <- data:
	default:
		slog.Warn("haserver: event queue full, dropping event")
	}
}

func (s *MockHAServer) handle(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("haserver: upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	s.mu.Lock()
	s.allConns = append(s.allConns, conn)
	s.mu.Unlock()

	// Step 1: send auth_required
	if err := conn.WriteJSON(ha.HassMessage{Type: ha.MsgAuthRequired, Version: "2024.1.0"}); err != nil {
		return
	}

	// Step 2: read auth
	_, data, err := conn.ReadMessage()
	if err != nil {
		return
	}
	var authMsg ha.AuthMessage
	if err := json.Unmarshal(data, &authMsg); err != nil {
		slog.Error("haserver: failed to unmarshal auth message", "error", err)
		return
	}

	if s.RejectAuth || authMsg.AccessToken != s.Token {
		conn.WriteJSON(ha.HassMessage{Type: ha.MsgAuthInvalid})
		return
	}
	if err := conn.WriteJSON(ha.HassMessage{Type: ha.MsgAuthOk}); err != nil {
		return
	}

	// Step 3+: concurrent read loop + event injection
	type readResult struct {
		data []byte
		err  error
	}
	msgCh := make(chan readResult, 8)
	go func() {
		for {
			_, d, err := conn.ReadMessage()
			msgCh <- readResult{d, err}
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case result := <-msgCh:
			if result.err != nil {
				return
			}
			var msg ha.HassMessage
			if err := json.Unmarshal(result.data, &msg); err != nil {
				slog.Warn("haserver: failed to unmarshal message", "error", err)
			} else {
				switch msg.Type {
				case ha.MsgSubscribeEvents:
					if err := conn.WriteJSON(ha.HassMessage{Type: ha.MsgResult, ID: msg.ID, Success: true}); err != nil {
						return
					}
				case ha.MsgPing:
					s.mu.Lock()
					s.pingsReceived++
					dropIt := s.DropOnPing
					s.mu.Unlock()
					if dropIt {
						conn.Close()
						return
					}
					if err := conn.WriteJSON(ha.HassMessage{Type: ha.MsgPong, ID: msg.ID}); err != nil {
						return
					}
				}
			}
		case eventData := <-s.eventQueue:
			if err := conn.WriteMessage(websocket.TextMessage, eventData); err != nil {
				return
			}
		}
	}
}
