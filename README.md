# wcWebCamClient Go Module

This is a library for convenient client work with the wcWebCamServer server via the JSON protocol.

## The structure of the client-server software package
The server designed to collect images and data streams from cameras (devices) and forwards messages between devices to control the periphery via an HTTP 2 connection is [wcwebcamserver (Lazarus/Free Pascal)](https://github.com/iLya2IK/wcwebcamserver).
Library for a C/C++ client is [wcwebcamclient (C/C++)](https://ilya2ik.github.io/wcwebcamclient_lib).
Abstract client for Lazarus is [wccurlclient (Lazarus/Free Pascal)](https://github.com/iLya2IK/wccurlclient).
A detailed implementation of an external device based on "ESP32-CAM" is given in the example [webcamdevice (С)](https://github.com/iLya2IK/webcamdevice).
The example of a desktop application for external device controlling and viewing images is [webcamclientviewer (Lazarus)](https://github.com/iLya2IK/webcamclientviewer).
An example of an Android application for controlling external devices, chatting and streaming is [wcwebcameracontrol (Java)](https://github.com/iLya2IK/wcwebcameracontrol).

## Usage

Use go get -u to download and install the prebuilt package.

```
go get -u github.com/ilya2ik/wcwebcamclient_go
```
or
```
go install github.com/ilya2ik/wcwebcamclient_go@latest
```

### Example for in/out streaming

```go
/* Streaming files from a folder as a set of frames  */

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

   wclib "github.com/ilya2ik/wcwebcamclient_go"
)

/* Relative path to files */
const TO_SEND_FOLDER = "tosend"

/* The max delta time between two frames in milliseconds */
const MAX_DELTA = 600

/* The total program timeout in seconds */
const TIME_OUT = 600

type appStatus int

const (
   StatusWaiting appStatus = iota
   StatusAuthorized
   StatusIOStarted
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

/* Read file to memory */
func raw_read(pathname string) error {

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

func main() {
   flag.Parse()

   cfg := wclib.ClientCfgNew()
   check(cfg.SetHostURL("https://username:password@localhost:8080"))
   cfg.SetDevice("device_to_listen")

   c, err := wclib.ClientNew(cfg)
   check(err)
   c.SetOnAuthSuccess(AuthSuccess)

   c.SetOnAfterLaunchOutStream(onIOTaskStarted)
   c.SetOnSuccessIOStream(onIOTaskFinished)

   fmt.Println("Trying to start client")
   check(c.Start())

   start_ts := time.Now().UnixMilli()
   frame_start_ts := start_ts

   for loop := true; loop; {

      switch c.GetClientStatus() {
      case wclib.StateConnectedWrongSID:
         {
            fmt.Println("Trying to authorize")
            check(c.AuthFromHostUrl())
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
                        files, err := os.ReadDir(TO_SEND_FOLDER)
                        if err != nil {
                           check(err)
                        }

                        for _, f := range files {
                           if strings.HasSuffix(strings.ToUpper(f.Name()), ".RAW") {
                              appStream.frames =
                                            append(appStream.frames,
                                                    TO_SEND_FOLDER+"/"+f.Name())
                           }
                        }

                        if appStream.FramesCount() == 0 {
                           fmt.Println("No frames found")
                           appStream.SetStatus(StatusError)
                        }

                        go func() {
                           fmt.Println("Starting output stream...")
                           if err := c.LaunchOutStream("RAW", MAX_DELTA, nil); err != nil {
                              fmt.Printf("Error on starting stream: %v\n", err)
                              appStream.SetStatus(StatusError)
                           }
                        }()
                     }
                  case StatusIOStarted:
                     {
                        cur_ts := time.Now().UnixMilli()

                        timeout := cur_ts - frame_start_ts

                        if timeout >= MAX_DELTA {
                           frame_start_ts = cur_ts
                           go func() {
                              check(raw_read(appStream.CurFrame()))
                              appStream.NextFrame()

                              appStream.Lock()
                              defer appStream.Unlock()
                              buf := appStream.mem_frame_buffer
                              if buf.Len() > 0 {
                                 appStream.mem_frame_buffer = bytes.NewBuffer(make([]byte, 0))
                                 fmt.Println("Next frame sended")
                                 c.PushOutData(buf)
                              }
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
```

```go
/* Streaming of the incoming data.
   Frames are extracted from the stream and
   saved in OUTPUT_FOLDER in separate files  */

package main

import (
   "bytes"
   "fmt"
   "os"
   "sync"
   "time"

   wclib "github.com/ilya2ik/wcwebcamclient_go"
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

func OnUpdateStreams(tsk wclib.ITask, jsonobj []map[string]any) {
   for _, jStrm := range jsonobj {
      var aStrm wclib.StreamStruct
      err := aStrm.JSONDecode(jStrm)
      check(err)

      if aStrm.Device == "device_to_listen" {
         fmt.Printf("Stream detected `%s`; subproto: `%s`; delta: %d\n",
            aStrm.Device, aStrm.SubProto, int(appStream.deltaTime))
         appStream.deltaTime = int(aStrm.Delta)
         appStream.SetStatus(StatusStreamDetected)
         return
      }
   }
   fmt.Println("Streaming device is not online. Retry")
   appStream.sequence <- &appStream
}

/* Callback. IO stream closed. */
func onIOTaskFinished(tsk wclib.ITask) {
   fmt.Println("Output stream closed")
   appStream.SetStatus(StatusIOFinished)
}

func onNextFrame(tsk wclib.ITask) {
   fmt.Println("New frame captured")
   appStream.sequence <- &appStream
}

func check(e error) {
   if e != nil {
      panic(e)
   }
}

func main() {
   cfg := wclib.ClientCfgNew()
   check(cfg.SetHostURL("https://user:password@localhost:8080"))
   cfg.SetDevice("test")

   c, err := wclib.ClientNew(cfg)
   check(err)
   c.SetOnAuthSuccess(AuthSuccess)
   c.SetOnUpdateStreams(OnUpdateStreams)
   c.SetOnSuccessIOStream(onIOTaskFinished)

   fmt.Println("Trying to start client")
   check(c.Start())

   var start_ts int64

   for loop := true; loop; {

      switch c.GetClientStatus() {
      case wclib.StateConnectedWrongSID:
         {
            fmt.Println("Trying to authorize")
            check(c.AuthFromHostUrl())
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
                           if err := c.LaunchInStream("device_to_listen", onNextFrame, nil); err != nil {
                              fmt.Printf("Error on starting stream: %v\n", err)
                              appStream.SetStatus(StatusError)
                           } else {
                              appStream.SetStatus(StatusIOStarted)
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
                              fmt.Sprintf("%s/frame%05d.raw", OUTPUT_FOLDER, appStream.curFrame))
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
```

## Documents

[wcWebCamClient library API User's Guide - Doxygen](https://ilya2ik.github.io/wcwebcamclient_lib/index.html)
