package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Username string   `json:"username" mapstructure:"USERNAME"`
	Password string   `json:"password" mapstructure:"PASSWORD"`
	Host     string   `json:"host" mapstructure:"HOST"`
	Port     int      `json:"port" mapstructure:"PORT"`
	Commands []string `json:"commands" mapstructure:"COMMANDS"`
}

func (cfg *Config) Normalize() {
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Username = strings.TrimSpace(cfg.Username)
	if cfg.Port == 0 {
		cfg.Port = 22
	}
}

func (cfg *Config) Validate() error {
	cfg.Normalize()
	if cfg.Host == "" {
		return errors.New("ssh host is required")
	}
	if cfg.Username == "" {
		return errors.New("ssh username is required")
	}
	if cfg.Password == "" {
		return errors.New("ssh password is required")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("ssh port %d is invalid", cfg.Port)
	}
	return nil
}

func GetSSHClient(cfg *Config) (*ssh.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	auth := ssh.Password(cfg.Password)

	sshCfg := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	dst := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client, err := ssh.Dial("tcp", dst, sshCfg)
	if err != nil {
		return nil, err
	}
	return client, nil
}

type SSHClient struct {
	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.ReadCloser
}

func (s *SSHClient) Resize(w, h int) {
	if err := s.session.WindowChange(h, w); err != nil {
		log.Default().Printf("ssh: failed to change window size: %v", err)
	}
}

func (s *SSHClient) Write(p []byte) (int, error) {
	return s.stdin.Write(p)
}

func (s *SSHClient) Read(p []byte) (int, error) {
	nr, err := s.stdout.Read(p)
	return nr, err
}

func (s *SSHClient) Close() {
	if s.session != nil {
		_ = s.session.Close()
	}
	if s.client != nil {
		_ = s.client.Close()
	}
}

func NewSSHClient(cfg *Config, w, h int) (*SSHClient, error) {
	client, err := GetSSHClient(cfg)
	if err != nil {
		return nil, err
	}
	session, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, err
	}
	terminalModes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing (different from the example in docs)
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	err = session.RequestPty("xterm", h, w, terminalModes)
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, err
	}
	if err = session.Shell(); err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, err
	}
	return &SSHClient{
		client:  client,
		session: session,
		stdin:   stdin,
		stdout:  stdout,
	}, nil

}
