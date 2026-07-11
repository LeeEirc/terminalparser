package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/LeeEirc/terminalparser"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

var cfg = Config{}
var connMap = NewConnMap()

var terms sync.Map

func main() {
	LoadCfgFromEnv(&cfg)
	loadConfigFromFile("config.yml", &cfg)
	log.Printf("Config: %#v\n", cfg)
	http.HandleFunc("/ws/ssh/", HandleWsSSH)
	http.HandleFunc("/api/ssh/", HandleSSHResult)
	log.Fatal(http.ListenAndServe(":5858", nil))
}

var maxMessageSize = 1024
var upgrader = websocket.Upgrader{
	ReadBufferSize:  maxMessageSize,
	WriteBufferSize: maxMessageSize,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type windowSize struct {
	High  int `json:"high"`
	Width int `json:"width"`
}

type wsConnectMessage struct {
	Type   string `json:"type"`
	Config Config `json:"config"`
}

type wsResponse struct {
	Type    string `json:"type"`
	UUID    string `json:"uuid,omitempty"`
	Message string `json:"message,omitempty"`
}

func HandleWsSSH(w http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println("upgrader.Upgrade:", err)
		return
	}
	log.Println("upgrader.Upgrade:", conn.RemoteAddr())
	conn.SetReadDeadline(time.Time{})
	conn.SetWriteDeadline(time.Time{})

	defer conn.Close()
	msgType, payload, err := conn.ReadMessage()
	if err != nil {
		log.Println("conn.ReadMessage connect:", err)
		return
	}
	if msgType != websocket.TextMessage {
		writeWSError(conn, "first websocket message must be a connect text message")
		return
	}

	connectMsg := wsConnectMessage{}
	if err = json.Unmarshal(payload, &connectMsg); err != nil {
		writeWSError(conn, "invalid connect message json")
		return
	}
	if connectMsg.Type != "connect" {
		writeWSError(conn, "first websocket message type must be connect")
		return
	}

	sshClient, err2 := NewSSHClient(&connectMsg.Config, 100, 120)
	if err2 != nil {
		log.Println("NewSSHClient:", err2)
		writeWSError(conn, err2.Error())
		return
	}

	term, err := terminalparser.New(
		terminalparser.WithSize(100, 120),
		terminalparser.WithMaxScrollback(2000),
	)
	if err != nil {
		log.Println("terminalparser.New:", err)
		sshClient.Close()
		writeWSError(conn, "failed to create terminal")
		return
	}
	defer term.Close()

	uuid, err := generateUUID()
	if err != nil {
		log.Println("generateUUID:", err)
		sshClient.Close()
		writeWSError(conn, "failed to generate connection uuid")
		return
	}
	terms.Store(uuid, term)
	connState := NewConn(uuid, sshClient)

	if ok := connMap.Add(connState); !ok {
		sshClient.Close()
		writeWSError(conn, "connection uuid already exists")
		return
	}
	if err = conn.WriteJSON(wsResponse{Type: "connected", UUID: uuid}); err != nil {
		connState.Close()
		connMap.Delete(uuid)
		log.Println("conn.WriteJSON connected:", err)
		return
	}

	wg := sync.WaitGroup{}
	closeOnce := sync.Once{}
	cleanup := func() {
		closeOnce.Do(func() {
			terms.Delete(uuid)
			connState.Close()
			connMap.Delete(uuid)
			_ = conn.Close()
		})
	}
	defer cleanup()

	wg.Add(2)
	go func() {
		defer wg.Done()
		defer cleanup()
		for {
			msgType, p, err1 := conn.ReadMessage()
			if err1 != nil {
				log.Println("conn.ReadMessage:", err1)
				return
			}
			switch msgType {
			case websocket.TextMessage:
				if _, err = sshClient.Write(p); err != nil {
					log.Println("sshClient.Write:", err)
					return
				}
				break
			case websocket.BinaryMessage:
				var wdSize windowSize
				if err = json.Unmarshal(p, &wdSize); err != nil {
					continue
				}
				log.Println("wdSize:", wdSize)
				sshClient.Resize(wdSize.Width, wdSize.High)
				if err = term.Resize(uint16(wdSize.Width), uint16(wdSize.High), 0, 0); err != nil {
					log.Println("terminal.Resize:", err)
				}
			default:

			}
		}

	}()

	go func() {
		defer wg.Done()
		defer cleanup()
		buf := make([]byte, maxMessageSize)
		for {
			n, err1 := sshClient.Read(buf)
			if err1 != nil {
				log.Println("sshClient.Read:", err1)
				return
			}
			err3 := conn.WriteMessage(websocket.BinaryMessage, buf[:n])
			if err3 != nil {
				log.Println("conn.WriteMessage:", err3)
				return
			}
			connState.lock.Lock()
			if _, err = term.Write(buf[:n]); err != nil {
				log.Println("terminal.Write:", err)
			}
			connState.lock.Unlock()
		}
	}()

	wg.Wait()
}

func HandleSSHResult(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	prefix := "/api/ssh/"
	suffix := "/result"
	if !strings.HasPrefix(req.URL.Path, prefix) || !strings.HasSuffix(req.URL.Path, suffix) {
		http.NotFound(w, req)
		return
	}
	uuid := strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, prefix), suffix)
	uuid = strings.Trim(uuid, "/")
	if uuid == "" {
		http.NotFound(w, req)
		return
	}
	conn, ok := connMap.Get(uuid)
	if !ok {
		http.NotFound(w, req)
		return
	}
	conn.lock.Lock()
	defer conn.lock.Unlock()
	w.Header().Set("Content-Type", "application/json")
	value, ok := terms.Load(uuid)
	if !ok {
		http.NotFound(w, req)
		return
	}
	term := value.(*terminalparser.TerminalVT)
	alternate, err := term.IsScreenAlternate()
	if err != nil {
		log.Println("term.IsScreenAlternate:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if alternate {
		log.Println("term.IsScreenAlternate: alternate screen may be vim, vi, or tmux")
		title, err := term.Title()
		if err != nil {
			log.Println("term.Title err:", err)
		}
		log.Println("term.Title:", title)
		x, err := term.CursorX()
		if err != nil {
			log.Println("term.CursorX err", err)
		}
		log.Println("colx ", x)
		y, err := term.CursorY()
		if err != nil {
			log.Println("term.CursorY err", err)
		}
		log.Println("term.CursorY:", y)
	}

	ret, err := term.String()
	if err != nil {
		log.Println("terminal.String:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = io.WriteString(w, ret)
	log.Printf("HandleSSHResult: served result for uuid %s\n", uuid)
}

func writeWSError(conn *websocket.Conn, message string) {
	if err := conn.WriteJSON(wsResponse{Type: "error", Message: message}); err != nil {
		log.Println("conn.WriteJSON error:", err)
	}
}

func generateUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	), nil
}

func LoadCfgFromEnv(conf *Config) {
	viper.AutomaticEnv()
	envViper := viper.New()
	for _, item := range os.Environ() {
		envItem := strings.SplitN(item, "=", 2)
		if len(envItem) == 2 {
			envViper.Set(envItem[0], viper.Get(envItem[0]))
		}
	}
	if err := envViper.Unmarshal(conf); err == nil {
		log.Printf("Load config from env %+v\n", conf)

	}
}

func loadConfigFromFile(path string, conf *Config) {
	var err error
	fileViper := viper.New()
	fileViper.SetConfigFile(path)
	if err = fileViper.ReadInConfig(); err == nil {
		if err = fileViper.Unmarshal(conf); err == nil {
			log.Printf("Load config from %s success\n", path)
		}
	}
}
