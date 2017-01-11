package zulip

type BaseResponse struct {
	Result, Msg string
}

type RegisterResponse struct {
	BaseResponse
	Queue_id                      string
	Max_message_id, Last_event_id int64
}

type MessageResponse struct {
	Content, Subject, Content_type                                   string
	Client, Gravatar_hash, Avatar_url                                string
	ID                                                               int64
	Sender_id, Recipient_id                                          int
	Sender_email, Sender_domain, Sender_full_name, Sender_short_name string
	Display_recipient                                                interface{} // either string or 2 structs
	Subject_links, Reactions                                         []string
	Is_mentioned                                                     bool
	Type                                                             string
	Timestamp                                                        int64
}

type EventResponse struct {
	Flags   []string
	Message MessageResponse
	Type    string
	ID      int64
}

type EventsResponse struct {
	BaseResponse
	Queue_id string
	Events   []EventResponse
}
