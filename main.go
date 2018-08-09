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
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"encoding/json"
	"io/ioutil"
	"bytes"
)

func main() {
	x := &handler{}
	var err error

	//SQL Connection
	x.db, err = sql.Open("mysql", "root@(Marcel-PC-1:3306)/whatsapp")
	if err != nil {
		panic(err)
	}

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

	r := mux.NewRouter()
	r.HandleFunc("/media/{messageID}/meta", x.GetMediaMeta).Methods("GET")
	r.HandleFunc("/media/{messageID}/data", x.GetMediaData).Methods("GET")
	r.HandleFunc("/addHandler/", x.RegisterHandler).Methods("POST")

	//http.ListenAndServe(":8080", r)

	<-time.After(3 * time.Second)

	x.startingTime = uint64(time.Now().Unix())
	select {}
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

func (h *handler) GetMediaMeta(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	ids, ok := vars["messageID"]
	if !ok {
		res.Write([]byte("no valid id provided."))
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(ids)
	if err != nil {
		res.Write([]byte("no valid id provided."))
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	r, err := h.db.Query("SELECT id,caption,mimetype,thumbnail FROM media WHERE id = (?)", id)
	defer r.Close()
	if err != nil {
		res.Write([]byte(fmt.Sprint(err)))
	}
	if !r.Next() {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	var m MediaMetaData
	err = r.Scan(&m.Id, &m.Caption, &m.Mimetype, &m.Thumbnail)
	if err != nil {
		res.Write([]byte(fmt.Sprint(err)))
	}
	data, err := json.Marshal(m)
	if err != nil {
		res.Write([]byte(fmt.Sprint(err)))
	}
	res.Write(data)
	res.WriteHeader(http.StatusOK)
	fmt.Printf("%v\n%v\n", req, id)
}

func (h *handler) GetMediaData(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	ids, ok := vars["messageID"]
	if !ok {
		res.Write([]byte("no valid id provided."))
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(ids)
	if err != nil {
		res.Write([]byte("no valid id provided."))
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	r, err := h.db.Query("SELECT data FROM media WHERE id = (?)", id)
	defer r.Close()
	if err != nil {
		res.Write([]byte(fmt.Sprint(err)))
	}
	if !r.Next() {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	var data []byte
	err = r.Scan(&data)
	if err != nil {
		res.Write([]byte(fmt.Sprint(err)))
	}

	res.Write(data)
	res.WriteHeader(http.StatusOK)
	fmt.Printf("%v\n%v\n", req, id)
}

func (h *handler) RegisterHandler(res http.ResponseWriter, req *http.Request) {
	body := req.Body
	defer body.Close()
	data, err := ioutil.ReadAll(body)
	if err != nil {
		res.Write([]byte(err.Error()))
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	var mHandler MessageHandler
	err = json.Unmarshal(data, &mHandler)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	h.handlers = append(h.handlers, mHandler)
	res.WriteHeader(http.StatusOK)
}

func (h *handler) HandleImageMessage(message whatsapp.ImageMessage) {
	fmt.Printf("Image: %v\n", message)
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
	h.NotifyHandlers(message.Info.Id, Image)
}

func (h *handler) HandleVideoMessage(message whatsapp.VideoMessage) {
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
	h.NotifyHandlers(message.Info.Id, Video)
}

func (h *handler) HandleAudioMessage(message whatsapp.AudioMessage) {
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
	h.NotifyHandlers(message.Info.Id, Audio)
}

func (h *handler) HandleDocumentMessage(message whatsapp.DocumentMessage) {
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
	h.NotifyHandlers(message.Info.Id, Document)
}

func (h *handler) HandleTextMessage(message whatsapp.TextMessage) {
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
	h.NotifyHandlers(message.Info.Id, Text)
	if h.startingTime != 0 &&
		message.Info.Timestamp > h.startingTime &&
		message.Text[0] == '!' {
		message.Info.Id = ""
		message.Info.FromMe = true
		message.Info.Timestamp = 0
		h.wac.Send(message)
	}
}

func (h *handler) NotifyHandlers(id string, t MessageType) {
	msg := getMessageInfoFromDB(id)
	switch t {
	case Text:
		for _, v := range h.handlers {
			if v.Text {
				v.Notify(msg)
			}
		}
	case Image:
		for _, v := range h.handlers {
			if v.Image {
				v.Notify(msg)
			}
		}
	case Video:
		for _, v := range h.handlers {
			if v.Video {
				v.Notify(msg)
			}
		}
	case Audio:
		for _, v := range h.handlers {
			if v.Audio {
				v.Notify(msg)
			}
		}
	case Document:
		for _, v := range h.handlers {
			if v.Document {
				v.Notify(msg)
			}
		}
	}
}

func getMessageInfoFromDB(s string) []byte {
	return []byte{2, 4, 5, 6}
}

func (m *MessageHandler) Notify(msg []byte) {
	req, err := http.NewRequest("POST", m.Url, bytes.NewBuffer(msg))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		//cannot connect
		return
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
}

func (h *handler) HandleError(err error) {
	panic("implement me")
}

func (h *handler) alreadyExists(id string) bool {
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

func (h *handler) insertMedia(id, remotejid string, fromme bool, timestamp uint64, caption string, thumbnail []byte, mime string, data []byte) {
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
