package realtime

import "sync"

type Hub struct {
	mu          sync.Mutex
	nextID      uint64
	subscribers map[string]map[uint64]chan []byte
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[string]map[uint64]chan []byte)}
}

func (h *Hub) Subscribe(matchID string) (<-chan []byte, func()) {
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	channel := make(chan []byte, 64)
	if h.subscribers[matchID] == nil {
		h.subscribers[matchID] = make(map[uint64]chan []byte)
	}
	h.subscribers[matchID][id] = channel
	h.mu.Unlock()

	var once sync.Once
	return channel, func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			if subscribers := h.subscribers[matchID]; subscribers != nil {
				if current, ok := subscribers[id]; ok {
					delete(subscribers, id)
					close(current)
				}
				if len(subscribers) == 0 {
					delete(h.subscribers, matchID)
				}
			}
		})
	}
}

func (h *Hub) Publish(matchID string, message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, subscriber := range h.subscribers[matchID] {
		copyOfMessage := append([]byte(nil), message...)
		select {
		case subscriber <- copyOfMessage:
		default:
			delete(h.subscribers[matchID], id)
			close(subscriber)
		}
	}
	if len(h.subscribers[matchID]) == 0 {
		delete(h.subscribers, matchID)
	}
}
