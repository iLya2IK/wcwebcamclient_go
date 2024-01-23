/*===============================================================*/
/* The WebCamClientLib go module                                 */
/*                                                               */
/* Copyright 2024 Ilya Medvedkov                                 */
/*===============================================================*/

package wclib

import (
	"bytes"
	"container/list"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"time"
)

type ClientStatus int

const (
	StateWaiting ClientStatus = iota
	StateConnectedWrongSID
	StateConnectedAuthorization
	StateConnected
	StateDisconnected
)

type ClientState int

const STATE_CONNECTION ClientState = 0
const STATE_VERIFYTLS ClientState = 1
const STATE_ERROR ClientState = 2
const STATE_LOG ClientState = 3
const STATE_STREAMING ClientState = 4
const STATE_STREAMS ClientState = 5
const STATE_DEVICES ClientState = 6
const STATE_RECORDS ClientState = 7
const STATE_RECORDSSTAMP ClientState = 8
const STATE_MSGS ClientState = 9
const STATE_SENDWITHSYNC ClientState = 22
const STATE_MSGSSTAMP ClientState = 10
const STATE_METADATA ClientState = 11
const STATE_DEVICENAME ClientState = 12
const STATE_SID ClientState = 13
const STATE_HOSTNAME ClientState = 14
const STATE_PROXY ClientState = 15
const STATE_PROXYAUTH ClientState = 16
const STATE_PROXYPROTOCOL ClientState = 17
const STATE_PROXYHOST ClientState = 18
const STATE_PROXYPORT ClientState = 19
const STATE_PROXYUSER ClientState = 20
const STATE_PROXYPWRD ClientState = 21

const JSON_RPC_OK string = "OK"
const JSON_RPC_BAD string = "BAD"

const REST_SYNC_MSG string = "{\"msg\":\"sync\"}"
const JSON_RPC_SYNC string = "sync"
const JSON_RPC_CONFIG string = "config"
const JSON_RPC_MSG string = "msg"
const JSON_RPC_MSGS string = "msgs"
const JSON_RPC_RECORDS string = "records"
const JSON_RPC_DEVICES string = "devices"
const JSON_RPC_RESULT string = "result"
const JSON_RPC_CODE string = "code"
const JSON_RPC_NAME string = "name"
const JSON_RPC_PASS string = "pass"
const JSON_RPC_SHASH string = "shash"
const JSON_RPC_META string = "meta"
const JSON_RPC_STAMP string = "stamp"
const JSON_RPC_MID string = "mid"
const JSON_RPC_RID string = "rid"
const JSON_RPC_DEVICE string = "device"
const JSON_RPC_TARGET string = "target"
const JSON_RPC_PARAMS string = "params"
const JSON_RPC_SUBPROTO string = "subproto"
const JSON_RPC_DELTA string = "delta"

const NO_ERROR = 0
const UNSPECIFIED = 1
const INTERNAL_UNKNOWN_ERROR = 2
const DATABASE_FAIL = 3
const JSON_PARSER_FAIL = 4
const JSON_FAIL = 5
const NO_SUCH_SESSION = 6
const NO_SUCH_USER = 7
const NO_DEVICES_ONLINE = 8
const NO_SUCH_RECORD = 9
const NO_DATA_RETURNED = 10
const EMPTY_REQUEST = 11
const MALFORMED_REQUEST = 12
const NO_CHANNEL = 13
const ERRORED_STREAM = 14
const NO_SUCH_DEVICE = 15

var RESPONSE_ERRORS = [...]string{
	"NO_ERROR",
	"UNSPECIFIED",
	"INTERNAL_UNKNOWN_ERROR",
	"DATABASE_FAIL",
	"JSON_PARSER_FAIL",
	"JSON_FAIL",
	"NO_SUCH_SESSION",
	"NO_SUCH_USER",
	"NO_DEVICES_ONLINE",
	"NO_SUCH_RECORD",
	"NO_DATA_RETURNED",
	"EMPTY_REQUEST",
	"MALFORMED_REQUEST",
	"NO_CHANNEL",
	"ERRORED_STREAM",
	"NO_SUCH_DEVICE"}

type EmptyNotifyFunc func(client *WCClient)
type NotifyEventFunc func(client *WCClient, data any)
type TaskNotifyFunc func(tsk *Task)
type TaskErrorFunc func(tsk *Task, err error)
type ConnNotifyEventFunc func(client *WCClient, status ClientStatus)
type StringNotifyFunc func(client *WCClient, value string)
type DataNotifyEventFunc func(tsk *Task, data []byte)
type JSONArrayNotifyEventFunc func(tsk *Task, jsonresult []any)
type JSONNotifyEventFunc func(tsk *Task, jsonresult any)

func ClientStatusText(status ClientStatus) string {
	switch status {
	case StateWaiting:
		return "Waiting"
	case StateConnected:
		return "Connected"
	case StateDisconnected:
		return "Disconnected"
	case StateConnectedWrongSID:
		return "No SID"
	default:
		return ""
	}
}

type WCClientConfig struct {
	hosturl *url.URL
	proxy   *url.URL

	host string
	port int

	device string
	meta   string

	secure bool

	locked bool
}

type ClientStatusThreadSafe struct {
	mux sync.Mutex

	value ClientStatus
}

func (c *ClientStatusThreadSafe) setValue(st ClientStatus) bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	if st != c.value {
		c.value = st
		return true
	} else {
		return false
	}
}

func (c *ClientStatusThreadSafe) getValue() ClientStatus {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.value
}

type StringThreadSafe struct {
	mux sync.Mutex

	Value string
}

func (c *StringThreadSafe) SetValue(str string) bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	if str != c.Value {
		c.Value = str
		return true
	} else {
		return false
	}
}

