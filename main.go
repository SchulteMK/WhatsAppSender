package main

import (
	"database/sql"
	"encoding/gob"
	"fmt"
	"github.com/Baozisoftware/qrcode-terminal-go"
	"github.com/Rhymen/go-whatsapp"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	x := &handler{}
	var err error

	//create new WhatsApp connection
	x.wac, err = whatsapp.NewConn(5 * time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating connection: %v\n", err)
		return
	}

	//Add handler
	x.wac.AddHandler(x)

	err = login(x.wac)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error logging in: %v\n", err)
		return
	}

	x.startingTime = uint64(time.Now().Unix())
	root := "toSend/"
	for {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() == false && filepath.Ext(path) == ".txt" && info.Size() > 0 {
				b, _ := ioutil.ReadFile(path)
				msg := whatsapp.TextMessage{
					Info: whatsapp.MessageInfo{
						RemoteJid: info.Name()[:len(info.Name())-4],
					},
					Text: string(b),
				}
				_, err := x.wac.Send(msg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error sending: %v", err)
				}
				_ = os.Remove(path)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error in read loop: %v", err)
			break
		}
		<-time.After(5 * time.Second)
	}
}

type handler struct {
	db           *sql.DB
	wac          *whatsapp.Conn
	handlers     []MessageHandler
	startingTime uint64
}

type MediaMetaData struct {
	Id        int
	Caption   string
	Mimetype  string
	Thumbnail []byte
}

type MessageHandler struct {
	Url      string
	Audio    bool
	Document bool
	Image    bool
	Text     bool
	Video    bool
}

type MessageType int8

const (
	Text MessageType = iota
	Image
	Video
	Audio
	Document
)

func (h *handler) HandleTextMessage(message whatsapp.TextMessage) {
	if message.Info.Status != whatsapp.Read {
		h.wac.Read(message.Info.RemoteJid, message.Info.Id)

		if h.startingTime != 0 && message.Info.Timestamp > h.startingTime && message.Text[0] == '!' {
			message.Info.Id = ""
			message.Info.FromMe = true
			message.Info.Timestamp = 0
			h.wac.Send(message)
		}
	}
}

func (h *handler) HandleError(err error) {
	if strings.Contains(err.Error(), whatsapp.ErrInvalidWsData.Error()) {
		return
	}
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	session, err := h.wac.Disconnect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error disconnecting: %v\n", err)
	}
	if len(session.ClientToken) >= 10 {
		err = writeSession(session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing Session: %v\n", err)
		}
	}
	fmt.Println("Waiting for reconnect")
	<-time.After(30 * time.Second)
	fmt.Println("Reconnecting")
	err = h.wac.Restore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error restoring session: %v\n", err)
	}
}

func login(wac *whatsapp.Conn) error {
	//load saved session
	session, err := readSession()
	if err == nil {
		//restore session
		session, err = wac.RestoreWithSession(session)
		if err != nil {
			return fmt.Errorf("restoring failed: %v\n", err)
		}
	} else {
		//no saved session -> regular login
		qr := make(chan string)
		go func() {
			terminal := qrcodeTerminal.New()
			terminal.Get(<-qr).Print()
		}()
		session, err = wac.Login(qr)
		if err != nil {
			return fmt.Errorf("error during login: %v\n", err)
		}
	}

	//save session
	err = writeSession(session)
	if err != nil {
		return fmt.Errorf("error saving session: %v\n", err)
	}
	return nil
}

func readSession() (whatsapp.Session, error) {
	session := whatsapp.Session{}
	file, err := os.Open("whatsappSession.gob")
	if err != nil {
		return session, err
	}
	defer file.Close()
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&session)
	if err != nil {
		return session, err
	}
	return session, nil
}

func writeSession(session whatsapp.Session) error {
	file, err := os.Create("whatsappSession.gob")
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(session)
	if err != nil {
		return err
	}
	return nil
}
