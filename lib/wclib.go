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

type MediaStruct struct {
	Device string  `json:"device"`
	Rid    float64 `json:"rid"`
	Stamp  string  `json:"stamp"`
}

type MediaMetaStruct struct {
	Device string `json:"device"`
	Meta   string `json:"meta"`
	Stamp  string `json:"stamp"`
}

type MessageStruct struct {
	Device string         `json:"device"`
	Msg    string         `json:"msg"`
	Stamp  string         `json:"stamp"`
	Params map[string]any `json:"params"`
}

type OutMessageStruct struct {
	Target string         `json:"target"`
	Msg    string         `json:"msg"`
	Params map[string]any `json:"params"`
}

type DeviceStruct struct {
	Device string `json:"device"`
	Meta   string `json:"meta"`
}

type StreamStruct struct {
	Device   string  `json:"device"`
	SubProto string  `json:"subproto"`
	Delta    float64 `json:"delta"`
}

/*
You can use JSONHelper methods to convert json maps to structs.
The methods are not efficient and applyable only for testing proposes.
You can use other external suitable modules to work with JSON maps and
convert to structs vise versa
*/
func JSONDecode(jsonmap map[string]any, dest any) error {
	jsonbody, err := json.Marshal(jsonmap)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(jsonbody, dest); err != nil {
		return err
	}
	return nil
}

type jsonField struct {
	name string
	tp   reflect.Kind
}

func (mr *MediaStruct) JSONDecode(jsonmap map[string]any) error {
	decl := []jsonField{
		{name: JSON_RPC_DEVICE, tp: reflect.String},
		{name: JSON_RPC_RID, tp: reflect.Float64},
		{name: JSON_RPC_STAMP, tp: reflect.String},
	}

	for num, v := range decl {
		_any, ok := jsonmap[v.name]
		if ok {
			dt := reflect.TypeOf(_any).Kind()
			if dt == v.tp {
				switch num {
				case 0:
					mr.Device, _ = _any.(string)
				case 1:
					mr.Rid, _ = _any.(float64)
				case 2:
					mr.Stamp, _ = _any.(string)
				}
			} else {
				return ThrowErrMalformedResponse(EMKWrongType, v.name, fmt.Sprintf("%v", v.tp))
			}
		} else {
			return ThrowErrMalformedResponse(EMKFieldExpected, v.name, nil)
		}
	}

	return nil
}

func (mr *MediaMetaStruct) JSONDecode(jsonmap map[string]any) error {
	decl := []jsonField{
		{name: JSON_RPC_DEVICE, tp: reflect.String},
		{name: JSON_RPC_META, tp: reflect.String},
		{name: JSON_RPC_STAMP, tp: reflect.String},
	}

	for num, v := range decl {
		_any, ok := jsonmap[v.name]
		if ok {
			dt := reflect.TypeOf(_any).Kind()
			if dt == v.tp {
				switch num {
				case 0:
					mr.Device, _ = _any.(string)
				case 1:
					mr.Meta, _ = _any.(string)
				case 2:
					mr.Stamp, _ = _any.(string)
				}
			} else {
				return ThrowErrMalformedResponse(EMKWrongType, v.name, fmt.Sprintf("%v", v.tp))
			}
		} else {
			return ThrowErrMalformedResponse(EMKFieldExpected, v.name, nil)
		}
	}

	return nil
}

func (mr *MessageStruct) JSONDecode(jsonmap map[string]any) error {
	decl := []jsonField{
		{name: JSON_RPC_DEVICE, tp: reflect.String},
		{name: JSON_RPC_MSG, tp: reflect.String},
		{name: JSON_RPC_STAMP, tp: reflect.String},
		{name: JSON_RPC_PARAMS, tp: reflect.Map},
	}

	for num, v := range decl {
		_any, ok := jsonmap[v.name]
		if ok {
			dt := reflect.TypeOf(_any).Kind()
			if dt == v.tp {
				switch num {
				case 0:
					mr.Device, _ = _any.(string)
				case 1:
					mr.Msg, _ = _any.(string)
				case 2:
					mr.Stamp, _ = _any.(string)
				case 3:
					mr.Params, _ = _any.(map[string]any)
				}
			} else {
				return ThrowErrMalformedResponse(EMKWrongType, v.name, fmt.Sprintf("%v", v.tp))
			}
		} else {
			return ThrowErrMalformedResponse(EMKFieldExpected, v.name, nil)
		}
	}

	return nil
}

