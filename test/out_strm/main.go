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
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"libs.com/wclib"
)

/* The max delta time between two frames in milliseconds */
const MAX_DELTA = 300

/* The total program timeout in seconds */
const TIME_OUT = 60

type appStatus int

const (
	StatusWaiting appStatus = iota
	StatusAuthorized
	StatusIOStarted
	StatusFrameSending
	StatusFrameSended
	StatusIOFinished
	StatusError
)

type appStreamStruct struct {
	mux              sync.Mutex
	frames           []string
	cur_frame        int
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

func (app *appStreamStruct) FramesCount() int {
	app.Lock()
	defer app.Unlock()

	return len(app.frames)
}

func (app *appStreamStruct) CurFrame() string {
	app.Lock()
	defer app.Unlock()

	return app.frames[app.cur_frame]
}

func (app *appStreamStruct) NextFrame() {
	app.Lock()
	defer app.Unlock()

	app.cur_frame++
	if app.cur_frame >= len(app.frames) {
		app.cur_frame = 0
	}
}

var appStream = appStreamStruct{
	status:           StatusWaiting,
	mem_frame_buffer: bytes.NewBuffer(make([]byte, 0)),
	sequence:         make(chan *appStreamStruct, 16),
}

/* Read png-image to memory */
func png_read(pathname string) error {

	fp, err := os.Open(pathname)
	if err != nil {
		return err
	}

	defer fp.Close()

	fi, err := fp.Stat()
	if err != nil {
		return err
	}

	var fsize uint32 = uint32(fi.Size())

	appStream.Lock()
	defer appStream.Unlock()

	appStream.mem_frame_buffer.Reset()

	frame_buffer := (((fsize + uint32(wclib.WC_STREAM_FRAME_HEADER_SIZE)) >> 12) + 1) << 12
	if appStream.mem_frame_buffer.Cap() < int(fsize) {
		appStream.mem_frame_buffer.Grow(int(frame_buffer))
	}
	binary.Write(appStream.mem_frame_buffer, binary.LittleEndian, wclib.WC_FRAME_START_SEQ)
	binary.Write(appStream.mem_frame_buffer, binary.LittleEndian, fsize)
	_, err = appStream.mem_frame_buffer.ReadFrom(fp)
	if err != nil {
		return err
	}

	return nil
}

func AuthSuccess(tsk wclib.ITask) {
	fmt.Println("SID ", tsk.GetClient().GetSID())
	appStream.SetStatus(StatusAuthorized)
}

func OnLog(client *wclib.WCClient, str string) {
	fmt.Println(str)
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

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var proxy_param = flag.String("proxy", "", "Proxy in format [scheme:]//[user[:password]@]host[:port]")
var host_param = flag.String("host", "https://localhost", "URL for server host in format https://[user[:password]@]hostname[:port]")
var param_param = flag.Int("port", 0, "Server port")
var device_param = flag.String("device", "test001", "Device name")
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

	c.SetOnAfterLaunchOutStream(onIOTaskStarted)
	c.SetOnSuccessIOStream(onIOTaskFinished)

	fmt.Println("Trying to start client")

	check(c.Start())

	fmt.Println("Client started")

	start_ts := time.Now().UnixMilli()
	frame_start_ts := start_ts

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
								files, err := os.ReadDir("images")
								if err != nil {
									check(err)
								}

								for _, f := range files {
									if strings.HasSuffix(strings.ToLower(f.Name()), ".png") {
										appStream.frames = append(appStream.frames, "images/"+f.Name())
									}
								}

								if appStream.FramesCount() == 0 {
									fmt.Println("No frames found")
									appStream.SetStatus(StatusError)
								}
								go func() {
									fmt.Println("Starting output stream...")
									if err := c.LaunchOutStream("RAW_PNG", MAX_DELTA, nil); err != nil {
										fmt.Printf("Error on starting stream: %v\n", err)
										appStream.SetStatus(StatusError)
									}
								}()
							}
						case StatusIOStarted, StatusFrameSended:
							{
								cur_ts := time.Now().UnixMilli()

								timeout := cur_ts - frame_start_ts

								if timeout >= MAX_DELTA {
									frame_start_ts = cur_ts
									go func() {
										check(png_read(appStream.CurFrame()))
										appStream.NextFrame()
										appStream.Lock()
										defer appStream.Unlock()
										buf := appStream.mem_frame_buffer
										appStream.mem_frame_buffer = bytes.NewBuffer(make([]byte, 0))
										fmt.Println("Next frame sended")
										c.PushOutData(buf)
										appStream.sequence <- &appStream
									}()
								} else {
									time.Sleep(10 * time.Millisecond)
									appStream.sequence <- &appStream
								}
							}
						case StatusIOFinished:
							{
								fmt.Println("Process fully finished")
								c.Disconnect()
							}
						}
					}
				default:
					time.Sleep(250 * time.Millisecond)
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
