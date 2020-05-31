package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/tlarsen7572/Golang-Public/ryx/config"
	"github.com/tlarsen7572/Golang-Public/ryx/tool_data_loader"
	cop "github.com/tlarsen7572/Golang-Public/ryx/traffic_cop"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	println(`loading configuration...`)
	conf, err := config.LoadConfig()
	if err != nil {
		println(err.Error())
		return
	}
	println(`creating log...`)
	log, err := os.Create(conf.LogPath)
	if err != nil {
		println(err.Error())
		return
	}
	println(`loading tool data...`)
	toolData, err := tool_data_loader.LoadAll(conf.InstallPath, conf.ProgramDataPath)
	if err != nil {
		println(err.Error())
		return
	}
	conf.ToolData = toolData
	println(`configuring webserver...`)
	in := make(chan cop.FunctionCall)
	go cop.StartTrafficCop(in)

	http.HandleFunc(`/`, generateServe(in, conf))
	http.HandleFunc("/main.dart.js", handleFile)
	http.HandleFunc("/main.dart.js.map", handleFile)
	http.HandleFunc("/main.dart.js.deps", handleFile)
	http.HandleFunc("/assets/", handleFile)
	println(`listening on ` + conf.Address)
	err = http.ListenAndServe(conf.Address, nil)
	if err != nil {
		writeLog(log, err.Error())
	}
	_ = log.Close()
}

type RequestPayload struct {
	Function   string
	Project    string
	Parameters map[string]interface{}
}

type ResponsePayload struct {
	Success bool
	Data    interface{}
}

func generateServe(in chan cop.FunctionCall, conf *config.Config) func(writer http.ResponseWriter, r *http.Request) {
	return func(writer http.ResponseWriter, r *http.Request) {
		if r.Method == `GET` {
			http.ServeFile(writer, r, filepath.Join(`html`, `index.html`))
			return
		}

		decoder := json.NewDecoder(r.Body)
		request := &RequestPayload{}
		err := decoder.Decode(request)
		if err != nil {
			sendErrorResponse(writer, err.Error())
			return
		}

		out := make(chan cop.FunctionResponse)
		funcCall := cop.FunctionCall{
			Project:    request.Project,
			Function:   request.Function,
			Parameters: request.Parameters,
			Out:        out,
			Config:     conf,
		}
		in <- funcCall

		response := <-out
		close(out)
		if response.Err != nil {
			sendErrorResponse(writer, response.Err.Error())
			return
		}
		sendNormalResponse(writer, response.Response)
	}
}

func handleFile(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Path
	ext := filepath.Ext(file)
	if ext == `.js` || ext == `.json` || ext == `.map` {
		setHeaders(w, "application/javascript")
	}
	if ext == `.deps` {
		setHeaders(w, "text/plain")
	}
	http.ServeFile(w, r, filepath.Join(`html`, file))
}

func writeLog(log *os.File, msg string) {
	timestamp := time.Now().Format(time.RFC3339)
	entry := fmt.Sprintf(`%v - %v`, timestamp, msg)
	_, _ = log.WriteString(entry)
}

func sendNormalResponse(w http.ResponseWriter, data interface{}) {
	setHeaders(w, "application/json")
	response := ResponsePayload{true, data}
	responseBytes, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		buffer := bytes.NewBufferString(marshalErr.Error())
		_, _ = w.Write(buffer.Bytes())
		return
	}
	_, _ = w.Write(responseBytes)
}

func sendErrorResponse(w http.ResponseWriter, err string) {
	setHeaders(w, "application/json")
	response := ResponsePayload{false, err}
	responseBytes, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		var buffer = bytes.NewBufferString(marshalErr.Error())
		_, _ = w.Write(buffer.Bytes())
		return
	}
	_, _ = w.Write(responseBytes)
}

func setHeaders(w http.ResponseWriter, contentType string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", contentType)
}
