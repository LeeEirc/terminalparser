package main

import (
	"fmt"
	"io"
	"log"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Username string `mapstructure:"USERNAME"`
	Password string `mapstructure:"PASSWORD"`
	Host     string `mapstructure:"HOST"`
	Port     int    `mapstructure:"PORT"`
}

func GetSSHClient(cfg *Config) *ssh.Client {
	var auth ssh.AuthMethod
	if cfg.Password != "" {
		auth = ssh.Password(cfg.Password)
	}

	sshCfg := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	dst := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client, err := ssh.Dial("tcp", dst, sshCfg)
	if err != nil {
		panic(err)
	}
	return client
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

func NewSSHClient(cfg *Config, w, h int) (*SSHClient, error) {
	client := GetSSHClient(cfg)
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return nil, err
	}
	terminalModes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing (different from the example in docs)
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	err = session.RequestPty("xterm", w, h, terminalModes)
	if err != nil {
		return nil, err
	}

	if err = session.Shell(); err != nil {
		return nil, err
	}

	return &SSHClient{
		client:  client,
		session: session,
		stdin:   stdin,
		stdout:  stdout,
	}, nil

}
