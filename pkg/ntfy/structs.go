package ntfy

type Message struct {
	Topic    string   `json:"topic"` // Required
	Message  string   `json:"message,omitempty"`
	Markdown bool     `json:"markdown"`
	Title    string   `json:"title"`              // Required
	Tags     []string `json:"tags,omitempty"`     // https://docs.ntfy.sh/publish/#tags-emojis
	Priority int      `json:"priority,omitempty"` // 1-5
	Delay    string   `json:"delay,omitempty"`    // Duration i.e. "5s"
	Icon     string   `json:"icon,omitempty"`     // URL
	Attach   string   `json:"attach,omitempty"`   // URL
	Filename string   `json:"filename,omitempty"`
	Click    string   `json:"click,omitempty"` // URL
	Actions  []struct {
		Action  string            `json:"action"` // "view", "broadcast", or "http"
		Label   string            `json:"label"`
		URL     string            `json:"url,omitempty"`
		Method  string            `json:"method,omitempty"`
		Headers string            `json:"headers,omitempty"`
		Body    string            `json:"body,omitempty"`
		Intent  string            `json:"intent,omitempty"`
		Clear   bool              `json:"clear,omitempty"`
		Extras  map[string]string `json:"extras,omitempty"`
	} `json:"actions,omitempty"`
}