func (c *StringThreadSafe) Clear() {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.Value = ""
}

func (c *StringThreadSafe) GetValueUnsafePtr() *string {
	c.mux.Lock()
	defer c.mux.Unlock()

	return &c.Value
}

func (c *StringThreadSafe) GetValue() string {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.Value
}

type BoolThreadSafe struct {
	mux sync.Mutex

	Value bool
}

func (c *BoolThreadSafe) SetValue(val bool) bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	if val != c.Value {
		c.Value = val
		return true
	} else {
		return false
	}
}

func (c *BoolThreadSafe) GetValue() bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.Value
}

type WCClient struct {
	//@private
	cbmux sync.Mutex

	onSuccessAuth TaskNotifyFunc      /* Successful authorization. */
	onConnected   ConnNotifyEventFunc /* The connection state has been changed. */
	onDisconnect  NotifyEventFunc     /* Client has been disconnected. */
	onSIDSetted   StringNotifyFunc    /* The session id has been changed. */
	onAddLog      StringNotifyFunc    /* Added new log entry. */

	/* streams block */
	onAfterLaunchInStream  TaskNotifyFunc /* Incoming stream started. */
	onAfterLaunchOutStream TaskNotifyFunc /* Outgoing stream started. */
	onSuccessIOStream      TaskNotifyFunc /* IO stream terminated for some reason. */

	/* data blobs block */
	onSuccessSaveRecord    TaskNotifyFunc      /* The request to save the media record has been completed. The response has arrived. */
	onSuccessRequestRecord DataNotifyEventFunc /* The request to get the media record has been completed. The response has arrived. */

	/* JSON block */
	onSuccessUpdateRecords     JSONArrayNotifyEventFunc /* The request to update list of records has been completed. The response has arrived. */
	onSuccessUpdateDevices     JSONArrayNotifyEventFunc /* The request to update list of online devices has been completed. The response has arrived. */
	onSuccessUpdateStreams     JSONArrayNotifyEventFunc /* The request to update list of streaming devices has been completed. The response has arrived. */
	onSuccessUpdateMsgs        JSONArrayNotifyEventFunc /* The request to update list of messages has been completed. The response has arrived. */
	onSuccessSendMsg           JSONNotifyEventFunc      /* The request to send message has been completed. The response has arrived.  */
	onSuccessRequestRecordMeta JSONNotifyEventFunc      /* The request to get metadata for the media record has been completed. The response has arrived. */
	onSuccessGetConfig         JSONNotifyEventFunc      /* The request to get actual config has been completed. The response has arrived. */
	onSuccessDeleteRecords     JSONNotifyEventFunc      /* The request to delete records has been completed. The response has arrived. */

	/* channels */
	wrk       chan *Task
	finished  chan *Task
	states    chan ClientState
	ferr      chan *Task
	terminate chan bool

	/* state */
	clientst   *ClientStatusThreadSafe
	lstError   *StringThreadSafe
	sid        *StringThreadSafe
	lmsgstamp  *StringThreadSafe
	lrecstamp  *StringThreadSafe
	needtosync *BoolThreadSafe
	log        *StringListThreadSafe

	/* config */
	cfg *WCClientConfig

	/* http */
	inclient   *http.Client
	context    context.Context
	cancelfunc context.CancelFunc
}

type StringListThreadSafe struct {
	mux sync.Mutex

	value *list.List
}

/* StringListThreadSafe */

func (c *StringListThreadSafe) PushBack(str string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.value.PushBack(str)
}

func (c *StringListThreadSafe) NotEmpty() bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.value.Len() > 0
}

func (c *StringListThreadSafe) PopFromLog() string {
	c.mux.Lock()
	defer c.mux.Unlock()

	el := c.value.Front()
	if el != nil {
		c.value.Remove(el)
		return el.Value.(string)
	} else {
		return ""
	}
}

func (c *StringListThreadSafe) Clear() {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.value = list.New()
}

/* ErrNoPasswordDetected */

type ErrWrongAuthData struct{}

func ThrowErrWrongAuthData() *ErrWrongAuthData {
	return &ErrWrongAuthData{}
}

func (e *ErrWrongAuthData) Error() string {
	return "The username or (and) password are not provided"
}

/* ErrWrongStatus */

type ErrWrongStatus struct {
	status ClientStatus
}

func ThrowErrWrongStatus(status ClientStatus) *ErrWrongStatus {
	return &ErrWrongStatus{status: status}
}

func (e *ErrWrongStatus) Error() string {
	return fmt.Sprintf("Wrong client status (%s)", ClientStatusText(e.status))
}

/* ErrParam */

type ErrParam struct {
	val any
}

func ThrowErrParam(val any) *ErrParam {
	return &ErrParam{val: val}
}

func (e *ErrParam) Error() string {
	return fmt.Sprintf("Param with value (%v) is not accepted", e.val)
}

/* ErrMalformedResponse */

type EMalformedKind int

const (
	EMKWrongType EMalformedKind = iota
	EMKFieldExpected
	EMKUnexpectedValue
)

type ErrMalformedResponse struct {
	kind EMalformedKind
	arg  string
	add  any
}

func ThrowErrMalformedResponse(kind EMalformedKind, arg string, par any) *ErrMalformedResponse {
	return &ErrMalformedResponse{kind: kind, arg: arg, add: par}
}

func (e *ErrMalformedResponse) Error() string {
	switch e.kind {
	case EMKWrongType:
		return fmt.Sprintf("Argument \"%s\" has wrong type. Expected: %v", e.arg, e.add)
	case EMKFieldExpected:
		return fmt.Sprintf("Expected field \"%s\" was not found", e.arg)
	case EMKUnexpectedValue:
		return fmt.Sprintf("Unexpected value (%v) for field \"%s\" was found", e.add, e.arg)
	default:
		return fmt.Sprintf("Malformed response")
	}
}

