package main

import (
	"fmt"
	"github.com/creamdog/gonfig"
	"log"
	"flag"
	"touchload/touchlibs/configs"
	"touchload/touchlibs/touchloader"
	"os"
	"path/filepath"
	"strings"
	"github.com/satori/go.uuid"
	"time"

	//"strconv"
	"syscall"
)

/** Go build for linux os : GOOS=linux go build
	#for i in `seq 1 1`; do nohup ./hallmark -users=8 -config=/home/ashutoshp/touchload/configs/TouchloadConfigs.gonfig -payload=low -runmode=prod-test & done
	*/

type file gonfig.Gonfig

var maTrackingUrl string

/* CLI args */
var concUsers int                // Defines total number of concurrent users for run instance
var logFile string                // Defines where the logging has to happen
var configFile string                // Defines the full path of the config file named TouchloadConfigs.gonfig
var payloadType string                // Defines what kind of payload has to be sent. Default has a batch 5 events and low is just 1 event
var runMode string                // Defines in what run environment the tool has to run. Eg values: test | prod | prod-test
var compression string        // Defines if the payload has to be compressed payload or not
var tickerTimeout int                // Defines the frequency by which the logging of all the counters happen into log file
var userIntensive bool                // Defines the frequency by which the logging of all the counters happen into log file
var userIntensiveTimeoutMax interface{}
var userIntensiveTimeoutMin interface{}
var noconsoleout bool                // To control the print statements on console. Intends to reduce the size of nohup.out
var notificationsPayload bool        // If yes, then will pick only notification payload events as load


func init() {
	var homePath string
	homePath += os.Getenv("HOME")
	logPathDefault := homePath + "/touchload.log"
	payloadDefault := "default"
	runModeDefault := "empty"
	compressionDefault := "none"
	waitDefault := 5
	configDefault, _ := filepath.Abs("./configs/TouchloadConfigs.gonfig")
	userIntensiveDefault := false
	notificationDefault := false

	flag.IntVar(&concUsers, "users", 1, "-users=1 (int type, total concurrent users)")
	flag.StringVar(&logFile, "log", logPathDefault, "-log=<log/file/path>")
	flag.StringVar(&configFile, "config", configDefault, "-config=<config/file/path>")
	flag.StringVar(&payloadType, "payload", payloadDefault, "-payload=(default | low)")
	flag.StringVar(&runMode, "runmode", runModeDefault, "-runmode=(test | prod | prod-test)")
	flag.StringVar(&compression, "compression", compressionDefault, "-compression=(gzip | none - default)")
	flag.IntVar(&tickerTimeout, "timeout", waitDefault, "-timeout=5 (int in minutes, 5 min's by default)")
	flag.BoolVar(&userIntensive, "userintensive", userIntensiveDefault, "-userintensive=true (Boolean value, false by default)")
	flag.BoolVar(&noconsoleout, "noconsoleout", false, "-noconsoleout=true (Boolean value, false by default)")
	flag.BoolVar(&notificationsPayload, "notifications", notificationDefault, "-notifications=true|false (Boolean value, false by default)")

	flag.Parse()
}

/** ==========================================================================================
	Check the completeness of CLI args
 */
func checkArgs() {
	if (strings.ToLower(runMode) == "empty") {
		fmt.Println("")
		fmt.Println("# WARNING: \"runmode\" arg is missing.")
		fmt.Println("")
		flag.Usage()
		os.Exit(1)
	}
}