func (mr *DeviceStruct) JSONDecode(jsonmap map[string]any) error {
	decl := []jsonField{
		{name: JSON_RPC_DEVICE, tp: reflect.String},
		{name: JSON_RPC_META, tp: reflect.String},
	}

	for num, v := range decl {
		_any, ok := jsonmap[v.name]
		if ok {
			dt := reflect.TypeOf(_any).Kind()
			if dt == v.tp {
				switch num {
				case 0:
					mr.Device, _ = _any.(string)
				case 1:
					mr.Meta, _ = _any.(string)
				}
			} else {
				return ThrowErrMalformedResponse(EMKWrongType, v.name, fmt.Sprintf("%v", v.tp))
			}
		} else {
			return ThrowErrMalformedResponse(EMKFieldExpected, v.name, nil)
		}
	}

	return nil
}

func (mr *StreamStruct) JSONDecode(jsonmap map[string]any) error {
	decl := []jsonField{
		{name: JSON_RPC_DEVICE, tp: reflect.String},
		{name: JSON_RPC_SUBPROTO, tp: reflect.String},
		{name: JSON_RPC_DELTA, tp: reflect.Float64},
	}

	for num, v := range decl {
		_any, ok := jsonmap[v.name]
		if ok {
			dt := reflect.TypeOf(_any).Kind()
			if dt == v.tp {
				switch num {
				case 0:
					mr.Device, _ = _any.(string)
				case 1:
					mr.SubProto, _ = _any.(string)
				case 2:
					mr.Delta, _ = _any.(float64)
				}
			} else {
				return ThrowErrMalformedResponse(EMKWrongType, v.name, fmt.Sprintf("%v", v.tp))
			}
		} else {
			return ThrowErrMalformedResponse(EMKFieldExpected, v.name, nil)
		}
	}

	return nil
}

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

// The size of frame header (6 bytes)
const WC_STREAM_FRAME_HEADER_SIZE uint16 = 6

// The frame header sequence
const WC_FRAME_START_SEQ uint16 = 0xaaaa

type EmptyNotifyFunc func(client *WCClient)
type NotifyEventFunc func(client *WCClient, data any)
type TaskNotifyFunc func(tsk ITask)
type ConnNotifyEventFunc func(client *WCClient, status ClientStatus)
type StringNotifyFunc func(client *WCClient, value string)
type DataNotifyEventFunc func(tsk ITask, data []byte)
type JSONArrayNotifyEventFunc func(tsk ITask, jsonresult []any)
type JSONNotifyEventFunc func(tsk ITask, jsonresult any)

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

/* WCClientConfig */

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

/* StringThreadSafe */

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

/* BoolThreadSafe */

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

/* FramesListThreadSafe */

type FramesListThreadSafe struct {
	mux sync.Mutex

	value *list.List
}

func (c *FramesListThreadSafe) PushBack(str *bytes.Buffer) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.value.PushBack(str)
}

func (c *FramesListThreadSafe) NotEmpty() bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.value.Len() > 0
}

func (c *FramesListThreadSafe) Pop() *bytes.Buffer {
	c.mux.Lock()
	defer c.mux.Unlock()

	el := c.value.Front()
	if el != nil {
		c.value.Remove(el)
		return el.Value.(*bytes.Buffer)
	} else {
		return nil
	}
}

