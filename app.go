package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
)

const dateLayout = time.RFC3339Nano
const logPattern = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile | log.LUTC

var infoLogger *log.Logger
var warnLogger *log.Logger
var errorLogger *log.Logger

func init() {
	initLogs(os.Stdout, os.Stdout, os.Stderr)
}

type brightcoveNotifier struct {
	port        int
	brightcove  string
	cmsNotifier string
	client      *http.Client
}

func main() {
	app := cli.App("brightcove-notifier", "Gets notified about Brightcove FT video events, creates UPP publish event and posts it to CMS Notifier.")
	port := app.Int(cli.IntOpt{
		Name:   "port",
		Value:  8080,
		Desc:   "application port",
		EnvVar: "PORT",
	})
	brightcove := app.String(cli.StringOpt{
		Name: "brightcove",
		// https://cms.api.brightcove.com/v1/accounts/:account_id/videos/:video_id
		Value:  "https://cms.api.brightcove.com/v1/accounts/%s/videos/%s",
		Desc:   "brightcove api's address",
		EnvVar: "BRIGHTCOVE",
	})
	cmsNotifier := app.String(cli.StringOpt{
		Name:   "cms-notifier",
		Value:  "http://localhost:13080/notify",
		Desc:   "cms notifier's address",
		EnvVar: "CMS_NOTIFIER",
	})

	bn := &brightcoveNotifier{*port, *brightcove, *cmsNotifier, &http.Client{}}

	app.Action = func() {
		go bn.listen()
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		infoLogger.Println("Received termination signal. Quitting... \nBye")
	}
	app.Run(os.Args)
}

func (bn brightcoveNotifier) listen() {
	r := mux.NewRouter()
	r.HandleFunc("/notify", bn.handleNotification).Methods("POST")
	r.HandleFunc("/__health", bn.health).Methods("GET")
	r.HandleFunc("/__gtg", bn.gtg).Methods("GET")

	http.Handle("/", r)
	infoLogger.Printf("Starting to listen on port [%d]", bn.port)
	err := http.ListenAndServe(":"+strconv.Itoa(bn.port), nil)
	if err != nil {
		errorLogger.Panicf("Couldn't set up HTTP listener: %+v\n", err)
	}
}

type videoEvent struct {
	TimeStamp int64  `json:"timeStamp"`
	AccountID string `json:"accountId"`
	Event     string `json:"event"`
	Video     string `json:"video"`
	Version   int    `json:"version"`
}

func (ve videoEvent) String() string {
	return fmt.Sprintf("videoEvent: TimeStamp: [%s], AccountId: [%s], Event: [%s], Video: [%s], Version: [%d]",
		time.Unix(0, ve.TimeStamp*int64(time.Millisecond)).Format(time.RFC3339), ve.AccountID, ve.Event, ve.Video, ve.Version)
}

func (bn brightcoveNotifier) handleNotification(w http.ResponseWriter, r *http.Request) {
	var event videoEvent

	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		warnLogger.Printf("[%v]", err)
	}

	infoLogger.Printf("Received: [%v]", event)

	video, err := bn.fetchVideo(event)
	if err == nil {
		warnLogger.Printf("Fetching video: [%v]", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = bn.fwdVideo(video)
	if err == nil {
		warnLogger.Printf("Forwarding video: [%v]", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (bn brightcoveNotifier) fetchVideo(ve videoEvent) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(bn.brightcove, ve.AccountID, ve.Video), nil)
	if err != nil {
		return nil, err
	}

	resp, err := bn.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer cleanupResp(resp)
	switch resp.StatusCode {
	case 401:
		//regenerate access token
		//then re-try
		return bn.fetchVideo(ve)
	case 404:
		fallthrough
	case 200:
		video, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return video, nil
	default:
		return nil, fmt.Errorf("Invalid statusCode received: [%d]", resp.StatusCode)
	}

}

func (bn brightcoveNotifier) fwdVideo(video []byte) error {
	req, err := http.NewRequest("POST", bn.cmsNotifier, bytes.NewReader(video))
	if err != nil {
		return err
	}

	resp, err := bn.client.Do(req)
	if err != nil {
		return err
	}
	defer cleanupResp(resp)
	switch resp.StatusCode {
	case 400:
		msg, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Status code 400. [%s]", string(msg[:]))
	case 200:
		return nil
	default:
		return fmt.Errorf("Invalid statusCode received: [%d]", resp.StatusCode)
	}
}

func cleanupResp(resp *http.Response) {
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
}

func (bn brightcoveNotifier) health(w http.ResponseWriter, r *http.Request) {}

func (bn brightcoveNotifier) gtg(w http.ResponseWriter, r *http.Request) {}

func initLogs(infoHandle io.Writer, warnHandle io.Writer, errorHandle io.Writer) {
	infoLogger = log.New(infoHandle, "INFO  - ", logPattern)
	warnLogger = log.New(warnHandle, "WARN  - ", logPattern)
	errorLogger = log.New(errorHandle, "ERROR - ", logPattern)
}
