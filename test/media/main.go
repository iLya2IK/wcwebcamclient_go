/*===============================================================*/
/* This is an example of how to use the wcWebCamClient library.  */
/* In this example, a client is created, authorized on the       */
/* server, uploads a media record and downloads it to disk.      */
/* After all the saved record will removed from the server by    */
/* the request.                                                  */
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
	"path"
	"strings"
	"sync"
	"time"

	"libs.com/wclib"
)

type mediaStatus int

const (
	StatusWaiting mediaStatus = iota
	StatusSended
	StatusRIDObtained
	StatusMetaObtained
	StatusDownloaded
	StatusRIDDeleted
	StatusError
)

type mediaRecord struct {
	mux      sync.Mutex
	metadata string
	id       int
	status   mediaStatus
	sequence chan *mediaRecord
}

var record = mediaRecord{
	metadata: "",
	id:       0,
	sequence: make(chan *mediaRecord, 4),
}

func (rec *mediaRecord) lock() {
	rec.mux.Lock()
}

func (rec *mediaRecord) unlock() {
	rec.mux.Unlock()
}

func (rec *mediaRecord) SetRID(rid int) {
	rec.lock()
	defer rec.unlock()

	rec.id = rid
}

func (rec *mediaRecord) SetMeta(meta string) {
	rec.lock()
	defer rec.unlock()

	rec.metadata = meta
}

func (rec *mediaRecord) SetStatus(st mediaStatus) {
	rec.lock()
	defer rec.unlock()

	if st != rec.status {
		rec.status = st
		rec.sequence <- rec
	}
}

func (rec *mediaRecord) GetRID() int {
	rec.lock()
	defer rec.unlock()

	return rec.id
}

func (rec *mediaRecord) GetMeta() string {
	rec.lock()
	defer rec.unlock()

	return rec.metadata
}