func (c *FramesListThreadSafe) Clear() {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.value = list.New()
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

/* WCClient */

type WCClient struct {
	//@private
	cbmux  sync.Mutex
	strmux sync.Mutex

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
	wrk       chan ITask
	finished  chan ITask
	states    chan ClientState
	ferr      chan ITask
	terminate chan bool

	/* state */
	clientst   *ClientStatusThreadSafe
	lstError   *StringThreadSafe
	sid        *StringThreadSafe
	lmsgstamp  *StringThreadSafe
	lrecstamp  *StringThreadSafe
	needtosync *BoolThreadSafe
	log        *StringListThreadSafe
	outstream  *OutStream

	/* config */
	cfg *WCClientConfig

	/* http */
	inclient   *http.Client
	context    context.Context
	cancelfunc context.CancelFunc
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

/* Task */

type TaskKind int

const (
	TaskDefault TaskKind = iota
	TaskInputStream
	TaskOutputStream
)

type taskSuccessFunc func(tsk ITask)
type taskErrorFunc func(tsk ITask)

type ITask interface {
	execute(after chan ITask)
	pushError(err error)
	getRequest() *http.Request
	getResponse() *http.Response
	getOnSuccess() taskSuccessFunc
	getOnError() taskErrorFunc

	GetClient() *WCClient
	GetUserData() any
	SetUserData(data any)
	GetKind() TaskKind
	GetLastError() error
}

type Task struct {
	//@private
	client   *WCClient
	request  *http.Request
	response *http.Response

	userdata any
	kind     TaskKind

	lsterr    error
	onSuccess taskSuccessFunc
	onError   taskErrorFunc
}

/* Task private methods */

func (tsk *Task) getOnSuccess() taskSuccessFunc {
	return tsk.onSuccess
}

func (tsk *Task) getOnError() taskErrorFunc {
	return tsk.onError
}

func (tsk *Task) getResponse() *http.Response {
	return tsk.response
}

func (tsk *Task) getRequest() *http.Request {
	return tsk.request
}

func (tsk *Task) pushError(err error) {
	tsk.lsterr = err
	tsk.client.ferr <- tsk
}

func (tsk *Task) execute(after chan ITask) {
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
		}

		return def, nil
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

func (tsk *Task) GetKind() TaskKind {
	return tsk.kind
}

func (tsk *Task) GetLastError() error {
	return tsk.lsterr
}

/* OutStream */

type OutStream struct {
	current_frame *bytes.Buffer
	outframes     *FramesListThreadSafe
	lsterror      error
	terminate     chan bool
}

func (c *OutStream) Read(b []byte) (n int, err error) {
	for working := true; working; {
		select {
		case <-c.terminate:
			{
				working = false
			}
		default:
			{
				if c.current_frame != nil {
					var n int = 0
					n, c.lsterror = c.current_frame.Read(b)
					if n == 0 {
						if (c.lsterror == nil) || (c.lsterror == io.EOF) {
							c.current_frame = nil
							continue
						} else {
							return 0, c.lsterror
						}
					} else {
						return n, c.lsterror
					}
				} else if c.outframes.NotEmpty() {
					c.current_frame = c.outframes.Pop()
				}
			}
		}
	}
	return 0, io.EOF
}

/* WCClientConfig constructor */

// Create new empty client configuration
func ClientCfgNew() *WCClientConfig {
	return &(WCClientConfig{locked: false})
}

/* WCClientConfig public methods */

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

	wcclient := &(WCClient{wrk: make(chan ITask, 16),
		terminate:  make(chan bool, 2),
		finished:   make(chan ITask, 16),
		ferr:       make(chan ITask, 16),
		states:     make(chan ClientState, 32),
		needtosync: &BoolThreadSafe{},
		lmsgstamp:  &StringThreadSafe{},
		lrecstamp:  &StringThreadSafe{},
		lstError:   &StringThreadSafe{},
		sid:        &StringThreadSafe{},
		clientst:   &ClientStatusThreadSafe{value: StateWaiting},
		inclient:   client,
		log:        &StringListThreadSafe{value: list.New()},
		outstream:  nil,
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

func (c *WCClient) lockstrs() {
	c.strmux.Lock()
}

func (c *WCClient) unlockstrs() {
	c.strmux.Unlock()
}

func (c *WCClient) stop() {
	c.setClientStatus(StateDisconnected)
}

func (c *WCClient) start() {
	c.setClientStatus(StateConnectedWrongSID)
}

func (c *WCClient) stopOutStream() {
	c.lockstrs()
	defer c.unlockstrs()
	if c.outstream != nil {
		c.outstream.terminate <- true
		c.outstream = nil
	}
}

func (c *WCClient) startOutStream() {
	c.lockstrs()
	defer c.unlockstrs()

	c.outstream = &(OutStream{
		outframes: &FramesListThreadSafe{value: list.New()},
		lsterror:  nil,
		terminate: make(chan bool, 2),
	})
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
					if rtsk.getResponse() == nil {
						rtsk.pushError(ThrowErrEmptyResponse())
					} else {
						if rtsk.getResponse().StatusCode == http.StatusOK {
							internalOnSuccess(rtsk)
						} else {
							rtsk.pushError(ThrowErrHttpStatus(rtsk.getResponse().StatusCode))
						}
					}
				}
			case etsk := <-c.ferr:
				{
					if internalOnError(etsk) {
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
	req.ContentLength = int64(len(payload))

	return req, nil
}

func (c *WCClient) doPutGet(method string, command string, reader io.Reader, size int64, params map[string]string) (*http.Request, error) {
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

	req, err = http.NewRequestWithContext(c.context, method, req_url.String(), reader)
	if err != nil {
		return nil, err
	}
	if req.ContentLength == 0 && size > 0 {
		req.ContentLength = size
	}

	return req, nil
}

func (c *WCClient) doDownload(command string, reader io.Reader, size int64, params map[string]string) (*http.Request, error) {
	return c.doPutGet("POST", command, reader, size, params)
}

func (c *WCClient) doGet(command string, params map[string]string) (*http.Request, error) {
	return c.doPutGet("POST", command, nil, 0, params)
}

func (c *WCClient) doUpload(command string, params map[string]string) (*http.Request, error) {
	req, err := c.doPutGet("PUT", command, nil, 0, params)
	if err != nil {
		return nil, err
	} else {
		c.lockstrs()
		defer c.unlockstrs()
		snapshot := *c.outstream
		req.Body = io.NopCloser(c.outstream)
		req.GetBody = func() (io.ReadCloser, error) {
			r := snapshot
			return io.NopCloser(&r), nil
		}
		req.ContentLength = 0x500000000
		return req, nil
	}
}

func internalOnSuccess(self ITask) {
	if self.getOnSuccess() != nil {
		go self.getOnSuccess()(self)
	}
}

func internalOnError(self ITask) bool {
	if self.getOnError() != nil {
		go self.getOnError()(self)
		return false
	}
	return true
}

func errorCommon(tsk ITask) {
	err := tsk.GetLastError()
	var lsterror string
	if tsk.getRequest() == nil {
		lsterror = fmt.Sprintf("Error occurred: %v", err)
	} else {
		lsterror = fmt.Sprintf("Error occurred - %s: %v", tsk.getRequest().URL, err)
	}
	tsk.GetClient().setLastError(lsterror)
	tsk.GetClient().AddLog(lsterror)
	//
	switch err.(type) {
	default:
		tsk.GetClient().Disconnect()
	case *ErrBadResponse:
		if err.(*ErrBadResponse).code == NO_SUCH_SESSION {
			tsk.GetClient().setClientStatus(StateConnectedWrongSID)
		} else {
			tsk.GetClient().Disconnect()
		}
	}
}

func errorAuth(tsk ITask) {
	errorCommon(tsk)
	tsk.GetClient().Disconnect()
}

func errorIOCommon(tsk ITask) {
	errorCommon(tsk)
	switch tsk.GetKind() {
	case TaskOutputStream:
		{
			tsk.GetClient().stopOutStream()
		}
	}
	if tsk.GetClient().onSuccessIOStream != nil {
		tsk.GetClient().onSuccessIOStream(tsk)
	}
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

func successGetMsgs(tsk ITask) {

	target := make(map[string]any)

	if tsk.(*Task).successJSONresponse(target) {
		arr, err := getwcObjArray(target, JSON_RPC_MSGS)
		if err != nil {
			tsk.pushError(err)
			return
		}

		tsk.GetClient().lockcbks()
		defer tsk.GetClient().unlockcbks()

		if tsk.GetClient().onSuccessUpdateMsgs != nil {
			tsk.GetClient().onSuccessUpdateMsgs(tsk, arr)
		}
	}
}

func successGetRecords(tsk ITask) {

	target := make(map[string]any)

	if tsk.(*Task).successJSONresponse(target) {
		arr, err := getwcObjArray(target, JSON_RPC_RECORDS)
		if err != nil {
			tsk.pushError(err)
			return
		}

		tsk.GetClient().lockcbks()
		defer tsk.GetClient().unlockcbks()

		if tsk.GetClient().onSuccessUpdateRecords != nil {
			tsk.GetClient().onSuccessUpdateRecords(tsk, arr)
		}
	}
}

func successGetDevices(tsk ITask) {

	target := make(map[string]any)

	if tsk.(*Task).successJSONresponse(target) {
		arr, err := getwcObjArray(target, JSON_RPC_DEVICES)
		if err != nil {
			tsk.pushError(err)
			return
		}

		tsk.GetClient().lockcbks()
		defer tsk.GetClient().unlockcbks()

		if tsk.GetClient().onSuccessUpdateDevices != nil {
			tsk.GetClient().onSuccessUpdateDevices(tsk, arr)
		}
	}
}

func successGetStreams(tsk ITask) {

	target := make(map[string]any)

	if tsk.(*Task).successJSONresponse(target) {
		arr, err := getwcObjArray(target, JSON_RPC_DEVICES)
		if err != nil {
			tsk.pushError(err)
			return
		}

		tsk.GetClient().lockcbks()
		defer tsk.GetClient().unlockcbks()

		if tsk.GetClient().onSuccessUpdateStreams != nil {
			tsk.GetClient().onSuccessUpdateStreams(tsk, arr)
		}
	}
}

func successAuth(tsk ITask) {

	target := make(map[string]any)

	if tsk.(*Task).successJSONresponse(target) {
		str, err := getwcValue(target, JSON_RPC_SHASH, "", true)
		if err != nil {
			tsk.pushError(err)
			return
		}

		tsk.GetClient().sid.SetValue(str.(string))
		tsk.GetClient().setClientStatus(StateConnected)

		tsk.GetClient().lockcbks()
		defer tsk.GetClient().unlockcbks()

		if tsk.GetClient().onSIDSetted != nil {
			tsk.GetClient().onSIDSetted(tsk.GetClient(), tsk.GetClient().GetSID())
		}

		if tsk.GetClient().onSuccessAuth != nil {
			tsk.GetClient().onSuccessAuth(tsk)
		}
	}
}

func successSaveRecord(tsk ITask) {
	target := make(map[string]any)

	if tsk.(*Task).successJSONresponse(target) {
		tsk.GetClient().lockcbks()
		defer tsk.GetClient().unlockcbks()

		if tsk.GetClient().onSuccessSaveRecord != nil {
			tsk.GetClient().onSuccessSaveRecord(tsk)
		}
	}
}

func successReqRecordMeta(tsk ITask) {
	target := make(map[string]any)

	if tsk.(*Task).successJSONresponse(target) {
		tsk.GetClient().lockcbks()
		defer tsk.GetClient().unlockcbks()

		if tsk.GetClient().onSuccessRequestRecordMeta != nil {
			tsk.GetClient().onSuccessRequestRecordMeta(tsk, target)
		}
	}
}

func successReqRecordData(tsk ITask) {
	defer tsk.getResponse().Body.Close()

	const BUF_SIZE = 4096

	data := make([]byte, 0, BUF_SIZE)
	buf := make([]byte, BUF_SIZE)

	for true {
		n, err := tsk.getResponse().Body.Read(buf)
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

	if tsk.GetClient().onSuccessRequestRecord != nil {
		tsk.GetClient().onSuccessRequestRecord(tsk, data)
	}
}

func successIOFinished(tsk ITask) {
	defer tsk.getResponse().Body.Close()

	switch tsk.GetKind() {
	case TaskOutputStream:
		{
			tsk.GetClient().stopOutStream()
		}
	}

	if tsk.GetClient().onSuccessIOStream != nil {
		tsk.GetClient().onSuccessIOStream(tsk)
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

func (c *WCClient) SetOnAfterLaunchOutStream(event TaskNotifyFunc) {
	c.lockcbks()
	defer c.unlockcbks()
	c.onAfterLaunchOutStream = event
}

func (c *WCClient) SetOnSuccessIOStream(event TaskNotifyFunc) {
	c.lockcbks()
	defer c.unlockcbks()
	c.onSuccessIOStream = event
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

	c.StopStreaming()

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
	case STATE_STREAMING:
		c.StopStreaming()
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
		if aBuf != nil {
			aBuf.Close()
		}
		return ThrowErrWrongStatus(st)
	}

	params := map[string]string{
		JSON_RPC_META: meta,
	}

	req, err := c.doDownload("addRecord.json", aBuf, aBufSize, params)
	if err != nil {
		if aBuf != nil {
			aBuf.Close()
		}
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

func (c *WCClient) LaunchOutStream(aSubProto string, aDelta int, userdata any) error {
	if st := c.GetClientStatus(); st != StateConnected {
		return ThrowErrWrongStatus(st)
	}

	params := map[string]string{
		JSON_RPC_SUBPROTO: aSubProto,
		JSON_RPC_DELTA:    strconv.FormatInt(int64(aDelta), 10),
	}

	c.startOutStream()

	req, err := c.doUpload("input.raw", params)
	if err != nil {
		return err
	}

	tsk := &(Task{
		client:    c,
		kind:      TaskOutputStream,
		request:   req,
		userdata:  userdata,
		onSuccess: successIOFinished,
		onError:   errorIOCommon})

	c.wrk <- tsk

	if c.onAfterLaunchOutStream != nil {
		c.onAfterLaunchOutStream(tsk)
	}

	return nil
}

func (c *WCClient) PushOutData(data *bytes.Buffer) error {
	c.lockstrs()
	defer c.unlockstrs()
	if c.outstream != nil {
		c.outstream.outframes.PushBack(data)
	}
	return nil
}

func (c *WCClient) StopStreaming() {
	c.stopOutStream()
	// c.stopInStream()
}

/*
func (c *WCClient) LaunchInStream(aDeviceName string) error {

}

func (c *WCClient) GetConfig() error {

}

func (c *WCClient) SetConfig(aStr string) error {

}

func (c *WCClient) DeleteRecords(aIndices []int) error {

}

func (c *WCClient) SendMsg(aMsg any) error {

}

*/
