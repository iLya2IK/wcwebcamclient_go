/*===============================================================*/
/* This is an example of how to use the wcWebCamClient library.  */
/* In this example, a client is created, authorized on the       */
/* server, and downloads a list of active devices.               */
/*                                                               */
/* Part of WebCamClientLib go module                             */
/*                                                               */
/* Copyright 2024 Ilya Medvedkov                                 */
/*===============================================================*/

package main

import (
	"flag"
	"fmt"
	"time"

	"libs.com/wclib"
)

func AuthSuccess(tsk wclib.ITask) {
	fmt.Println("SID ", tsk.GetClient().GetSID())
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
var lname_param = flag.String("name", "", "Login name")
var lpwrd_param = flag.String("pwrd", "", "Login password")
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

	fmt.Println("Trying to start client")

	check(c.Start())

	fmt.Println("Client started")

	type fire struct {
		command   func(...any) error
		onsuccess any
		timeout   int64
		mask      []wclib.ClientStatus
	}

	sheduler := make(chan *fire, 3)
	sheduler <- &fire{
		command: c.UpdateMsgs,
		onsuccess: func(tsk wclib.ITask, res []map[string]any) {
			for _, v := range res {
				fmt.Println(v)
			}
		},
		timeout: 2000,
		mask:    []wclib.ClientStatus{wclib.StateConnected},
	}
	sheduler <- &fire{
		command: c.UpdateDevices,
		onsuccess: func(tsk wclib.ITask, res []map[string]any) {
			for _, v := range res {
				fmt.Println(v)
			}
		},
		timeout: 1000,
		mask:    []wclib.ClientStatus{wclib.StateConnected},
	}
	sheduler <- &fire{
		command: func(...any) error {
			return c.Disconnect()
		},
		timeout: 5000,
		mask: []wclib.ClientStatus{
			wclib.StateConnected,
			wclib.StateConnectedAuthorization,
			wclib.StateConnectedWrongSID,
		},
	}

	for loop := true; loop; {

		switch c.GetClientStatus() {
		case wclib.StateConnectedWrongSID:
			{
				fmt.Println("Trying to authorize")
				check(c.Auth(*lname_param, *lpwrd_param))
			}
		case wclib.StateDisconnected:
			{
				loop = false
				break
			}
		default:
			{
				select {
				case v := <-sheduler:
					{
						go func(v *fire) {
							time.Sleep(time.Duration(v.timeout) * time.Millisecond)
							if c.IsClientStatusInRange(v.mask) {
								check(v.command(v.onsuccess))
							}
							sheduler <- v
						}(v)
					}
				default:
					time.Sleep(250 * time.Millisecond)
				}
			}
		}

	}

	close(sheduler)

	fmt.Println("Client finished")
}
