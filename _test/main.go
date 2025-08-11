package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

var cfg = Config{}

func main() {
	LoadCfgFromEnv(&cfg)
	loadConfigFromFile("config.yml", &cfg)
	log.Printf("Config: %#v\n", cfg)
	http.HandleFunc("/ws/ssh/", HandleWsSSH)
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
	wg := sync.WaitGroup{}
	sshClient, err2 := NewSSHClient(&cfg, 120, 100)
	if err2 != nil {
		log.Println(err2)
		return
	}
	defer sshClient.client.Close()
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			msgType, p, err1 := conn.ReadMessage()
			if err1 != nil {
				log.Println("conn.ReadMessage:", err1)
				return
			}
			switch msgType {
			case websocket.TextMessage:
				_, _ = sshClient.Write(p)
				break
			case websocket.BinaryMessage:
				var wdSize windowSize
				if err = json.Unmarshal(p, &wdSize); err != nil {
					continue
				}
				log.Println("wdSize:", wdSize)
				sshClient.Resize(wdSize.Width, wdSize.High)
			default:

			}
		}

	}()

	go func() {
		defer wg.Done()
		buf := make([]byte, maxMessageSize)
		for {
			n, err1 := sshClient.Read(buf)
			if err1 != nil {
				log.Println("sshClient.Read:", err)
				return
			}
			err3 := conn.WriteMessage(websocket.TextMessage, buf[:n])
			if err3 != nil {
				log.Println("conn.WriteMessage:", err3)
				return
			}
		}
	}()

	wg.Wait()
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