/* ErrWrongHostName */

type ErrWrongHostName struct{ url string }

func ThrowErrWrongHostName(url string) *ErrWrongHostName {
	return &ErrWrongHostName{url: url}
}

func (e *ErrWrongHostName) Error() string {
	return fmt.Sprintf("Host name (%s) has wrong format, expected: https://[user[:password]@]hostname[:port]", e.url)
}

/* ErrLockedConfig */

type ErrLockedConfig struct{}

func ThrowErrLockedConfig() *ErrLockedConfig {
	return &ErrLockedConfig{}
}

func (e *ErrLockedConfig) Error() string { return "Trying to change locked config" }

/* ErrBadResponse */

type ErrBadResponse struct {
	code int
}

func ThrowErrBadResponse(code int) *ErrBadResponse {
	return &ErrBadResponse{code}
}

func (e *ErrBadResponse) Error() string {
	return fmt.Sprintf("Bad code %d (%s) in JSON response", e.code, RESPONSE_ERRORS[e.code])
}

/* ErrHttpStatus */

type ErrHttpStatus struct {
	status int
}

func ThrowErrHttpStatus(status int) *ErrHttpStatus {
	return &ErrHttpStatus{status}
}

func (e *ErrHttpStatus) Error() string {
	return fmt.Sprintf("Bad HTTP status %d (%s) in response", e.status, http.StatusText(e.status))
}

/* ErrEmptyResponse */

type ErrEmptyResponse struct{}

func ThrowErrEmptyResponse() *ErrEmptyResponse {
	return &ErrEmptyResponse{}
}

func (*ErrEmptyResponse) Error() string {
	return "Response is empty"
}

/* ErrAuth */

type ErrAuth struct{}

func ThrowErrAuth() *ErrAuth {
	return &ErrAuth{}
}

func (*ErrAuth) Error() string {
	return "Authentification error"
}

type Task struct {
	//@private
	client   *WCClient
	request  *http.Request
	response *http.Response

	userdata any

	lsterr    error
	onSuccess TaskNotifyFunc
	onError   TaskErrorFunc
}

/* Task private methods */

func (tsk *Task) pushError(err error) {
	tsk.lsterr = err
	tsk.client.ferr <- tsk
}

func (tsk *Task) execute(after chan *Task) {
	var err error
	tsk.response, err = tsk.client.inclient.Do(tsk.request)

	if err != nil {
		tsk.pushError(err)
	} else {
		after <- tsk
	}
}

func getwcObjArray(res map[string]any, field string) ([]any, error) {
	const cARRAYOFOBJS = `"array of objects"`
	v, ok := res[field]
	if ok {
		switch reflect.TypeOf(v).Kind() {
		case reflect.Array, reflect.Slice:
			{
				if len(v.([]any)) > 0 {
					if reflect.TypeOf(v.([]any)[0]).Kind() == reflect.Map {
						return v.([]any), nil
					} else {
						return nil, ThrowErrMalformedResponse(EMKWrongType, field, cARRAYOFOBJS)
					}
				} else {
					return v.([]any), nil
				}
			}
		default:
			return nil, ThrowErrMalformedResponse(EMKWrongType, field, cARRAYOFOBJS)
		}
	} else {
		return nil, ThrowErrMalformedResponse(EMKFieldExpected, field, nil)
	}
}

func getwcValue(res map[string]any, field string, def any, mandatory bool) (any, error) {
	v, ok := res[field]
	if ok {
		vt := reflect.TypeOf(v)
		dt := reflect.TypeOf(def)
		if vt == dt {
			return v, nil
		} else if (vt.Kind() == reflect.Float64) && (dt.Kind() == reflect.Int) {
			return int(v.(float64)), nil
		} else {
			return def, ThrowErrMalformedResponse(EMKWrongType, field, fmt.Sprintf("%v", dt))
		}
	} else {
		if mandatory {
			return def, ThrowErrMalformedResponse(EMKFieldExpected, field, nil)
		} else {
			return def, nil
		}
	}
}

func getwcResult(res map[string]any) (string, error) {
	str, err := getwcValue(res, JSON_RPC_RESULT, "", true)

	return str.(string), err
}

func getwcResultCode(res map[string]any) (int, error) {
	code, err := getwcValue(res, JSON_RPC_CODE, 0, true)

	return code.(int), err
}

// func (tsk *Task) successJSONresponse(res wcJsonResulter) bool {
func (tsk *Task) successJSONresponse(res map[string]any) bool {
	defer tsk.response.Body.Close()

	d := json.NewDecoder(tsk.response.Body)
	if err := d.Decode(&res); err != nil {
		tsk.pushError(err)
		return false
	}

	v, err := getwcResult(res)

	if err != nil {
		tsk.pushError(err)
		return false
	}

	if v == JSON_RPC_OK {
		return true
	} else if v == JSON_RPC_BAD {
		code, err := getwcResultCode(res)

		if err != nil {
			tsk.pushError(err)
			return false
		}

		tsk.pushError(ThrowErrBadResponse(code))
		return false
	} else {
		tsk.pushError(ThrowErrMalformedResponse(EMKUnexpectedValue, JSON_RPC_CODE, v))
		return false
	}
}

/* Task public methods */

func (tsk *Task) GetClient() *WCClient {
	return tsk.client
}

func (tsk *Task) GetUserData() any {
	return tsk.userdata
}

func (tsk *Task) SetUserData(data any) {
	tsk.userdata = data
}

/* WCClientConfig constructor */

