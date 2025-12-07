// Package irc provides a Twitch IRC client for chat message ingestion.
package irc

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"math/rand/v2"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	// TwitchIRCServerTLS is the Twitch IRC server address for TLS connections.
	TwitchIRCServerTLS = "irc.chat.twitch.tv:6697"

	// TwitchIRCServer is the Twitch IRC server address (non-TLS, deprecated).
	// Prefer TwitchIRCServerTLS for secure connections.
	TwitchIRCServer = "irc.chat.twitch.tv:6667"

	// reconnect settings
	initialReconnectDelay = 1 * time.Second
	maxReconnectDelay     = 30 * time.Second
	reconnectBackoffMult  = 2
	maxReconnectAttempts  = 10  // Maximum attempts before giving up temporarily
	reconnectJitterFactor = 0.2 // 20% jitter to prevent thundering herd
)

// AuthMode represents the IRC authentication mode.
type AuthMode string

const (
	// AuthModeAuthenticated uses OAuth token for full access.
	AuthModeAuthenticated AuthMode = "authenticated"
	// AuthModeAnonymous uses justinfan nick for read-only access.
	AuthModeAnonymous AuthMode = "anonymous"
)

// Message represents a parsed IRC PRIVMSG.
type Message struct {
	Channel     string
	Username    string
	DisplayName string
	Text        string
	Tags        map[string]string
	ReceivedAt  time.Time
}

// MessageHandler is called for each incoming chat message.
type MessageHandler func(msg Message)

// ChannelChangeHandler is called when channel join/part is requested.
type ChannelChangeHandler func(channel string, joined bool)

// Client is a Twitch IRC client.
type Client struct {
	authMode   AuthMode
	username   string
	oauthToken string

	conn   net.Conn
	reader *bufio.Reader

	mu             sync.RWMutex
	channels       map[string]bool
	connected      bool
	reconnecting   bool
	reconnectDelay time.Duration

	onMessage       MessageHandler
	onChannelChange ChannelChangeHandler

	done chan struct{}
	wg   sync.WaitGroup
}

// ClientConfig holds IRC client configuration.
type ClientConfig struct {
	AuthMode        AuthMode // "authenticated" or "anonymous"
	Username        string   // Required for authenticated, optional for anonymous
	OAuthToken      string   // Required for authenticated, must be empty for anonymous
	OnMessage       MessageHandler
	OnChannelChange ChannelChangeHandler
}

// NewClient creates a new IRC client.
func NewClient(cfg ClientConfig) *Client {
	username := cfg.Username

	// For anonymous mode, generate a justinfan nick if not provided
	if cfg.AuthMode == AuthModeAnonymous && username == "" {
		username = fmt.Sprintf("justinfan%d", rand.IntN(99999)+1)
	}

	return &Client{
		authMode:        cfg.AuthMode,
		username:        username,
		oauthToken:      cfg.OAuthToken,
		channels:        make(map[string]bool),
		reconnectDelay:  initialReconnectDelay,
		onMessage:       cfg.OnMessage,
		onChannelChange: cfg.OnChannelChange,
		done:            make(chan struct{}),
	}
}

// Connect establishes a TLS connection to Twitch IRC.
// The context is used for cancellation during the connection process.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// Check for context cancellation before connecting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Use TLS for secure connection (port 6697)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", TwitchIRCServerTLS, &tls.Config{
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to Twitch IRC (TLS): %w", err)
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)

	// Request capabilities (works for both auth modes)
	if err := c.send("CAP REQ :twitch.tv/tags twitch.tv/commands"); err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to request capabilities: %w", err)
	}

	// Authenticate based on mode
	if c.authMode == AuthModeAuthenticated {
		// Authenticated mode: send PASS with OAuth token, then NICK
		if err := c.send("PASS " + c.oauthToken); err != nil {
			c.conn.Close()
			return fmt.Errorf("failed to send password: %w", err)
		}
	}
	// For both modes, send NICK (anonymous uses justinfan nick)
	if err := c.send("NICK " + c.username); err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to send nick: %w", err)
	}

	c.connected = true
	c.reconnectDelay = initialReconnectDelay

	// Start read loop
	c.wg.Add(1)
	go c.readLoop()

	return nil
}

// Disconnect closes the IRC connection.
func (c *Client) Disconnect() error {
	c.mu.Lock()

	// Check if already disconnected (done channel closed)
	select {
	case <-c.done:
		// Already closed, nothing to do
		c.mu.Unlock()
		return nil
	default:
	}

	// Signal all goroutines to stop
	close(c.done)
	c.connected = false

	// Close connection to unblock any pending reads
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()

	// Wait for all goroutines (readLoop and reconnect) to finish
	c.wg.Wait()
	return nil
}

// Join joins a channel.
func (c *Client) Join(channel string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	channel = normalizeChannel(channel)

	if c.channels[channel] {
		return nil // Already joined
	}

	if c.connected {
		if err := c.send("JOIN " + channel); err != nil {
			return fmt.Errorf("failed to join channel %s: %w", channel, err)
		}
	}

	c.channels[channel] = true

	if c.onChannelChange != nil {
		go c.onChannelChange(channel, true)
	}

	return nil
}

// Part leaves a channel.
func (c *Client) Part(channel string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	channel = normalizeChannel(channel)

	if !c.channels[channel] {
		return nil // Not in channel
	}

	if c.connected {
		if err := c.send("PART " + channel); err != nil {
			return fmt.Errorf("failed to part channel %s: %w", channel, err)
		}
	}

	delete(c.channels, channel)

	if c.onChannelChange != nil {
		go c.onChannelChange(channel, false)
	}

	return nil
}

