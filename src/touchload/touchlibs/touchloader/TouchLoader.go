package touchloader

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"os"
	"time"
	"sync/atomic"
	"strings"
	"encoding/json"
	"bytes"
	"compress/gzip"
	"math/rand"
)

var Counter2xx uint64 = 0
var Counter4xx uint64 = 0
var Counter5xx uint64 = 0

/* 	Get the 2xx counter
	*/
func GetCounter2xx() (uint64) {
	return atomic.LoadUint64(&Counter2xx)
}


/* 	Get the Non-2xx counter
	*/
func GetCounter4xx() (uint64) {
	return atomic.LoadUint64(&Counter4xx)
}

func GetCounter5xx() (uint64) {
	return atomic.LoadUint64(&Counter5xx)
}

/*	# Does the following actions
	- Reset event counts (application when runmode=test)
	- Post events
	*/



func ResetPoster(referenceBridge chan int, maTrackingUrl string, runMode string,
			payload interface{}, compression string, userIntensive bool,
			userIntenseTimeoutMax int, userIntenseTimeoutMin int) {

	if (strings.ToLower(runMode) == "test") {
		// Reset event counter
		resetUrl := "http://localhost:8080/springwebeg/track/reset"
		ExecGetUri(resetUrl)
	}

	// marshalling the json payload string
	marshalledJson, _ := json.Marshal(payload)
	rawJsonPayload := string(marshalledJson)

	var err error
	var buf bytes.Buffer

	if (compression == "gzip") {
		// if request is for gzip based compression payload
		writerObj := gzip.NewWriter(&buf)

		_, err = writerObj.Write([]byte(rawJsonPayload))
		if err != nil {
			fmt.Println(err)
			return
		}

		if err = writerObj.Close(); err != nil {
			fmt.Println(err)
			return
		}
	}

	<-referenceBridge

	for {
		// This makes sure that we have user intensive hits compared to events intensive
		if userIntensive == true {
			randomNum := rand.Intn(int(userIntenseTimeoutMax - userIntenseTimeoutMin)) + userIntenseTimeoutMin
			time.Sleep(time.Second * time.Duration(randomNum) )
		}

		Poster(maTrackingUrl, rawJsonPayload, compression, buf)

		if (strings.ToLower(runMode) == "test") {
			time.Sleep(time.Second * 2)        // send requests at a interval of 2 secs, if test mode
		}
	}
}


/*	Poster method to start hitting post requests
 	*/
func Poster(maTrackingUrl string, rawJsonPayload string, compression string, buf bytes.Buffer) {

	var err error
	var req *http.Request

	if (compression == "gzip") {
		// if request is for gzip based compression payload
		// send out request
		req, err = http.NewRequest("POST", maTrackingUrl, &buf)
		req.Header.Set("Content-Encoding", "gzip")

	} else {
		// send out plain request
		var jsonStr = []byte(string(rawJsonPayload))
		req, err = http.NewRequest("POST", maTrackingUrl, bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)        // thread safe client

	if err != nil {
		//fmt.Println("Status:= ",err)
		atomic.AddUint64(&Counter4xx, 1)
		return
	}

	statusCode := resp.StatusCode

	switch {
	case (statusCode == 200) :
		atomic.AddUint64(&Counter2xx, 1)

	case (statusCode >= 400 && statusCode < 500) :
		atomic.AddUint64(&Counter4xx, 1)

	case (statusCode >= 500) :
		atomic.AddUint64(&Counter5xx, 1)

	default:
		fmt.Println("# None of the statusCode found.")
	}

	defer resp.Body.Close()
}


/* 	Hitting GET requests
	*/
func ExecGetUri(URL string) (string) {
	resp, err := http.Get(URL)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	htmlData, err := ioutil.ReadAll(resp.Body) //<--- here!

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return string(htmlData)
}
