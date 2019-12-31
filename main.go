package main

import (
	"database/sql"
	"encoding/gob"
	"fmt"
	"github.com/Baozisoftware/qrcode-terminal-go"
	"github.com/Rhymen/go-whatsapp"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	x := &handler{}
	var err error

	//create new WhatsApp connection
	x.wac, err = whatsapp.NewConn(5 * time.Second)
	if err != nil {
		log.Fatalf("error creating connection: %v\n", err)
	}

	x.startingTime = uint64(time.Now().Unix())

	//Add handler
	x.wac.AddHandler(x)

	if err := login(x.wac); err != nil {
		log.Fatalf("error logging in: %v\n", err)
	}

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

		if message.Info.Timestamp > h.startingTime && message.Text[0] == '!' {
			message.Info.Id = ""
			message.Info.FromMe = true
			message.Info.Timestamp = 0
			h.wac.Send(message)
		}
	}
}

func (h *handler) HandleError(err error) {
	if e, ok := err.(*whatsapp.ErrConnectionFailed); ok {
		log.Printf("Connection failed, underlying error: %v", e.Err)
		log.Println("Waiting 30sec...")
		<-time.After(30 * time.Second)
		log.Println("Reconnecting...")
		if err := h.wac.Restore(); err != nil {
			log.Fatalf("Restore failed: %v", err)
		}
	} else {
		log.Printf("error occoured: %v\n", err)
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