func (rec *mediaRecord) GetStatus() mediaStatus {
	rec.lock()
	defer rec.unlock()

	return rec.status
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

func check(e error) {
	if e != nil {
		panic(e)
	}
}

/* Callback. The list of media records was changed. */
func OnGetRecords(task wclib.ITask, jArr []map[string]any) {
	if jArr != nil {
		for j := len(jArr) - 1; j >= 0; j-- {
			media := wclib.MediaStruct{}
			if err := media.JSONDecode(jArr[j]); err == nil {
				if media.Device == *device_param {
					fmt.Printf("Records received - last record id %d - stamp %s\n", int(media.Rid), media.Stamp)
					record.SetRID(int(media.Rid))
					record.SetStatus(StatusRIDObtained)
					return
				}
			}
		}
	}

	record.SetStatus(StatusError)
}

/* Callback. The media record metadata received. */
func OnGetRecordMeta(task wclib.ITask, jObj map[string]any) {
	if jObj != nil {
		meta := wclib.MediaMetaStruct{}
		if err := meta.JSONDecode(jObj); err == nil {
			fmt.Printf("Record %d - meta data received: \"%s\"\n", record.id, meta.Meta)
			record.SetMeta(meta.Meta)
			record.SetStatus(StatusMetaObtained)
			return
		}
	}

	record.SetStatus(StatusError)
}

/* Callback. The request to save the media record has been completed. The response has arrived. */
func OnAfterSaveRecord(task wclib.ITask) {
	file_name, ok := task.GetUserData().(*string)

	if ok {
		fmt.Printf("File \"%s\" sended\n", *file_name)
		record.SetStatus(StatusSended)
	} else {
		record.SetStatus(StatusError)
	}
}

/* Callback. The request to get the media record has been completed. The response has arrived. */
func OnGetRecordData(task wclib.ITask, data *bytes.Buffer) {
	len := data.Len()

	fmt.Printf("Record %d (%d bytes) successfully downloaded\n", record.id, len)

	output_file := fmt.Sprintf("record %d.%s", record.id, record.metadata)
	outfile, err := os.Create(output_file)
	if err != nil {
		record.SetStatus(StatusError)
		panic(err)
	}

	defer outfile.Close()

	_, err = data.WriteTo(outfile)

	if err != nil {
		record.SetStatus(StatusError)
		panic(err)
	}

	fmt.Printf("File \"%s\" (%d bytes) successfully saved\n", output_file, len)

	record.SetStatus(StatusDownloaded)
}

var proxy_param = flag.String("proxy", "", "Proxy in format [scheme:]//[user[:password]@]host[:port]")
var host_param = flag.String("host", "https://localhost", "URL for server host in format https://[user[:password]@]hostname[:port]")
var param_param = flag.Int("port", 0, "Server port")
var device_param = flag.String("device", "test001", "Device name")
var metadata_param = flag.String("meta", "", "Meta data for device")
var log_name_param = flag.String("name", "", "Login name")
var log_pwrd_param = flag.String("pwrd", "", "Login password")
var ignoreTLS_param = flag.Bool("k", false, "Ignore TLS certificate errors")
var inputfile_param = flag.String("i", "morti.png", "The name of file to send")

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
	c.SetOnAuthSuccess(func(tsk wclib.ITask) {
		fmt.Println("SID ", tsk.GetClient().GetSID())
		record.sequence <- &record
	})
	c.SetOnAddLog(func(client *wclib.WCClient, str string) {
		fmt.Println(str)
	})
	c.SetOnConnected(OnClientStateChange)
	c.SetOnReqRecordData(OnGetRecordData)

	fmt.Println("Trying to start client")

	check(c.Start())

	fmt.Println("Client started")

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
				case v := <-record.sequence:
					{
						switch st := v.GetStatus(); st {
						case StatusError:
							{
								fmt.Println("Some error occurred")
								c.Disconnect()
							}
						case StatusWaiting:
							{
								fp, err := os.Open(*inputfile_param)
								check(err)

								fi, err := fp.Stat()
								check(err)

								go func() {
									fmt.Println("Sending media record...")
									ext := strings.ToUpper(path.Ext(*inputfile_param))
									if len(ext) > 0 {
										ext = ext[1:]
									}
									if err := c.SaveRecord(fp, fi.Size(), ext, inputfile_param, OnAfterSaveRecord); err != nil {
										fmt.Printf("Error on sending record: %v\n", err)
										record.SetStatus(StatusError)
									}
								}()
							}
						case StatusSended:
							{
								go func() {
									if err := c.UpdateRecords(OnGetRecords); err != nil {
										fmt.Printf("Error on updating records: %v\n", err)
										record.SetStatus(StatusError)
									}
								}()
							}
						case StatusRIDObtained:
							{
								go func() {
									if err := c.RequestRecordMeta(record.GetRID(), OnGetRecordMeta); err != nil {
										fmt.Printf("Error on requesting metadata: %v\n", err)
										record.SetStatus(StatusError)
									}
								}()
							}
						case StatusMetaObtained:
							{
								go func() {
									if err := c.RequestRecord(record.GetRID()); err != nil {
										fmt.Printf("Error on requesting data: %v\n", err)
										record.SetStatus(StatusError)
									}
								}()
							}
						case StatusDownloaded:
							{
								go func() {
									arr := [...]int{record.GetRID()}
									if err := c.DeleteRecords(arr[0:], func(tsk wclib.ITask) {
										fmt.Printf("Media record %d deleted\n", record.id)
										record.SetStatus(StatusRIDDeleted)
									}); err != nil {
										fmt.Printf("Error on deleting records: %v\n", err)
										record.SetStatus(StatusError)
									}
								}()
							}
						case StatusRIDDeleted:
							{
								fmt.Println("Process fully finished")
								c.Disconnect()
							}
						}
					}
				default:
					time.Sleep(250 * time.Millisecond)
				}
			}
		}

	}

	close(record.sequence)

	fmt.Println("Client finished")
}
