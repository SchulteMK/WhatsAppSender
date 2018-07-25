package main

import (
	"database/sql"
	"encoding/gob"
	"fmt"
	"github.com/Baozisoftware/qrcode-terminal-go"
	"github.com/Rhymen/go-whatsapp"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"time"
	"net/url"
)

func main() {
	x := &hanlder{}
	var err error
	x.db, err = sql.Open("mysql", "root@(Marcel-PC-1:3306)/whatsapp")
	if err != nil {
		panic(err)
	}

	//create new WhatsApp connection
	wac, err := whatsapp.NewConn(5 * time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating connection: %v\n", err)
		return
	}

	//Add handler
	wac.AddHandler(x)

	err = login(wac)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error logging in: %v\n", err)
		return
	}

	select {}
}

type hanlder struct {
	db *sql.DB
}

func (h *hanlder) HandleImageMessage(message whatsapp.ImageMessage) {
	if h.alreadyExists(message.Info.Id) {
		return
	}
	data, err := message.Download()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error downloading image: %v\n", err)
		return
	}
	h.insertMedia(message.Info.Id,
		message.Info.RemoteJid,
		message.Info.FromMe,
		message.Info.Timestamp,
		message.Caption,
		message.Thumbnail,
		message.Type,
		data)
}

func (h *hanlder) HandleVideoMessage(message whatsapp.VideoMessage) {
	if h.alreadyExists(message.Info.Id) {
		return
	}
	data, err := message.Download()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error downloading image: %v\n", err)
		return
	}
	h.insertMedia(message.Info.Id,
		message.Info.RemoteJid,
		message.Info.FromMe,
		message.Info.Timestamp,
		message.Caption,
		message.Thumbnail,
		message.Type,
		data)
}

func (h *hanlder) HandleAudioMessage(message whatsapp.AudioMessage) {
	if h.alreadyExists(message.Info.Id) {
		return
	}
	data, err := message.Download()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error downloading image: %v\n", err)
		return
	}
	h.insertMedia(message.Info.Id,
		message.Info.RemoteJid,
		message.Info.FromMe,
		message.Info.Timestamp,
		"",
		nil,
		message.Type,
		data)
}

func (h *hanlder) HandleDocumentMessage(message whatsapp.DocumentMessage) {
	if h.alreadyExists(message.Info.Id) {
		return
	}
	data, err := message.Download()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error downloading image: %v\n", err)
		return
	}
	h.insertMedia(message.Info.Id,
		message.Info.RemoteJid,
		message.Info.FromMe,
		message.Info.Timestamp,
		message.Title,
		message.Thumbnail,
		message.Type,
		data)
}

func (h *hanlder) HandleTextMessage(message whatsapp.TextMessage) {
	_, err := h.db.Exec("CALL whatsapp.insert_text((?),(?),(?),from_unixtime((?)),(?))",
		message.Info.Id,
		message.Info.RemoteJid,
		message.Info.FromMe,
		message.Info.Timestamp,
		url.QueryEscape(message.Text))

	if err != nil {
		fmt.Fprintf(os.Stderr, "error inserting: %T\n", err)
		fmt.Fprintf(os.Stderr, "error inserting: %v\n", err)
	}
}

func (*hanlder) HandleError(err error) {
	panic("implement me")
}

func (h *hanlder) alreadyExists(id string) bool {
	var count int
	err := h.db.QueryRow("SELECT COUNT(*) FROM message_info WHERE id = (?)", id).Scan(&count)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sql error: %v\n", err)
		return false
	}

	if count > 0 {
		return true
	}
	return false
}
func (h *hanlder) insertMedia(id, remotejid string, fromme bool, timestamp uint64, caption string, thumbnail []byte, mime string, data []byte) {
	_, err := h.db.Exec("CALL whatsapp.insert_media((?),(?),(?),from_unixtime((?)),(?),(?),(?),(?))",
		id,
		remotejid,
		fromme,
		timestamp,
		caption,
		thumbnail,
		mime,
		data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error inserting: %v\n", err)
	}
	fmt.Printf("downloaded media: %v  %v\n", time.Unix(int64(timestamp), 0), remotejid)
}

func login(wac *whatsapp.Conn) error {
	//load saved session
	session, err := readSession()
	if err == nil {
		//restore session
		session, err = wac.RestoreSession(session)
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
	file, err := os.Open(os.TempDir() + "/whatsappSession.gob")
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
	file, err := os.Create(os.TempDir() + "/whatsappSession.gob")
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
