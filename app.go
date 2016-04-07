package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	cmsNotifier string
}

func main() {
	app := cli.App("brightcove-notifier", "Gets notified about Brightcove FT video events, creates UPP publish event and posts it to CMS Notifier.")
	port := app.Int(cli.IntOpt{
		Name:   "port",
		Value:  8080,
		Desc:   "application port",
		EnvVar: "PORT",
	})
	cmsNotifier := app.String(cli.StringOpt{
		Name:   "cms-notifier",
		Value:  "localhost:13080",
		Desc:   "cms notifier's address",
		EnvVar: "CMS_NOTIFIER",
	})

	bn := &brightcoveNotifier{*port, *cmsNotifier}

	app.Action = func() {
		go bn.listen()
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
	}
	app.Run(os.Args)
}

func (bn brightcoveNotifier) listen() {
	r := mux.NewRouter()
	r.HandleFunc("/", bn.handleNotification).Methods("POST")
	r.HandleFunc("/__health", bn.health).Methods("GET")
	r.HandleFunc("/__gtg", bn.gtg).Methods("GET")

	http.Handle("/", r)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		errorLogger.Panicf("Couldn't set up HTTP listener: %+v\n", err)
	}
}

func (bn brightcoveNotifier) handleNotification(w http.ResponseWriter, r *http.Request) {}
func (bn brightcoveNotifier) health(w http.ResponseWriter, r *http.Request)             {}
func (bn brightcoveNotifier) gtg(w http.ResponseWriter, r *http.Request)                {}

func initLogs(infoHandle io.Writer, warnHandle io.Writer, errorHandle io.Writer) {
	infoLogger = log.New(infoHandle, "INFO  - ", logPattern)
	warnLogger = log.New(warnHandle, "WARN  - ", logPattern)
	errorLogger = log.New(errorHandle, "ERROR - ", logPattern)
}
