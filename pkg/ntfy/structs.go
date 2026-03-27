package ntfy

import "encoding/json"

type Message struct {
	ID       string   `json:"sequence_id,omitempty"` // populated by Send on new message; set to update existing
	Topic    string   `json:"topic"`                 // Required
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

type RunOptions struct {
	TaskName string `json:"taskName"`
	Par1     string `json:"par1,omitempty"`
	Par2     string `json:"par2,omitempty"`
	Priority int    `json:"priority,omitempty"`
}

type VarOptions struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TaskerApi struct {
	Run     []RunOptions `json:"run,omitempty"`
	Set     []VarOptions `json:"set,omitempty"`
	Enable  []string     `json:"enable,omitempty"`
	Disable []string     `json:"disable,omitempty"`
}

func (t *TaskerApi) ToString() string {
	b, _ := json.Marshal(t)
	return string(b)
}
