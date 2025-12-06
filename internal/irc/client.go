// Package irc provides a Twitch IRC client for chat message ingestion.
package irc

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	// TwitchIRCServer is the Twitch IRC server address.
	TwitchIRCServer = "irc.chat.twitch.tv:6667"

	// reconnect settings
	initialReconnectDelay = 1 * time.Second
	maxReconnectDelay     = 30 * time.Second
	reconnectBackoffMult  = 2
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
	Username        string
	OAuthToken      string
	OnMessage       MessageHandler
	OnChannelChange ChannelChangeHandler
}

// NewClient creates a new IRC client.
func NewClient(cfg ClientConfig) *Client {
	return &Client{
		username:        cfg.Username,
		oauthToken:      cfg.OAuthToken,
		channels:        make(map[string]bool),
		reconnectDelay:  initialReconnectDelay,
		onMessage:       cfg.OnMessage,
		onChannelChange: cfg.OnChannelChange,
		done:            make(chan struct{}),
	}
}

// Connect establishes a connection to Twitch IRC.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	conn, err := net.DialTimeout("tcp", TwitchIRCServer, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to Twitch IRC: %w", err)
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)

	// Authenticate
	if err := c.send("CAP REQ :twitch.tv/tags twitch.tv/commands"); err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to request capabilities: %w", err)
	}
	if err := c.send("PASS " + c.oauthToken); err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to send password: %w", err)
	}
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
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	close(c.done)
	c.connected = false

	if c.conn != nil {
		c.conn.Close()
	}

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
	go c.reconnect()
}

func (c *Client) reconnect() {
	for {
		select {
		case <-c.done:
			return
		default:
		}

		c.mu.RLock()
		delay := c.reconnectDelay
		c.mu.RUnlock()

		time.Sleep(delay)

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

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
