package main

import (
	"database/sql"
	"github.com/Rhymen/go-whatsapp"
	_ "github.com/go-sql-driver/mysql"
	"time"
	"fmt"
	"os"
	"github.com/Baozisoftware/qrcode-terminal-go"
	"encoding/gob"
)

func main() {
	x := &hanlder{}
	var err error
	x.db, err = sql.Open("mysql", "root@/whatsapp")
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

func (h *hanlder) HandleTextMessage(message whatsapp.TextMessage) {
	_, err := h.db.Exec("CALL whatsapp.insert_text((?),(?),(?),from_unixtime((?)),(?))",
		message.Info.Id,
		message.Info.RemoteJid,
		message.Info.FromMe,
		message.Info.Timestamp,
		message.Text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error inserting: %T\n", err)
		fmt.Fprintf(os.Stderr, "error inserting: %v\n", err)
	}
}

func (*hanlder) HandleError(err error) {
	panic("implement me")
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