// Create new empty client configuration
func ClientCfgNew() *WCClientConfig {
	return &(WCClientConfig{locked: false})
}

/* WCClientConfig public metehods */

// Set new meta data for the device (sa. "authorize.json" - WCPD)
func (c *WCClientConfig) SetMeta(val string) error {
	if c.locked {
		return ThrowErrLockedConfig()
	}

	c.meta = val

	return nil
}

// Get the assigned meta data for the device (sa. "authorize.json" - WCPD)
func (c *WCClientConfig) GetMeta() string {
	return c.meta
}

// Set the server proxy params in format `[scheme:]//[user[:password]@]host[:port]`
func (c *WCClientConfig) SetProxy(proxy string) error {
	if c.locked {
		return ThrowErrLockedConfig()
	}

	if len(proxy) == 0 {
		return nil
	}

	var err error
	c.proxy, err = url.Parse(proxy)

	if err != nil {
		return err
	}

	return nil
}

// Set the server host address `https://[username[:password]@]hostname[:port]`
func (c *WCClientConfig) SetHostURL(hosturl string) error {
	if c.locked {
		return ThrowErrLockedConfig()
	}

	if len(hosturl) == 0 {
		return ThrowErrWrongHostName(hosturl)
	}

	var err error
	c.hosturl, err = url.Parse(hosturl)
	if err != nil {
		return err
	}

	if s := c.hosturl.Scheme; s != "https" {
		return ThrowErrWrongHostName(hosturl)
	}

	var parsed_host = ""
	var parsed_post = 0

	parsed_host = c.hosturl.Hostname()

	if p := c.hosturl.Port(); len(p) > 0 {
		parsed_post, _ = strconv.Atoi(p)
	}

	if len(parsed_host) == 0 {
		return ThrowErrWrongHostName(hosturl)
	}

	c.host = c.hosturl.Scheme + "://" + parsed_host

	if parsed_post > 0 {
		c.port = parsed_post
	}

	return nil
}

// Get the assigned server host address
func (c *WCClientConfig) GetHost() string {
	return c.host
}

// Set the server host port
func (c *WCClientConfig) SetPort(port int) error {
	if c.locked {
		return ThrowErrLockedConfig()
	}

	if port > 0 {
		c.port = port
	}

	return nil
}

// Get the assigned server host port
func (c *WCClientConfig) GetPort() int {
	return c.port
}

// Set the device name
func (c *WCClientConfig) SetDevice(device string) error {
	if c.locked {
		return ThrowErrLockedConfig()
	}

	c.device = device
	return nil
}

// Get the assigned device name
func (c *WCClientConfig) GetDevice() string {
	return c.device
}

// Get the complete url for the JSON-request
func (c *WCClientConfig) GetUrl(command string) string {
	if c.GetPort() == 0 {
		return fmt.Sprintf("%s/%s", c.GetHost(), command)
	} else {
		return fmt.Sprintf("%s:%d/%s", c.GetHost(), c.GetPort(), command)
	}
}

// Set or unset the flag - should client verify TLS certificate
func (c *WCClientConfig) SetVerifyTLS(sec bool) error {
	if c.locked {
		return ThrowErrLockedConfig()
	}

	c.secure = sec
	return nil
}

// Get the value of the flag - should client verify TLS certificate
func (c *WCClientConfig) GetVerifyTLS() bool {
	return c.secure
}

/* WCClient constructor */

/*
Create client.

	`cfg` is a complete configuration for the client.
	`return` pointer to the new client instance and the error object.
*/
func ClientNew(cfg *WCClientConfig) (*WCClient, error) {

	if cfg == nil {
		return nil, fmt.Errorf("Client config is empty")
	}

	cfg.locked = true

	tr := http.DefaultTransport.(*http.Transport).Clone()

	if !cfg.secure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if cfg.proxy != nil {
		tr.Proxy = http.ProxyURL(cfg.proxy)
	}

	tr.ForceAttemptHTTP2 = true

	client := &http.Client{Transport: tr}

	wcclient := &(WCClient{wrk: make(chan *Task, 16),
		terminate:  make(chan bool, 2),
		finished:   make(chan *Task, 16),
		ferr:       make(chan *Task, 16),
		states:     make(chan ClientState, 32),
		needtosync: &BoolThreadSafe{},
		lmsgstamp:  &StringThreadSafe{},
		lrecstamp:  &StringThreadSafe{},
		lstError:   &StringThreadSafe{},
		sid:        &StringThreadSafe{},
		clientst:   &ClientStatusThreadSafe{value: StateWaiting},
		inclient:   client,
		log:        &StringListThreadSafe{value: list.New()},
		cfg:        cfg,
	})

	wcclient.context, wcclient.cancelfunc = context.WithCancel(context.Background())
	return wcclient, nil
}

/* WCClient private methods */

func (c *WCClient) lockcbks() {
	c.cbmux.Lock()
}

func (c *WCClient) unlockcbks() {
	c.cbmux.Unlock()
}

func (c *WCClient) stop() {
	c.setClientStatus(StateDisconnected)
}

func (c *WCClient) start() {
	c.setClientStatus(StateConnectedWrongSID)
}

func (c *WCClient) setClientStatus(st ClientStatus) {
	if c.clientst.setValue(st) {
		c.lockcbks()
		defer c.unlockcbks()

		if c.onConnected != nil {
			c.onConnected(c, st)
		}

		if st == StateDisconnected {
			if c.onDisconnect != nil {
				c.onDisconnect(c, nil)
			}
		}
	}
}

func (c *WCClient) setLastError(what string) {
	c.lstError.SetValue(what)
}