// IsConnected returns true if the client is connected.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Channels returns a list of currently joined channels.
func (c *Client) Channels() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	channels := make([]string, 0, len(c.channels))
	for ch := range c.channels {
		channels = append(channels, ch)
	}
	return channels
}

// IsAnonymous returns true if the client is using anonymous (justinfan) mode.
// In anonymous mode, the client is read-only and cannot send messages.
func (c *Client) IsAnonymous() bool {
	return c.authMode == AuthModeAnonymous
}

// AuthMode returns the current authentication mode.
func (c *Client) AuthMode() AuthMode {
	return c.authMode
}

func (c *Client) send(msg string) error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	_, err := c.conn.Write([]byte(msg + "\r\n"))
	return err
}

func (c *Client) readLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.done:
			return
		default:
		}

		c.conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		line, err := c.reader.ReadString('\n')
		if err != nil {
			c.handleDisconnect()
			return
		}

		line = strings.TrimSpace(line)
		c.handleLine(line)
	}
}

func (c *Client) handleLine(line string) {
	// Handle PING
	if strings.HasPrefix(line, "PING") {
		c.mu.Lock()
		c.send("PONG" + line[4:])
		c.mu.Unlock()
		return
	}

	// Handle NOTICE (may contain auth failures or rate limit warnings)
	if strings.Contains(line, "NOTICE") {
		// Common auth failure messages:
		// - "Login authentication failed"
		// - "Improperly formatted auth"
		// Rate limit messages typically contain "You are sending"
		// For now, we log these internally but don't take action
		// (could add callback for error handling in future)
	}

	// Parse PRIVMSG
	if strings.Contains(line, "PRIVMSG") {
		msg := c.parseMessage(line)
		if msg != nil && c.onMessage != nil {
			c.onMessage(*msg)
		}
	}
}

func (c *Client) parseMessage(line string) *Message {
	msg := &Message{
		Tags:       make(map[string]string),
		ReceivedAt: time.Now().UTC(),
	}

	// Parse tags if present
	if strings.HasPrefix(line, "@") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			return nil
		}
		tagStr := parts[0][1:]
		line = parts[1]

		for _, tag := range strings.Split(tagStr, ";") {
			kv := strings.SplitN(tag, "=", 2)
			if len(kv) == 2 {
				msg.Tags[kv[0]] = kv[1]
			}
		}

		if dn, ok := msg.Tags["display-name"]; ok {
			msg.DisplayName = dn
		}
	}

	// Parse :user!user@user.tmi.twitch.tv PRIVMSG #channel :message
	parts := strings.SplitN(line, " ", 4)
	if len(parts) < 4 {
		return nil
	}

	// Extract username
	if strings.HasPrefix(parts[0], ":") {
		userParts := strings.SplitN(parts[0][1:], "!", 2)
		if len(userParts) >= 1 {
			msg.Username = strings.ToLower(userParts[0])
		}
	}

	// Extract channel
	if strings.HasPrefix(parts[2], "#") {
		msg.Channel = strings.ToLower(parts[2])
	}

	// Extract message text
	if strings.HasPrefix(parts[3], ":") {
		msg.Text = parts[3][1:]
	} else {
		msg.Text = parts[3]
	}

	return msg
}

func (c *Client) handleDisconnect() {
	c.mu.Lock()
	c.connected = false
	c.reconnecting = true
	c.mu.Unlock()

	// Attempt reconnection with exponential backoff
	// Track in WaitGroup so Disconnect() waits for it
	c.wg.Add(1)
	go c.reconnect()
}

func (c *Client) reconnect() {
	defer c.wg.Done()

	attempts := 0
	for {
		select {
		case <-c.done:
			return
		default:
		}

		// Check if we've exceeded max attempts
		attempts++
		if attempts > maxReconnectAttempts {
			// Reset delay and wait longer before trying again
			c.mu.Lock()
			c.reconnectDelay = maxReconnectDelay
			c.mu.Unlock()
			attempts = 0
		}

		c.mu.RLock()
		delay := c.reconnectDelay
		c.mu.RUnlock()

		// Add jitter to prevent thundering herd (±20% randomization)
		jitter := time.Duration(float64(delay) * reconnectJitterFactor * (2*rand.Float64() - 1))
		sleepTime := delay + jitter
		if sleepTime < 0 {
			sleepTime = delay
		}

		// Use select with timer instead of time.Sleep to allow cancellation
		timer := time.NewTimer(sleepTime)
		select {
		case <-c.done:
			timer.Stop()
			return
		case <-timer.C:
		}

		if err := c.Connect(context.Background()); err != nil {
			c.mu.Lock()
			c.reconnectDelay = min(c.reconnectDelay*reconnectBackoffMult, maxReconnectDelay)
			c.mu.Unlock()
			continue
		}

		// Rejoin channels
		c.mu.RLock()
		channels := make([]string, 0, len(c.channels))
		for ch := range c.channels {
			channels = append(channels, ch)
		}
		c.mu.RUnlock()

		for _, ch := range channels {
			c.mu.Lock()
			c.send("JOIN " + ch)
			c.mu.Unlock()
		}

		c.mu.Lock()
		c.reconnecting = false
		c.mu.Unlock()
		return
	}
}

func normalizeChannel(channel string) string {
	channel = strings.ToLower(strings.TrimSpace(channel))
	if !strings.HasPrefix(channel, "#") {
		channel = "#" + channel
	}
	return channel
}
