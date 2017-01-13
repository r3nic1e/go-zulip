package zulip

import "net/http"
import "net/url"
import "io/ioutil"
import "fmt"
import "encoding/json"
import "strconv"
import "strings"
import "time"
import "github.com/davecgh/go-spew/spew"

type EventListener interface {
	HandleEvent(EventResponse) bool
}

type Zulip struct {
	authLogin, authPass string
	baseUrl             string
	queueID             string
	Debug               bool
}

func NewZulipApi(baseUrl string) *Zulip {
	return &Zulip{baseUrl: baseUrl}
}

func (z *Zulip) SetBasicAuth(login, pass string) {
	z.authLogin = login
	z.authPass = pass
}

func (z *Zulip) tryToCallApi(url, method string, params url.Values) []byte {
	client := &http.Client{}

	url = fmt.Sprintf("%s/%s?%s", z.baseUrl, url, params.Encode())
	if z.Debug {
		fmt.Println(url)
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return []byte{}
	}
	req.SetBasicAuth(z.authLogin, z.authPass)

	resp, err := client.Do(req)
	if err != nil {
		return []byte{}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}
	}

	return body
}

func (z *Zulip) api(url, method string, params url.Values) (bytes []byte, err error) {
	for i := 0; i <= 5; i++ {
		bytes = z.tryToCallApi(url, method, params)

		var res BaseResponse
		err = json.Unmarshal(bytes, &res)
		if err != nil {
			continue
		}
		if z.Debug {
			spew.Dump(res)
		}

		if res.Result == "error" {
			if strings.HasPrefix(res.Msg, "API usage exceeded rate limit") {
				if z.Debug {
					fmt.Println("Exceeded API rate limit, sleeping for 1 second")
				}
				time.Sleep(time.Second)
				continue
			}
		}
		return
	}
	return
}

func (z *Zulip) Register(event_types []string) string {
	v := url.Values{}
	json_types, _ := json.Marshal(event_types)
	v.Set("event_types", string(json_types))

	bytes, err := z.api("api/v1/register", "POST", v)
	if err != nil {
		panic(err)
	}

	var res RegisterResponse
	err = json.Unmarshal(bytes, &res)
	if err != nil {
		panic(err)
	}

	z.queueID = res.Queue_id
	return res.Queue_id
}

func (z *Zulip) tryToGetEvents(last_event_id string) []byte {
	v := url.Values{}
	v.Set("queue_id", z.queueID)
	v.Set("last_event_id", last_event_id)

	res, err := z.api("api/v1/events", "GET", v)
	if err != nil {
		panic(err)
	}

	return res
}

func (z *Zulip) GetEvents(handler EventListener) {
	var last_event_id int64 = -1
	for {
		bytes := z.tryToGetEvents(strconv.FormatInt(last_event_id, 10))
		var res EventsResponse
		err := json.Unmarshal(bytes, &res)
		if err != nil {
			panic(err)
		}

		if res.Result != "success" {
			continue
		}
		events := res.Events
		for _, event := range events {
			if event.ID > last_event_id {
				last_event_id = event.ID
			}
			if event.Type == "heartbeat" {
				continue
			}
			result := handler.HandleEvent(event)
			if !result {
				return
			}
		}
	}
}

type messageType string

const (
	privateMessage messageType = "private"
	streamMessage  messageType = "stream"
)

type message struct {
	Type                  messageType
	Content               string
	StreamName, TopicName string
	Usernames             []string
}

func NewPrivateMessage(recipients []string) *message {
	return &message{Type: privateMessage, Usernames: recipients}
}

func NewStreamMessage(stream, topic string) *message {
	return &message{Type: streamMessage, StreamName: stream, TopicName: topic}
}

func (z *Zulip) SendMessage(msg *message) {
	v := url.Values{}
	v.Set("type", string(msg.Type))
	v.Set("content", msg.Content)
	switch msg.Type {
	case privateMessage:
		recipients, _ := json.Marshal(msg.Usernames)
		v.Set("to", string(recipients))
	case streamMessage:
		v.Set("to", msg.StreamName)
		v.Set("subject", msg.TopicName)
	}

	z.api("api/v1/messages", "POST", v)
}