func (c *WCClient) internalStart() {

	for c.Working() {
		if c.inclient != nil {
			select {
			case tsk := <-c.wrk:
				go tsk.execute(c.finished)
			case rtsk := <-c.finished:
				{
					if rtsk.response == nil {
						rtsk.pushError(ThrowErrEmptyResponse())
					} else {
						if rtsk.response.StatusCode == http.StatusOK {
							if rtsk.onSuccess != nil {
								go rtsk.onSuccess(rtsk)
							}
						} else {
							rtsk.pushError(ThrowErrHttpStatus(rtsk.response.StatusCode))
						}
					}
				}
			case etsk := <-c.ferr:
				{
					if etsk.onError != nil {
						go etsk.onError(etsk, etsk.lsterr)
					} else {
						c.Disconnect()
					}
				}
			case sig := <-c.terminate:
				{
					if sig {
						c.Disconnect()
					}
				}
			case state := <-c.states:
				{
					switch state {
					case STATE_MSGS:
						go c.updateMsgs()
					case STATE_DEVICES:
						go c.updateDevices()
					case STATE_RECORDS:
						go c.updateRecords()
					case STATE_STREAMS:
						go c.updateStreams()
					}
				}
			default:
				time.Sleep(10 * time.Millisecond)
			}
		} else {
			c.stop()
		}
	}
}

