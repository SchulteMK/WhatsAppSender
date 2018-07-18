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

	r := mux.NewRouter()
	r.HandleFunc("/media/{messageID}/meta", x.GetMediaMeta).Methods("GET")
	r.HandleFunc("/media/{messageID}/data", x.GetMediaData).Methods("GET")

	http.ListenAndServe(":8080", r)
	select {}
}

type handler struct {
	db *sql.DB
}

type Media struct {
	Id        int
	Caption   string
	Mimetype  string
	Thumbnail []byte
}

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
	var m Media
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

func (h *handler) HandleImageMessage(message whatsapp.ImageMessage) {
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
}

func (*handler) HandleError(err error) {
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
