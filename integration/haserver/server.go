package haserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"

	ha "codeberg.org/jhot/hats/pkg/homeassistant"
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

// New creates and starts a MockHAServer. Call Close() when done.
func New(token string) *MockHAServer {
	s := &MockHAServer{
		Token:      token,
		eventQueue: make(chan []byte, 64),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/websocket", s.handle)
	s.Server = httptest.NewServer(mux)
	return s
}

// Addr returns "127.0.0.1:PORT" (no scheme) for use in HatsConfig.HomeAssistantHost/Port.
func (s *MockHAServer) Addr() string {
	return s.Server.Listener.Addr().String()
}

// Close shuts down the underlying httptest.Server.
func (s *MockHAServer) Close() {
	s.Server.Close()
}

// DropConnection forcibly closes the active WebSocket connection.
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
		Type: ha.MessageType.Event,
		Id:   1001,
		Event: ha.Event{
			EventType: ha.MessageType.StateChanged,
			Data: ha.EventData{
				EntityId: entityId,
				NewState: ha.StateData{State: newState},
				OldState: ha.StateData{State: oldState},
			},
		},
	})
}

// InjectZHAEvent queues a zha_event for delivery to the subscriber.
func (s *MockHAServer) InjectZHAEvent(deviceIeee, command string) {
	s.inject(ha.HassMessage{
		Type: ha.MessageType.Event,
		Id:   1002,
		Event: ha.Event{
			EventType: ha.MessageType.ZhaEvent,
			Data:      ha.EventData{DeviceIeee: deviceIeee, Command: command},
		},
	})
}

// InjectNFCEvent queues a tag_scanned event for delivery to the subscriber.
func (s *MockHAServer) InjectNFCEvent(tagId string) {
	s.inject(ha.HassMessage{
		Type: ha.MessageType.Event,
		Id:   1003,
		Event: ha.Event{
			EventType: ha.MessageType.TagScanned,
			Data:      ha.EventData{TagId: tagId},
		},
	})
}

// InjectTimerFinished queues a timer.finished event for delivery to the subscriber.
func (s *MockHAServer) InjectTimerFinished(entityId string) {
	s.inject(ha.HassMessage{
		Type: ha.MessageType.Event,
		Id:   1005,
		Event: ha.Event{
			EventType: ha.MessageType.TimerFinished,
			Data:      ha.EventData{EntityId: entityId},
		},
	})
}

// InjectZWaveEvent queues a zwave_js_value_notification event for delivery to the subscriber.
func (s *MockHAServer) InjectZWaveEvent(deviceId string) {
	s.inject(ha.HassMessage{
		Type: ha.MessageType.Event,
		Id:   1004,
		Event: ha.Event{
			EventType: ha.MessageType.ZwaveEvent,
			Data:      ha.EventData{DeviceId: deviceId},
		},
	})
}

func (s *MockHAServer) inject(msg ha.HassMessage) {
	data, _ := json.Marshal(msg)
	s.eventQueue <- data
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
	if err := conn.WriteJSON(ha.HassMessage{Type: ha.MessageType.AuthRequired, Version: "2024.1.0"}); err != nil {
		return
	}

	// Step 2: read auth
	_, data, err := conn.ReadMessage()
	if err != nil {
		return
	}
	var authMsg ha.HassMessage
	json.Unmarshal(data, &authMsg)

	if s.RejectAuth || authMsg.AccessToken != s.Token {
		conn.WriteJSON(ha.HassMessage{Type: ha.MessageType.AuthInvalid})
		return
	}
	if err := conn.WriteJSON(ha.HassMessage{Type: ha.MessageType.AuthOk}); err != nil {
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
			json.Unmarshal(result.data, &msg)
			switch msg.Type {
			case ha.MessageType.SubscribeEvents:
				conn.WriteJSON(ha.HassMessage{Type: ha.MessageType.Result, Id: msg.Id, Success: true})
			case ha.MessageType.Ping:
				s.mu.Lock()
				s.pingsReceived++
				dropIt := s.DropOnPing
				s.mu.Unlock()
				if dropIt {
					conn.Close()
					return
				}
				conn.WriteJSON(ha.HassMessage{Type: ha.MessageType.Pong, Id: msg.Id})
			}
		case eventData := <-s.eventQueue:
			if err := conn.WriteMessage(websocket.TextMessage, eventData); err != nil {
				return
			}
		}
	}
}