func (c *WCClient) doPost(command string, payload []byte) (*http.Request, error) {
	var err error
	var req *http.Request
	var io io.Reader = nil

	if payload != nil {
		io = bytes.NewReader(payload)
	}

	req, err = http.NewRequestWithContext(c.context, "POST", c.cfg.GetUrl(command), io)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (c *WCClient) doUpload(command string, reader io.ReadCloser, size int64, params map[string]string) (*http.Request, error) {
	var err error
	var req *http.Request

	req_url, err := url.Parse(c.cfg.GetUrl(command))
	if err != nil {
		return nil, err
	}
	values := req_url.Query()
	values.Add(JSON_RPC_SHASH, c.GetSID())
	for k, v := range params {
		values.Add(k, v)
	}
	req_url.RawQuery = values.Encode()

	upstr := req_url.String()

	req, err = http.NewRequestWithContext(c.context, "POST", upstr, reader)
	if err != nil {
		return nil, err
	}
	req.ContentLength = size

	return req, nil
}

func (c *WCClient) doGet(command string, params map[string]string) (*http.Request, error) {
	var err error
	var req *http.Request

	req_url, err := url.Parse(c.cfg.GetUrl(command))
	if err != nil {
		return nil, err
	}
	values := req_url.Query()
	values.Add(JSON_RPC_SHASH, c.GetSID())
	for k, v := range params {
		values.Add(k, v)
	}
	req_url.RawQuery = values.Encode()

	req, err = http.NewRequestWithContext(c.context, "POST", req_url.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func errorCommon(tsk *Task, err error) {
	var lsterror string
	if tsk.request == nil {
		lsterror = fmt.Sprintf("Error occured: %v", err)
	} else {
		lsterror = fmt.Sprintf("Error occured - %s: %v", tsk.request.URL, err)
	}
	tsk.client.setLastError(lsterror)
	tsk.client.AddLog(lsterror)
	//
	switch err.(type) {
	default:
		tsk.client.Disconnect()
	case *ErrBadResponse:
		if err.(*ErrBadResponse).code == NO_SUCH_SESSION {
			tsk.client.setClientStatus(StateConnectedWrongSID)
		} else {
			tsk.client.Disconnect()
		}
	}
}

func errorAuth(tsk *Task, err error) {
	errorCommon(tsk, err)
	tsk.client.Disconnect()
}

func (c *WCClient) initJSONRequest() (map[string]any, error) {
	shash := c.GetSID()
	if shash == "" {
		return nil, ThrowErrWrongStatus(StateConnectedWrongSID)
	}

	req := map[string]any{
		JSON_RPC_SHASH: shash,
	}

	return req, nil
}

func (c *WCClient) simpleJSONRequest() ([]byte, error) {
	req, err := c.initJSONRequest()
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (c *WCClient) updateMsgs() error {
	umRequest, err := c.initJSONRequest()
	if err != nil {
		return err
	}

	if lms := c.GetLstMsgStamp(); lms != "" {
		umRequest[JSON_RPC_STAMP] = lms
	}

	b, err := json.Marshal(umRequest)
	if err != nil {
		return err
	}

	var aReqCommand string

	if c.needtosync.GetValue() {
		aReqCommand = "getMsgsAndSync.json"
	} else {
		aReqCommand = "getMsgs.json"
	}

	req, err := c.doPost(aReqCommand, b)
	if err != nil {
		return err
	}

	tsk := &(Task{client: c,
		request:   req,
		onSuccess: successGetMsgs,
		onError:   errorCommon})
	c.wrk <- tsk

	return nil
}

func (c *WCClient) updateStreams() error {
	b, err := c.simpleJSONRequest()
	if err != nil {
		return err
	}

	req, err := c.doPost("getStreams.json", b)
	if err != nil {
		return err
	}

	tsk := &(Task{client: c,
		request:   req,
		onSuccess: successGetStreams,
		onError:   errorCommon})
	c.wrk <- tsk

	return nil
}

func (c *WCClient) updateDevices() error {
	b, err := c.simpleJSONRequest()
	if err != nil {
		return err
	}

	req, err := c.doPost("getDevicesOnline.json", b)
	if err != nil {
		return err
	}

	tsk := &(Task{client: c,
		request:   req,
		onSuccess: successGetDevices,
		onError:   errorCommon})
	c.wrk <- tsk

	return nil
}

func (c *WCClient) updateRecords() error {
	urRequest, err := c.initJSONRequest()
	if err != nil {
		return err
	}

	if lms := c.GetLstRecStamp(); lms != "" {
		urRequest[JSON_RPC_STAMP] = lms
	}

	b, err := json.Marshal(urRequest)
	if err != nil {
		return err
	}

	req, err := c.doPost("getRecordCount.json", b)
	if err != nil {
		return err
	}

	tsk := &(Task{client: c,
		request:   req,
		onSuccess: successGetRecords,
		onError:   errorCommon})
	c.wrk <- tsk

	return nil
}

/* WCClient responses */

func successGetMsgs(tsk *Task) {

	target := make(map[string]any)

	if tsk.successJSONresponse(target) {
		arr, err := getwcObjArray(target, JSON_RPC_MSGS)
		if err != nil {
			tsk.pushError(err)
			return
		}

		tsk.client.lockcbks()
		defer tsk.client.unlockcbks()

		if tsk.client.onSuccessUpdateMsgs != nil {
			tsk.client.onSuccessUpdateMsgs(tsk, arr)
		}
	}
}

func successGetRecords(tsk *Task) {

	target := make(map[string]any)

	if tsk.successJSONresponse(target) {
		arr, err := getwcObjArray(target, JSON_RPC_RECORDS)
		if err != nil {
			tsk.pushError(err)
			return
		}

		tsk.client.lockcbks()
		defer tsk.client.unlockcbks()

		if tsk.client.onSuccessUpdateRecords != nil {
			tsk.client.onSuccessUpdateRecords(tsk, arr)
		}
	}
}

func successGetDevices(tsk *Task) {

	target := make(map[string]any)

	if tsk.successJSONresponse(target) {
		arr, err := getwcObjArray(target, JSON_RPC_DEVICES)
		if err != nil {
			tsk.pushError(err)
			return
		}

		tsk.client.lockcbks()
		defer tsk.client.unlockcbks()

		if tsk.client.onSuccessUpdateDevices != nil {
			tsk.client.onSuccessUpdateDevices(tsk, arr)
		}
	}
}

func successGetStreams(tsk *Task) {

	target := make(map[string]any)

	if tsk.successJSONresponse(target) {
		arr, err := getwcObjArray(target, JSON_RPC_DEVICES)
		if err != nil {
			tsk.pushError(err)
			return
		}

		tsk.client.lockcbks()
		defer tsk.client.unlockcbks()

		if tsk.client.onSuccessUpdateStreams != nil {
			tsk.client.onSuccessUpdateStreams(tsk, arr)
		}
	}
}

func successAuth(tsk *Task) {

	target := make(map[string]any)

	if tsk.successJSONresponse(target) {
		str, err := getwcValue(target, JSON_RPC_SHASH, "", true)
		if err != nil {
			tsk.pushError(err)
			return
		}

		tsk.client.sid.SetValue(str.(string))
		tsk.client.setClientStatus(StateConnected)

		tsk.client.lockcbks()
		defer tsk.client.unlockcbks()

		if tsk.client.onSIDSetted != nil {
			tsk.client.onSIDSetted(tsk.client, tsk.client.GetSID())
		}

		if tsk.client.onSuccessAuth != nil {
			tsk.client.onSuccessAuth(tsk)
		}
	}
}

func successSaveRecord(tsk *Task) {
	target := make(map[string]any)

	if tsk.successJSONresponse(target) {
		tsk.client.lockcbks()
		defer tsk.client.unlockcbks()

		if tsk.client.onSuccessSaveRecord != nil {
			tsk.client.onSuccessSaveRecord(tsk)
		}
	}
}

func successReqRecordMeta(tsk *Task) {
	target := make(map[string]any)

	if tsk.successJSONresponse(target) {
		tsk.client.lockcbks()
		defer tsk.client.unlockcbks()

		if tsk.client.onSuccessRequestRecordMeta != nil {
			tsk.client.onSuccessRequestRecordMeta(tsk, target)
		}
	}
}

func successReqRecordData(tsk *Task) {
	defer tsk.response.Body.Close()

	const BUF_SIZE = 4096

	data := make([]byte, 0, BUF_SIZE)
	buf := make([]byte, BUF_SIZE)

	for true {
		n, err := tsk.response.Body.Read(buf)
		if err != nil {
			tsk.pushError(err)
			return
		}

		if n > 0 {
			data = append(data, buf[0:n]...)
		}

		if n < BUF_SIZE {
			break
		}
	}

	if tsk.client.onSuccessRequestRecord != nil {
		tsk.client.onSuccessRequestRecord(tsk, data)
	}
}

/* WCClient public methods */

/* WCClient callbacks */

/*
Set new callback for the "Successful authorization" event.

	`event` is the reference to the callback function
*/
func (c *WCClient) SetOnAuthSuccess(event TaskNotifyFunc) {
	c.lockcbks()
	defer c.unlockcbks()
	c.onSuccessAuth = event
}

/*
Set new callback for the "The connection state has been changed" event.

	`event` is the reference to the callback function
*/
func (c *WCClient) SetOnConnected(event ConnNotifyEventFunc) {
	c.lockcbks()
	defer c.unlockcbks()
	c.onConnected = event
}

/*
Set new callback for the "Client has been disconnected" event.

	`event` is the reference to the callback function
*/
func (c *WCClient) SetOnDisconnect(event NotifyEventFunc) {
	c.lockcbks()
	defer c.unlockcbks()
	c.onDisconnect = event
}

/*
Set new callback for the "The session id has been changed" event.

	`event` is the reference to the callback function
*/
func (c *WCClient) SetSIDSetted(event StringNotifyFunc) {
	c.lockcbks()
	defer c.unlockcbks()
	c.onSIDSetted = event
}

/*
Set new callback for the "Added new log entry" event.

	`event` is the reference to the callback function
*/
func (c *WCClient) SetOnAddLog(event StringNotifyFunc) {
	c.lockcbks()
	defer c.unlockcbks()
	c.onAddLog = event
}

/*
Set new callback for the "The request to update list of media records has been completed.
The response has arrived." event.

	`event` is the reference to the callback function
	`jsonresult` inside JSONArrayNotifyEventFunc will contain reference to the array of
	the media records (with no data. to get the data of media record by its id use
	the GetRecordData/GetRecordMeta methods)
*/
func (c *WCClient) SetOnUpdateRecords(event JSONArrayNotifyEventFunc) {
	c.lockcbks()
	defer c.unlockcbks()
	c.onSuccessUpdateRecords = event
}

/*
Set new callback for the "The request to update list of online devices has been completed.
The response has arrived." event.

	`event` is the reference to the callback function
	`jsonresult` inside JSONArrayNotifyEventFunc will contain reference to the array of
	the online devices
*/
func (c *WCClient) SetOnUpdateDevices(event JSONArrayNotifyEventFunc) {
	c.lockcbks()
	defer c.unlockcbks()
	c.onSuccessUpdateDevices = event
}

/*
Set new callback for the "The request to update list of streaming devices has been completed
The response has arrived." event.

	`event` is the reference to the callback function
	`jsonresult` inside JSONArrayNotifyEventFunc will contain reference to the array of
	the streaming devices
*/
func (c *WCClient) SetOnUpdateStreams(event JSONArrayNotifyEventFunc) {
	c.lockcbks()
	defer c.unlockcbks()
	c.onSuccessUpdateStreams = event
}

/*
Set new callback for the "The request to update list of messages has been completed.
The response has arrived." event.

	`event` is the reference to the callback function
	`jsonresult` inside JSONArrayNotifyEventFunc will contain reference to the array of
	the incoming messages
*/
func (c *WCClient) SetOnUpdateMsgs(event JSONArrayNotifyEventFunc) {
	c.lockcbks()
	defer c.unlockcbks()
	c.onSuccessUpdateMsgs = event
}

func (c *WCClient) SetOnSuccessSaveRecord(event TaskNotifyFunc) {
	c.lockcbks()
	defer c.unlockcbks()
	c.onSuccessSaveRecord = event
}

func (c *WCClient) SetOnReqRecordMeta(event JSONNotifyEventFunc) {
	c.lockcbks()
	defer c.unlockcbks()
	c.onSuccessRequestRecordMeta = event
}

func (c *WCClient) SetOnReqRecordData(event DataNotifyEventFunc) {
	c.lockcbks()
	defer c.unlockcbks()
	c.onSuccessRequestRecord = event
}

/* WCClient states */

// Is the client's working thread running
func (c *WCClient) Working() bool {
	switch c.clientst.getValue() {
	case StateWaiting, StateDisconnected:
		return false
	default:
		return true
	}
}

// Get the current session id for the client
func (c *WCClient) GetSID() string {
	return c.sid.GetValue()
}

// Get the current status for the client
func (c *WCClient) GetClientStatus() ClientStatus {
	return c.clientst.getValue()
}

func (c *WCClient) IsClientStatusInRange(st []ClientStatus) bool {
	curstatus := c.GetClientStatus()
	for _, v := range st {
		if v == curstatus {
			return true
		}
	}

	return false
}

// Get the last occured error string description for the client
func (c *WCClient) LastError() string {
	return c.lstError.GetValue()
}

// The timestamp of the last received message
func (c *WCClient) GetLstMsgStamp() string {
	return c.lmsgstamp.GetValue()
}

// The timestamp of the last received media record
func (c *WCClient) GetLstRecStamp() string {
	return c.lrecstamp.GetValue()
}

/*
Launch client.

	The function initializes and starts the client''s working thread.
	After calling this method the assigned client configuaration will be locked

	`return` nil on success or error object.
*/
func (c *WCClient) Start() error {
	switch st := c.GetClientStatus(); st {
	case StateWaiting:
		c.start()

		go c.internalStart()

		return nil
	default:
		return ThrowErrWrongStatus(st)
	}
}

/*
Launch request to authorize the client on the server host.

	See protocol request `authorize`. Username and password are parsed from the
	host URL (\sa SetHostURL)

	`return` nil on success or the error object.
*/
func (c *WCClient) AuthFromHostUrl() error {
	l := c.cfg.hosturl.User.Username()
	p, b := c.cfg.hosturl.User.Password()
	if len(l) == 0 || !b {
		return ThrowErrWrongAuthData()
	}
	return c.Auth(l, p)
}

/*
Launch request to authorize the client on the server host.

	See protocol request `authorize`. If the specified `aLogin` or `aPwrd` are empty
	strings, the client tries to connect to the host using the username section
	from the host URL (\sa SetHostURL)

	`aLogin` is the name of the user on the server.
	`aPwrd` is the password of the user on the server.
	`return` nil on success or the error object.
*/
func (c *WCClient) Auth(aLogin, aPwrd string) error {
	if len(aLogin) == 0 {
		if len(aPwrd) > 0 {
			c.cfg.hosturl.User = url.UserPassword(c.cfg.hosturl.User.Username(), aPwrd)
		}
		return c.AuthFromHostUrl()
	}

	if len(aLogin) == 0 || len(aPwrd) == 0 {
		return ThrowErrWrongAuthData()
	}

	authRequest := map[string]string{
		JSON_RPC_NAME:   aLogin,
		JSON_RPC_PASS:   aPwrd,
		JSON_RPC_DEVICE: c.cfg.device,
		JSON_RPC_META:   c.cfg.meta,
	}
	b, err := json.Marshal(authRequest)
	if err != nil {
		return err
	}

	req, err := c.doPost("authorize.json", b)
	if err != nil {
		return err
	}

	tsk := &(Task{client: c,
		request:   req,
		onSuccess: successAuth,
		onError:   errorAuth})

	c.setClientStatus(StateConnectedAuthorization)

	c.wrk <- tsk

	return nil
}

/*
Disconnect client from the server host.

	`return` nil on success or error object.
*/
func (c *WCClient) Disconnect() error {
	switch st := c.GetClientStatus(); st {
	case StateDisconnected, StateWaiting:
		return ThrowErrWrongStatus(st)
	}

	if c.cancelfunc != nil {
		c.cancelfunc()
	}
	c.stop()

	return nil
}

// Add the new string to the message log
func (c *WCClient) AddLog(aStr string) {
	c.log.PushBack(aStr)

	c.lockcbks()
	defer c.unlockcbks()

	if c.onAddLog != nil {
		c.onAddLog(c, aStr)
	}
}

// Get the client”s log
func (c *WCClient) GetLog() *StringListThreadSafe {
	return c.log
}

// Clear the client”s log
func (c *WCClient) ClearLog() {
	c.log.Clear()
}

// Clear the client”s last error string
func (c *WCClient) ClearError() {
	c.lstError.Clear()
}

// Set is the next update of the message list will
// occur with or without the sending of a 'sync' message
// (sa. getMsgsAndSync - WCPD)
func (c *WCClient) SetNeedToSync(val bool) {
	c.needtosync.SetValue(val)
}

func (c *WCClient) InvalidateState(aStateId ClientState) error {
	var err error
	switch aStateId {
	case STATE_LOG:
		c.ClearLog()
	case STATE_ERROR:
		c.ClearError()
	case STATE_STREAMS:
		err = c.UpdateStreams()
	case STATE_DEVICES:
		err = c.UpdateDevices()
	case STATE_RECORDS:
		err = c.UpdateRecords()
	case STATE_MSGS:
		err = c.UpdateMsgs()
	case STATE_SENDWITHSYNC:
		c.needtosync.SetValue(false)
	case STATE_MSGSSTAMP:
		c.lmsgstamp.Clear()
	case STATE_RECORDSSTAMP:
		c.lrecstamp.Clear()
	default:
		err = ThrowErrParam(aStateId)
	}
	return err
}

func (c *WCClient) UpdateStreams() error {
	if st := c.GetClientStatus(); st != StateConnected {
		return ThrowErrWrongStatus(st)
	}

	c.states <- STATE_STREAMS

	return nil
}

func (c *WCClient) UpdateDevices() error {
	if st := c.GetClientStatus(); st != StateConnected {
		return ThrowErrWrongStatus(st)
	}

	c.states <- STATE_DEVICES

	return nil
}

func (c *WCClient) UpdateRecords() error {
	if st := c.GetClientStatus(); st != StateConnected {
		return ThrowErrWrongStatus(st)
	}

	c.states <- STATE_RECORDS

	return nil
}

func (c *WCClient) UpdateMsgs() error {
	if st := c.GetClientStatus(); st != StateConnected {
		return ThrowErrWrongStatus(st)
	}

	c.states <- STATE_MSGS

	return nil
}

func (c *WCClient) SaveRecord(aBuf io.ReadCloser, aBufSize int64, meta string, userdata any) error {
	if st := c.GetClientStatus(); st != StateConnected {
		return ThrowErrWrongStatus(st)
	}

	params := map[string]string{
		JSON_RPC_META: meta,
	}

	req, err := c.doUpload("addRecord.json", aBuf, aBufSize, params)
	if err != nil {
		return err
	}

	tsk := &(Task{client: c,
		request:   req,
		userdata:  userdata,
		onSuccess: successSaveRecord,
		onError:   errorCommon})

	c.wrk <- tsk

	return nil
}

func (c *WCClient) RequestRecordMeta(rid int, userdata any) error {
	if st := c.GetClientStatus(); st != StateConnected {
		return ThrowErrWrongStatus(st)
	}

	rmRequest, err := c.initJSONRequest()
	if err != nil {
		return err
	}
	rmRequest[JSON_RPC_RID] = float64(rid)

	b, err := json.Marshal(rmRequest)
	if err != nil {
		return err
	}

	req, err := c.doPost("getRecordMeta.json", b)
	if err != nil {
		return err
	}

	tsk := &(Task{client: c,
		request:   req,
		userdata:  userdata,
		onSuccess: successReqRecordMeta,
		onError:   errorCommon})

	c.wrk <- tsk

	return nil
}

func (c *WCClient) RequestRecord(rid int, userdata any) error {
	if st := c.GetClientStatus(); st != StateConnected {
		return ThrowErrWrongStatus(st)
	}

	params := map[string]string{
		JSON_RPC_RID: strconv.FormatInt(int64(rid), 10),
	}

	req, err := c.doGet("getRecordData.json", params)
	if err != nil {
		return err
	}

	tsk := &(Task{client: c,
		request:   req,
		userdata:  userdata,
		onSuccess: successReqRecordData,
		onError:   errorCommon})

	c.wrk <- tsk

	return nil
}

/*
func (c *WCClient) LaunchOutStream(aSubProto string, aDelta int) bool {

}

func (c *WCClient) LaunchInStream(aDeviceName string) bool {

}

func (c *WCClient) GetConfig() {

}

func (c *WCClient) SetConfig(aStr string) {

}

func (c *WCClient) DeleteRecords(aIndices []int) {

}

func (c *WCClient) SendMsg(aMsg any) {

}

func (c *WCClient) RequestRecord(rid int) {

}

func (c *WCClient) SaveAsSnapshot(aBuf any) {

}

func (c *WCClient) StopStreaming() {

}
*/
