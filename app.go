package main

import (
	"io"
	"log"
	"os"
	"time"
)

const dateLayout = time.RFC3339Nano
const logPattern = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile | log.LUTC

var infoLogger *log.Logger
var warnLogger *log.Logger
var errorLogger *log.Logger

func init() {
	initLogs(os.Stdout, os.Stdout, os.Stderr)
}

func main() {
	app := cli.App("brightcove-notifier", "Gets notified about Brightcove FT video events, creates UPP publish event and posts it to CMS Notifier.")
	cmsNotifier := app.String(cli.StringOpt{
		Name:   "cms-notifier",
		Value:  "localhost:13080",
		Desc:   "CMS notifier's address",
		EnvVar: "CMS_NOTIFIER",
	})

}

func initLogs(infoHandle io.Writer, warnHandle io.Writer, errorHandle io.Writer) {
	infoLogger = log.New(infoHandle, "INFO  - ", logPattern)
	warnLogger = log.New(warnHandle, "WARN  - ", logPattern)
	errorLogger = log.New(errorHandle, "ERROR - ", logPattern)
}
