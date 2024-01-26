/*===============================================================*/
/* This is an example of how to use the wcWebCamClient library.  */
/* In this example, a client is created, authorized on the       */
/* server, uploads a media record and downloads it to disk.      */
/*                                                               */
/* Part of WebCamClientLib go module                             */
/*                                                               */
/* Copyright 2024 Ilya Medvedkov                                 */
/*===============================================================*/

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"libs.com/wclib"
)

/* Relative path to save output data */
const OUTPUT_FOLDER = "output"

/* The total program timeout in seconds */
const TIME_OUT = 60

type appStatus int

const (
	StatusWaiting appStatus = iota
	StatusAuthorized
	StatusStreamDetected
	StatusIOStarted
	StatusIOFinished
	StatusError
)

type appStreamStruct struct {
	mux              sync.Mutex
	deltaTime        int
	curFrame         int
	ext              string
	status           appStatus
	mem_frame_buffer *bytes.Buffer
	sequence         chan *appStreamStruct
}

func (app *appStreamStruct) Lock() {
	app.mux.Lock()
}

func (app *appStreamStruct) Unlock() {
	app.mux.Unlock()
}

func (app *appStreamStruct) SetStatus(st appStatus) {
	app.Lock()
	defer app.Unlock()

	if st != app.status {
		app.status = st
		app.sequence <- app
	}
}

func (app *appStreamStruct) GetStatus() appStatus {
	app.Lock()
	defer app.Unlock()

	return app.status
}

var appStream = appStreamStruct{
	status:           StatusWaiting,
	mem_frame_buffer: bytes.NewBuffer(make([]byte, 0)),
	sequence:         make(chan *appStreamStruct, 16),
	deltaTime:        1000,
}

func AuthSuccess(tsk wclib.ITask) {
	fmt.Println("SID ", tsk.GetClient().GetSID())
	appStream.SetStatus(StatusAuthorized)
}

func OnLog(client *wclib.WCClient, str string) {
	fmt.Println(str)
}

func OnUpdateStreams(tsk wclib.ITask, jsonobj []map[string]any) {
	for _, jStrm := range jsonobj {
		var aStrm wclib.StreamStruct
		err := aStrm.JSONDecode(jStrm)
		check(err)

		if aStrm.Device == *listen_param {
			fmt.Printf("Stream detected `%s`; subproto: `%s`; delta: %d\n", aStrm.Device, aStrm.SubProto, int(appStream.deltaTime))

			s := strings.Split(aStrm.SubProto, "_")

			if len(s) > 1 {
				appStream.ext = s[len(s)-1]
			} else {
				appStream.ext = aStrm.SubProto
			}

			appStream.ext = strings.ToLower("." + appStream.ext)

			appStream.deltaTime = int(aStrm.Delta) / 4
			appStream.SetStatus(StatusStreamDetected)
			return
		}
	}
	fmt.Println("Streaming device is not online. Retry")
	appStream.sequence <- &appStream
}

func OnClientStateChange(c *wclib.WCClient, st wclib.ClientStatus) {
	switch st {
	case wclib.StateConnected:
		fmt.Printf("Client fully connected to server with sid %s\n", c.GetSID())
	case wclib.StateConnectedWrongSID:
		fmt.Printf("Client has no SID\n")
	case wclib.StateDisconnected:
		fmt.Printf("Client disconnected\n")
	}
}

/* Callback. IO stream closed. */
func onIOTaskFinished(tsk wclib.ITask) {
	fmt.Println("Output stream closed")
	appStream.SetStatus(StatusIOFinished)
}

/* Callback. IO stream started. */
func onIOTaskStarted(tsk wclib.ITask) {
	fmt.Println("Output stream started")
	appStream.SetStatus(StatusIOStarted)
}

