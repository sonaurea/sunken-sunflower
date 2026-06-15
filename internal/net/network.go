package net

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// ─── Multiplayer Networking ─────────────────────────────────────────
// Peer-to-peer networking for up to 4 players.
// Supports: LAN discovery, direct IP connect, Steam lobby (future).

const (
	ProtocolVersion  = 1
	MaxPlayers       = 4
	DefaultPort      = 14250
	DiscoveryPort    = 14251
	BroadcastAddr    = "255.255.255.255"
	SyncRate         = 60 // Hz (matches game TPS)
	DisconnectTimeout = 5 * time.Second
)

// ─── Peer ID ────────────────────────────────────────────────────────

// PeerID is a unique identifier for each connected player.
type PeerID [8]byte

func NewPeerID() PeerID {
	var id PeerID
	if _, err := rand.Read(id[:]); err != nil {
		log.Printf("[net] warning: failed to generate random peer ID: %v", err)
	}
	return id
}

func (id PeerID) String() string {
	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x%02x%02x",
		id[0], id[1], id[2], id[3], id[4], id[5], id[6], id[7])
}

// ─── Message Types ─────────────────────────────────────────────────

type MessageType byte

const (
	MsgHeartbeat  MessageType = iota
	MsgJoin
	MsgLeave
	MsgStateSync
	MsgInput
	MsgChat
	MsgReady
	MsgStartGame
	MsgPing
	MsgPong
)

// Message is the wire format for all network communication.
type Message struct {
	Type      MessageType     `json:"t"`
	SenderID  PeerID          `json:"s"`
	Timestamp int64           `json:"ts"`
	SeqNum    uint32          `json:"seq"`
	Payload   json.RawMessage `json:"p,omitempty"`
}

// ─── Player State ──────────────────────────────────────────────────

