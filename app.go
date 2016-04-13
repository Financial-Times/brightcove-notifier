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

const logPattern = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile | log.LUTC

var infoLogger *log.Logger
var warnLogger *log.Logger
var errorLogger *log.Logger

func init() {
	initLogs(os.Stdout, os.Stdout, os.Stderr)
}

type brightcoveNotifier struct {
	port            int
	brightcoveConf  *brightcoveConfig
	cmsNotifierConf *cmsNotifierConfig
	client          *http.Client
}

type brightcoveConfig struct {
	addr        string
	accessToken string

	//Brightcove OAuth API access token endpoint
	oauthAddr string
	auth      string
}

type cmsNotifierConfig struct {
	addr string
	auth string
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
		Desc:   "brightcove video api address",
		EnvVar: "BRIGHTCOVE",
	})
	brightcoveOAuth := app.String(cli.StringOpt{
		Name:   "brightcove-oauth",
		Value:  "https://oauth.brightcove.com/v3/access_token",
		Desc:   "brightcove oauth api address",
		EnvVar: "BRIGHTCOVE_OAUTH",
	})
	brightcoveAuth := app.String(cli.StringOpt{
		Name:   "brightcove-auth",
		Value:  "",
		Desc:   "brightcove OAUTH API authorization header",
		EnvVar: "BRIGHTCOVE_AUTH",
	})
	cmsNotifier := app.String(cli.StringOpt{
		Name:   "cms-notifier",
		Value:  "http://localhost:13080/notify",
		Desc:   "cms notifier address",
		EnvVar: "CMS_NOTIFIER",
	})
	cmsNotifierAuth := app.String(cli.StringOpt{
		Name:   "cms-notifier-auth",
		Value:  "",
		Desc:   "cms notifier authorization header",
		EnvVar: "CMS_NOTIFIER_AUTH",
	})

	bn := &brightcoveNotifier{
		port: *port,
		brightcoveConf: &brightcoveConfig{
			addr:      *brightcove,
			oauthAddr: *brightcoveOAuth,
			auth:      *brightcoveAuth,
		},
		cmsNotifierConf: &cmsNotifierConfig{
			addr: *cmsNotifier,
			auth: *cmsNotifierAuth,
		},
		client: &http.Client{},
	}

	app.Action = func() {
		go bn.listen()
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		infoLogger.Println("Received termination signal. Quitting... \nBye")
	}
	err := app.Run(os.Args)
	if err != nil {
		errorLogger.Printf("[%v]", err)
	}
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
	req, err := http.NewRequest("GET", fmt.Sprintf(bn.brightcoveConf.addr, ve.AccountID, ve.Video), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", "Bearer "+bn.brightcoveConf.accessToken)
	resp, err := bn.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer cleanupResp(resp)
	switch resp.StatusCode {
	case 401:
		err = bn.renewAccessToken()
		if err != nil {
			errorLogger.Printf("Video publishing won't work. Renewing access token failure: [%v].", err)
			return nil, err
		}
		return bn.fetchVideo(ve)
	case 404:
		fallthrough
	case 200:
		video, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		infoLogger.Printf("Fetching video successful. Size: [%d]", len(video))
		return video, nil
	default:
		return nil, fmt.Errorf("Invalid statusCode received: [%d]", resp.StatusCode)
	}

}

func (bn brightcoveNotifier) fwdVideo(video []byte) error {
	req, err := http.NewRequest("POST", bn.cmsNotifierConf.addr, bytes.NewReader(video))
	if err != nil {
		return err
	}
	req.Header.Add("X-Origin-System-Id", "brightcove")
	req.Header.Add("Authorization", bn.cmsNotifierConf.auth)
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
		infoLogger.Println("Forwarding video successful.")
		return nil
	default:
		return fmt.Errorf("Invalid statusCode received: [%d]", resp.StatusCode)
	}
}

const tokenRequest = "grant_type=client_credentials"

type accessTokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Expires     int    `json:"expires_in"`
}

func (bn brightcoveNotifier) renewAccessToken() error {
	req, err := http.NewRequest("POST", bn.brightcoveConf.oauthAddr, bytes.NewReader([]byte(tokenRequest)))
	if err != nil {
		return err
	}
	req.Header.Add("Content-type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", bn.brightcoveConf.auth)
	resp, err := bn.client.Do(req)
	if err != nil {
		return err
	}
	defer cleanupResp(resp)
	if resp.StatusCode != 200 {
		return fmt.Errorf("Invalid statusCode received: [%d]", resp.StatusCode)
	}
	var accTokenResp accessTokenResp
	err = json.NewDecoder(resp.Body).Decode(&accTokenResp)
	if err != nil {
		return err
	}
	if accTokenResp.AccessToken == "" {
		return fmt.Errorf("Empty access token: [%#v]", accTokenResp)
	}
	bn.brightcoveConf.accessToken = accTokenResp.AccessToken
	return nil
}

func cleanupResp(resp *http.Response) {
	_, err := io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		warnLogger.Printf("[%v]", err)
	}
	err = resp.Body.Close()
	if err != nil {
		warnLogger.Printf("[%v]", err)
	}
}

func (bn brightcoveNotifier) health(w http.ResponseWriter, r *http.Request) {}

func (bn brightcoveNotifier) gtg(w http.ResponseWriter, r *http.Request) {}

func initLogs(infoHandle io.Writer, warnHandle io.Writer, errorHandle io.Writer) {
	infoLogger = log.New(infoHandle, "INFO  - ", logPattern)
	warnLogger = log.New(warnHandle, "WARN  - ", logPattern)
	errorLogger = log.New(errorHandle, "ERROR - ", logPattern)
}