func onNextFrame(tsk wclib.ITask) {
	fmt.Println("New frame captured")
	if appStream.GetStatus() != StatusIOStarted {
		appStream.SetStatus(StatusIOStarted)
	} else {
		appStream.sequence <- &appStream
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var proxy_param = flag.String("proxy", "", "Proxy in format [scheme:]//[user[:password]@]host[:port]")
var host_param = flag.String("host", "https://localhost", "URL for server host in format https://[user[:password]@]hostname[:port]")
var param_param = flag.Int("port", 0, "Server port")
var listen_param = flag.String("listen", "test003_outgoing", "Device name to listen")
var device_param = flag.String("device", "test004_incoming", "Device name")
var metadata_param = flag.String("meta", "", "Meta data for device")
var log_name_param = flag.String("name", "", "Login name")
var log_pwrd_param = flag.String("pwrd", "", "Login password")
var ignoreTLS_param = flag.Bool("k", false, "Ignore TLS certificate errors")

func main() {
	flag.Parse()

	cfg := wclib.ClientCfgNew()
	check(cfg.SetProxy(*proxy_param))
	check(cfg.SetHostURL(*host_param))
	check(cfg.SetPort(*param_param))
	cfg.SetVerifyTLS(!*ignoreTLS_param)
	cfg.SetDevice(*device_param)
	cfg.SetMeta(*metadata_param)

	c, err := wclib.ClientNew(cfg)
	check(err)
	c.SetOnAuthSuccess(AuthSuccess)
	c.SetOnAddLog(OnLog)
	c.SetOnConnected(OnClientStateChange)
	c.SetOnUpdateStreams(OnUpdateStreams)

	c.SetOnAfterLaunchOutStream(onIOTaskStarted)
	c.SetOnSuccessIOStream(onIOTaskFinished)

	fmt.Println("Trying to start client")

	check(c.Start())

	fmt.Println("Client started")

	start_ts := time.Now().UnixMilli()

	for loop := true; loop; {

		switch c.GetClientStatus() {
		case wclib.StateConnectedWrongSID:
			{
				fmt.Println("Trying to authorize")
				check(c.Auth(*log_name_param, *log_pwrd_param))
			}
		case wclib.StateDisconnected:
			{
				loop = false
				break
			}
		default:
			{
				select {
				case v := <-appStream.sequence:
					{
						switch st := v.GetStatus(); st {
						case StatusError:
							{
								fmt.Println("Some error occurred")
								c.Disconnect()
							}
						case StatusAuthorized:
							{
								check(c.UpdateStreams(nil))
							}
						case StatusStreamDetected:
							{
								start_ts = time.Now().UnixMilli()
								go func() {
									fmt.Println("Starting incoming stream...")
									if err := c.LaunchInStream(*listen_param, onNextFrame, nil); err != nil {
										fmt.Printf("Error on starting stream: %v\n", err)
										appStream.SetStatus(StatusError)
									}
								}()
							}
						case StatusIOStarted:
							{

								fr, err := c.PopInFrame()
								if err != nil {
									fmt.Printf("Error on frame extract: %v\n", err)
									appStream.SetStatus(StatusError)
								}

								go func(frame *bytes.Buffer) {
									outfile, err := os.Create(
										fmt.Sprintf("%s/frame%05d%s", OUTPUT_FOLDER, appStream.curFrame, appStream.ext))
									if err != nil {
										appStream.SetStatus(StatusError)
										panic(err)
									}

									appStream.curFrame++

									defer outfile.Close()

									_, err = frame.WriteTo(outfile)

									if err != nil {
										appStream.SetStatus(StatusError)
										panic(err)
									}
								}(fr)
							}
						case StatusIOFinished:
							{
								fmt.Println("Process fully finished")
								c.Disconnect()
							}
						}
					}
				default:
					time.Sleep(time.Duration(appStream.deltaTime) * time.Millisecond)
					cur_ts := time.Now().UnixMilli()

					if (cur_ts-start_ts)/1000 > TIME_OUT {
						fmt.Println("Timeout")
						appStream.SetStatus(StatusIOFinished)
					}
				}
			}
		}

	}

	close(appStream.sequence)

	fmt.Println("Client finished")
}