/** ==========================================================================================
	golang - MaTouchload launcher script
*/
func main() {
	// check if "runmode" arg is missing since its mandatory
	checkArgs()

	file := configs.GetFileHandler(configFile)

	// ToDo Get : low, high & batch payload
	/** Determine runs based on below params
	 	testmode = true | false (this will hit local test instance not prod)
		Payload = default | low (this will differ in the payload size sent)
		*/
	var urlJsonPath string
	var payloadJsonPath string
	urlJsonPath = "url/" + strings.ToLower(runMode)

	// Looking for low only as the default value for payload = "default"
	payloadJsonPath = "payload/" + strings.ToLower(payloadType)

	// get endpoint url
	url, _ := file.Get(urlJsonPath, "empty-url")
	maTrackingUrl = url.(string)

	// get payload
	load, _ := file.Get(payloadJsonPath, "empty-json")

	max, _ := file.Get("userintensive/timeout/max", "900")
	min, _ := file.Get("userintensive/timeout/min", "120")
	userIntensiveTimeoutMax = int(max.(float64))
	userIntensiveTimeoutMin = int(min.(float64))

	fmt.Print("\n")
	log.Print(":= (START TIME) =:")
	fmt.Print(":= Load Start :=",
		"\nUsers/parallel runs -> ", concUsers,
		"\nLog file: ", logFile,
		"\nPayload: ", payloadType,
		"\nConfig File: ", configFile,
		"\nProd URL: ", maTrackingUrl,
		"\nPayload Json Path: ", payloadJsonPath,
		"\nPayload Type: ", payloadType,
		"\nIs Compressed: ", compression,
		"\nPrint Ticker timeout: ", tickerTimeout,
		"\nUser Intensive Mode: ", userIntensive,
		"\nUser Intensive Timeout Max: ", userIntensiveTimeoutMax,
		"\nUser Intensive Timeout Min: ", userIntensiveTimeoutMin,
		"\nRun Mode: ", runMode, "\n")

	// concurrently hitting the url
	go sendLoad(maTrackingUrl, load)

	select {}
}

func sendLoad(maTrackingUrl string, payload interface{}) {
	// open a file
	logFH, err := os.OpenFile(logFile, os.O_APPEND | os.O_CREATE | os.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	defer logFH.Close()
	log.SetOutput(logFH)        // assign it to the standard logger

	fmt.Println("===========================================================")
	fmt.Println("# Runtime stats:= ")

	log.Println("===========================================================")
	log.Println("Run time stats at log will be printing on a frequency of every", tickerTimeout, "- Minute/s.")
	log.Println("# Runtime stats:= ")

	bridge := make(chan int, 1) //creating the channel


	for i := 1; i <= concUsers; i++ {
		bridge <- i //sending integer through the bridge(channel), this is blocked till its recieved in the goroutine

		uniqueUDID := uuid.NewV4().String() + uuid.NewV5(uuid.NewV4(), uuid.NewV1().String()).String()

		log.Println("uniqueUDID # ", i, uniqueUDID)
		fmt.Println("uniqueUDID # ", i, uniqueUDID)

		if payloadType == "low" {
			payload.(map[string]interface{})["device"].(map[string]interface{})["device_id"] = uniqueUDID
			payload.(map[string]interface{})["session"].(map[string]interface{})["session_id"] = uniqueUDID

		} else if payloadType == "default" {
			val := payload.([]interface{})

			for i := range val {
				//fmt.Println(val[i].(map[string]interface {}))
				val[i].(map[string]interface{})["device"].(map[string]interface{})["device_id"] = uniqueUDID
				val[i].(map[string]interface{})["session"].(map[string]interface{})["session_id"] = uniqueUDID
			}
		} else if payloadType == "notifications" {
			fmt.Printf("==========================")
			fmt.Println(payload.(map[string]interface{})["pushNotificationDismissed"])
			fmt.Printf("==========================")
			syscall.Exit(0)
		}


		// Start the sort in a goroutine; when it completes, signal on the channel.
		go touchloader.ResetPoster(bridge,
			maTrackingUrl,
			runMode,
			payload,
			compression,
			userIntensive,
			userIntensiveTimeoutMax.(int),
			userIntensiveTimeoutMin.(int))
	}

	type Counters2xx int
	type Counters4xx int
	type Counters5xx int
	var resp string = "-"

	timeout := tickerTimeout
	ticker := time.NewTicker(time.Minute * time.Duration(timeout))

	for {
		Counters2xx := touchloader.GetCounter2xx()
		Counters4xx := touchloader.GetCounter4xx()
		Counters5xx := touchloader.GetCounter5xx()

		if (strings.ToLower(runMode) == "test") {
			uri := "http://localhost:8080/springwebeg/track/infopings"
			resp = touchloader.ExecGetUri(uri)
		}

		if noconsoleout != true {
			fmt.Printf("\rCounters:= (2xx/4xx/5xx/TestSrvResp) := %d/%d/%d/%s", Counters2xx, Counters4xx, Counters5xx, resp)
		}

		select {
		case <-ticker.C:
			log.Println("Counters:= (2xx/4xx/5xx/TestSrvResp) : ", Counters2xx, "/", Counters4xx, "/", Counters5xx, "/", resp)
		default:
		}
	}

	select {}

}