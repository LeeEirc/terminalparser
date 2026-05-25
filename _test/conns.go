package main

import (
	"sync"
	"time"
)

type ConnMap struct {
	lock  sync.RWMutex
	conns map[string]*Conn
}

type Conn struct {
	UUID      string
	SSHClient *SSHClient
	Parser    *TerminalParser

	lock      sync.RWMutex
	result    ParseResult
	connected bool
	createdAt time.Time
	updatedAt time.Time
	closed    bool
}

type ParseResult struct {
	UUID      string    `json:"uuid"`
	Connected bool      `json:"connected"`
	Command   string    `json:"command"`
	Output    string    `json:"output"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func NewConnMap() *ConnMap {
	return &ConnMap{
		conns: make(map[string]*Conn),
	}
}

func (m *ConnMap) Add(conn *Conn) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, ok := m.conns[conn.UUID]; ok {
		return false
	}
	m.conns[conn.UUID] = conn
	return true
}

func (m *ConnMap) Get(uuid string) (*Conn, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	conn, ok := m.conns[uuid]
	return conn, ok
}

func (m *ConnMap) Delete(uuid string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.conns, uuid)
}

func NewConn(uuid string, sshClient *SSHClient) *Conn {
	now := time.Now()
	return &Conn{
		UUID:      uuid,
		SSHClient: sshClient,
		connected: true,
		createdAt: now,
		updatedAt: now,
		result: ParseResult{
			UUID:      uuid,
			Connected: true,
			UpdatedAt: now,
		},
	}
}

func (c *Conn) Close() {
	c.lock.Lock()
	if c.closed {
		c.lock.Unlock()
		return
	}
	c.closed = true
	c.connected = false
	c.result.Connected = false
	c.result.UpdatedAt = time.Now()
	c.lock.Unlock()

	if c.SSHClient != nil {
		c.SSHClient.Close()
	}
}
