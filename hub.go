package pgbroadcaster

import "fmt"

// The hub maintains the set of active connections and broadcasts messages to
// the connections.
type hub struct {
	// Registered connections.
	connections map[*connection]bool
	// Inbound notifications from PostgreSQL.
	broadcast chan pgnotification
	// Register requests from the connections.
	register chan *connection
	// Unregister requests from connections.
	unregister chan *connection
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			// Register new client in hub
			h.connections[c] = true
		case c := <-h.unregister:
			// Unregister client from hub
			if _, ok := h.connections[c]; ok {
				delete(h.connections, c)
				close(c.send)
			}
		case m := <-h.broadcast:
			// New pgnotification received.
			// Loop over all clients in hub.
			fmt.Println("hub:", m, h.connections)
			for c := range h.connections {
				// Only broadcast if the client is subscribed to the table.
				if c.subscriptions[m.Table] {
					// Send the notification. If the send fails (for example when the
					// buffer is full, close the connection. This means a bulk update
					// can close the connection with the websocket.
					select {
					case c.send <- m:
					default:
						close(c.send)
						delete(h.connections, c)
					}
				}
			}
		}
	}
}
