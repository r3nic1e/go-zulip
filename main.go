package zulip

import "net/http"
import "net/url"
import "io/ioutil"
import "fmt"
import "encoding/json"
import "strconv"

var BaseUrl string
var authLogin, authPass string

type EventListener interface {
	HandleEvent(map[string]interface{}) bool
}

func SetBasicAuth(login, pass string) {
	authLogin = login
	authPass = pass
}

func api(url, method string, params url.Values) []byte {
	client := &http.Client{}

	url = fmt.Sprintf("%s/%s?%s", BaseUrl, url, params.Encode())

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	req.SetBasicAuth(authLogin, authPass)

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return body
}

func Register(event_types []string) string {
	v := url.Values{}
	json_types, _ := json.Marshal(event_types)
	v.Set("event_types", string(json_types))
	bytes := api("api/v1/register", "POST", v)
	var res map[string]interface{}
	json.Unmarshal(bytes, &res)
	return res["queue_id"].(string)
}

func tryToGetEvents(queue_id, last_event_id string) map[string]interface{} {
	v := url.Values{}
	v.Set("queue_id", queue_id)
	v.Set("last_event_id", last_event_id)

	bytes := api("api/v1/events", "GET", v)
	var res map[string]interface{}
	json.Unmarshal(bytes, &res)

	return res
}

func GetEvents(queue_id string, handler EventListener) {
	last_event_id := -1
	for {
		result := tryToGetEvents(queue_id, strconv.Itoa(last_event_id))
		if result["result"] != "success" {
			continue
		}
		events := result["events"].([]interface{})
		for _, e := range events {
			event := e.(map[string]interface{})
			id := int(event["id"].(float64))
			if id > last_event_id {
				last_event_id = id
			}
			if event["type"] == "heartbeat" {
				continue
			}
			result := handler.HandleEvent(event)
			if !result {
				return
			}
		}
	}
}