type PlayerState struct {
	PeerID    PeerID  `json:"id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Health    float64 `json:"hp"`
	Direction int     `json:"dir"`
	IsDashing bool    `json:"dash"`
	InputSeq  uint32  `json:"seq"`
	ColorIdx  int     `json:"color"` // 0-3 for player color differentiation
}

// GameStateSync is the full game state sent at SyncRate.
type GameStateSync struct {
	Players     []PlayerState `json:"players"`
	Depth       int           `json:"depth"`
	RoomID      int           `json:"room"`
	Tick        uint32        `json:"tick"`
	ElapsedTime float64       `json:"time"`
}

// InputPayload is a player's input for a single tick.
type InputPayload struct {
	Left   bool    `json:"l"`
	Right  bool    `json:"r"`
	Up     bool    `json:"u"`
	Down   bool    `json:"d"`
	Dash   bool    `json:"h"`
	Action bool    `json:"a"`
	Cancel bool    `json:"c"`
	SeqNum uint32  `json:"seq"`
}

// ─── Lobby ─────────────────────────────────────────────────────────

type Lobby struct {
	ID           string      `json:"id"`
	HostPeerID   PeerID      `json:"host"`
	Players      []PlayerInfo `json:"players"`
	MaxPlayers   int         `json:"max"`
	GameStarted  bool        `json:"started"`
	JoinPassword string      `json:"pw,omitempty"`
}

type PlayerInfo struct {
	PeerID  PeerID `json:"id"`
	Name    string `json:"name"`
	IsHost  bool   `json:"host"`
	Ready   bool   `json:"ready"`
	Latency int    `json:"ms"`
}

// ─── Network Manager ───────────────────────────────────────────────

// NetMode indicates how this instance operates.
type NetMode int

const (
	ModeOffline  NetMode = iota // Single-player, no networking
	ModeHost                    // Hosts a game, accepts connections
	ModeClient                  // Connects to a host
)

// NetworkManager handles all peer-to-peer networking.
type NetworkManager struct {
	mu          sync.RWMutex
	mode        NetMode
	peerID      PeerID
	players     map[PeerID]*PeerConn
	playerOrder []PeerID // deterministic order
	udpConn     *net.UDPConn
	lobby       *Lobby

	// Callbacks
	OnPlayerJoin    func(peerID PeerID)
	OnPlayerLeave   func(peerID PeerID)
	OnStateReceived func(state GameStateSync)
	OnInputReceived func(peerID PeerID, input InputPayload)

	// Host-only: pending connections
	pendingConnections []PeerID

	// Sync state
	tick       uint32
	gameState  GameStateSync
	inputQueue map[PeerID][]InputPayload // buffered inputs per client

	running  bool
	stopChan chan struct{}

	// Statistics
	BytesSent     int64
	BytesReceived int64
	PacketsSent   int64
	PacketsLost   int64
}

// PeerConn tracks a single connected peer.
type PeerConn struct {
	PlayerInfo
	Addr        *net.UDPAddr
	LastSeen    time.Time
	Latency     time.Duration
	SeqNum      uint32
	InputBuffer []InputPayload
}

func NewNetworkManager() *NetworkManager {
	return &NetworkManager{
		peerID:      NewPeerID(),
		players:     make(map[PeerID]*PeerConn),
		stopChan:    make(chan struct{}),
		inputQueue:  make(map[PeerID][]InputPayload),
	}
}

// ─── Host Mode ─────────────────────────────────────────────────────

// StartHost initializes this instance as a game host.
func (nm *NetworkManager) StartHost(port int) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if nm.running {
		return fmt.Errorf("already running")
	}

	if port == 0 {
		port = DefaultPort
	}

	addr := &net.UDPAddr{Port: port}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	nm.mode = ModeHost
	nm.udpConn = conn
	nm.running = true
	nm.lobby = &Lobby{
		ID:         nm.peerID.String(),
		HostPeerID: nm.peerID,
		MaxPlayers: MaxPlayers,
		Players: []PlayerInfo{
			{PeerID: nm.peerID, Name: "Host", IsHost: true, Ready: true},
		},
	}

	// Add host as player 0
	nm.playerOrder = append(nm.playerOrder, nm.peerID)

	go nm.receiveLoop()
	go nm.broadcastLoop()

	log.Printf("[net] Host started on port %d (peer: %s)", port, nm.peerID.String())
	return nil
}

// ─── Client Mode ───────────────────────────────────────────────────

// Connect connects to a remote host.
func (nm *NetworkManager) Connect(host string, port int) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if nm.running {
		return fmt.Errorf("already connected")
	}

	if port == 0 {
		port = DefaultPort
	}

	raddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return fmt.Errorf("resolve: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	nm.mode = ModeClient
	nm.udpConn = conn
	nm.running = true

	// Send join request
	joinMsg := Message{
		Type:     MsgJoin,
		SenderID: nm.peerID,
		SeqNum:   0,
	}
	data, _ := json.Marshal(joinMsg)
	if _, err := conn.Write(data); err != nil {
		log.Printf("[net] failed to send join message: %v", err)
	}

	go nm.receiveLoop()

	log.Printf("[net] Connected to %s:%d (peer: %s)", host, port, nm.peerID.String())
	return nil
}

// ─── Disconnect ────────────────────────────────────────────────────

func (nm *NetworkManager) Disconnect() {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if !nm.running {
		return
	}

	// Send leave message
	leaveMsg := Message{
		Type:     MsgLeave,
		SenderID: nm.peerID,
	}
	data, _ := json.Marshal(leaveMsg)
	if nm.udpConn != nil {
		if _, err := nm.udpConn.Write(data); err != nil {
			log.Printf("[net] failed to send leave message: %v", err)
		}
		nm.udpConn.Close()
	}

	close(nm.stopChan)
	nm.running = false
	nm.mode = ModeOffline

	log.Printf("[net] Disconnected")
}

// ─── Send / Receive ────────────────────────────────────────────────

func (nm *NetworkManager) sendTo(addr *net.UDPAddr, msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	n, err := nm.udpConn.WriteTo(data, addr)
	if err != nil {
		nm.PacketsLost++
		return err
	}
	nm.BytesSent += int64(n)
	nm.PacketsSent++
	return nil
}

func (nm *NetworkManager) broadcast(msg Message) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	for _, conn := range nm.players {
		if conn.PeerID == nm.peerID {
			continue // skip self
		}
		if err := nm.sendTo(conn.Addr, msg); err != nil {
			nm.PacketsLost++
		}
	}
}

func (nm *NetworkManager) receiveLoop() {
	buf := make([]byte, 2048)
	for {
		select {
		case <-nm.stopChan:
			return
		default:
		}

		_ = nm.udpConn.SetReadDeadline(time.Now().Add(time.Second))
		n, addr, err := nm.udpConn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if !nm.running {
				return
			}
			continue
		}

		nm.BytesReceived += int64(n)

		var msg Message
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			continue
		}

		nm.handleMessage(msg, addr)
	}
}

func (nm *NetworkManager) handleMessage(msg Message, addr *net.UDPAddr) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	switch msg.Type {
	case MsgJoin:
		if nm.mode != ModeHost {
			return
		}
		// Check if lobby is full
		if len(nm.lobby.Players) >= MaxPlayers {
			log.Printf("[net] Rejected join from %s: lobby full", addr)
			return
		}
		// Check for duplicate
		for _, p := range nm.lobby.Players {
			if p.PeerID == msg.SenderID {
				return // already in lobby
			}
		}
		// Add player
		nm.lobby.Players = append(nm.lobby.Players, PlayerInfo{
			PeerID: msg.SenderID,
			Name:   fmt.Sprintf("Player %d", len(nm.lobby.Players)),
			Ready:  true,
		})
		nm.playerOrder = append(nm.playerOrder, msg.SenderID)
		nm.players[msg.SenderID] = &PeerConn{
			PlayerInfo: nm.lobby.Players[len(nm.lobby.Players)-1],
			Addr:       addr,
			LastSeen:   time.Now(),
		}
		// Broadcast lobby state to all
		nm.broadcastLobby()
		log.Printf("[net] Player joined: %s (total: %d)", msg.SenderID.String(), len(nm.lobby.Players))
		if nm.OnPlayerJoin != nil {
			nm.OnPlayerJoin(msg.SenderID)
		}

	case MsgLeave:
		delete(nm.players, msg.SenderID)
		for i, p := range nm.lobby.Players {
			if p.PeerID == msg.SenderID {
				nm.lobby.Players = append(nm.lobby.Players[:i], nm.lobby.Players[i+1:]...)
				break
			}
		}
		for i, id := range nm.playerOrder {
			if id == msg.SenderID {
				nm.playerOrder = append(nm.playerOrder[:i], nm.playerOrder[i+1:]...)
				break
			}
		}
		nm.broadcastLobby()
		log.Printf("[net] Player left: %s", msg.SenderID.String())
		if nm.OnPlayerLeave != nil {
			nm.OnPlayerLeave(msg.SenderID)
		}

	case MsgStateSync:
		var state GameStateSync
		if err := json.Unmarshal(msg.Payload, &state); err == nil {
			if nm.OnStateReceived != nil {
				nm.OnStateReceived(state)
			}
		}

	case MsgInput:
		if nm.mode != ModeHost {
			return
		}
		var input InputPayload
		if err := json.Unmarshal(msg.Payload, &input); err == nil {
			nm.inputQueue[msg.SenderID] = append(nm.inputQueue[msg.SenderID], input)
			if nm.OnInputReceived != nil {
				nm.OnInputReceived(msg.SenderID, input)
			}
		}

	case MsgPing:
		// Respond with pong
		pong := Message{Type: MsgPong, SenderID: nm.peerID}
		if err := nm.sendTo(addr, pong); err != nil {
			nm.PacketsLost++
		}

	case MsgPong:
		if conn, ok := nm.players[msg.SenderID]; ok {
			conn.Latency = time.Since(time.Unix(0, msg.Timestamp))
		}
	}
}

func (nm *NetworkManager) broadcastLoop() {
	ticker := time.NewTicker(time.Second / SyncRate)
	defer ticker.Stop()

	for range ticker.C {
		if !nm.running {
			return
		}
		nm.mu.RLock()
		mode := nm.mode
		nm.mu.RUnlock()

		if mode == ModeHost {
			nm.tick++
			nm.broadcastState()
		}
	}
}

func (nm *NetworkManager) broadcastState() {
	nm.mu.RLock()
	state := nm.gameState
	state.Tick = nm.tick
	nm.mu.RUnlock()

	payload, _ := json.Marshal(state)
	msg := Message{
		Type:     MsgStateSync,
		SenderID: nm.peerID,
		SeqNum:   nm.tick,
		Payload:  payload,
	}
	nm.broadcast(msg)
}

func (nm *NetworkManager) broadcastLobby() {
	if nm.lobby == nil {
		return
	}
	// Send lobby state to all connected clients
	payload, _ := json.Marshal(nm.lobby)
	msg := Message{
		Type:     MsgStateSync,
		SenderID: nm.peerID,
		Payload:  payload,
	}
	for _, conn := range nm.players {
		if conn.PeerID != nm.peerID {
			if err := nm.sendTo(conn.Addr, msg); err != nil {
				nm.PacketsLost++
			}
		}
	}
}

// ─── Public API ────────────────────────────────────────────────────

func (nm *NetworkManager) Mode() NetMode {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.mode
}

func (nm *NetworkManager) PeerID() PeerID {
	return nm.peerID
}

func (nm *NetworkManager) Lobby() *Lobby {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.lobby
}

func (nm *NetworkManager) Players() []PlayerInfo {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	if nm.lobby == nil {
		return nil
	}
	return nm.lobby.Players
}

func (nm *NetworkManager) PlayerCount() int {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return len(nm.players) + 1 // +1 for self
}

func (nm *NetworkManager) IsHost() bool {
	return nm.mode == ModeHost
}

func (nm *NetworkManager) IsConnected() bool {
	return nm.mode != ModeOffline
}

// SendInput sends the local player's input to the host.
func (nm *NetworkManager) SendInput(input InputPayload) {
	if nm.mode != ModeClient {
		return
	}
	payload, _ := json.Marshal(input)
	msg := Message{
		Type:     MsgInput,
		SenderID: nm.peerID,
		Payload:  payload,
	}
	nm.mu.RLock()
	conn := nm.udpConn
	nm.mu.RUnlock()
	if conn != nil {
		data, _ := json.Marshal(msg)
		if _, err := conn.Write(data); err != nil {
			log.Printf("[net] failed to send input: %v", err)
		}
	}
}

// UpdateGameState sets the authoritative game state (host only).
func (nm *NetworkManager) UpdateGameState(state GameStateSync) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.gameState = state
}

// GetInputs returns all buffered inputs since last call (host only).
func (nm *NetworkManager) GetInputs() map[PeerID][]InputPayload {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	inputs := nm.inputQueue
	nm.inputQueue = make(map[PeerID][]InputPayload)
	return inputs
}

// ─── LAN Discovery ─────────────────────────────────────────────────

type DiscoveredHost struct {
	PeerID     PeerID `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"addr"`
	Port      int    `json:"port"`
	PlayerCount int   `json:"players"`
	MaxPlayers int   `json:"max"`
}

// DiscoverLAN scans the local network for game hosts.
func DiscoverLAN(timeout time.Duration) ([]DiscoveredHost, error) {
	addr := &net.UDPAddr{Port: DiscoveryPort}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen discovery: %w", err)
	}
	defer conn.Close()

	var hosts []DiscoveredHost
	seen := make(map[string]bool)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		buf := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		var host DiscoveredHost
		if err := json.Unmarshal(buf[:n], &host); err == nil {
			key := host.Address
			if !seen[key] {
				seen[key] = true
				hosts = append(hosts, host)
			}
		}
	}

	return hosts, nil
}

// AdvertiseLAN broadcasts this host's presence on the LAN.
func (nm *NetworkManager) AdvertiseLAN() {
	addr := &net.UDPAddr{IP: net.ParseIP(BroadcastAddr), Port: DiscoveryPort}
	host := DiscoveredHost{
		PeerID:     nm.peerID,
		Name:      "Sunken Sunflower",
		Port:      DefaultPort,
		PlayerCount: nm.PlayerCount(),
		MaxPlayers: MaxPlayers,
	}
	data, _ := json.Marshal(host)
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return
	}
	defer conn.Close()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if !nm.running {
			return
		}
		if _, err := conn.Write(data); err != nil {
			log.Printf("[net] failed to advertise: %v", err)
		}
	}
}