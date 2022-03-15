package message

// MessageSQS represents a message that has been consumed.
// TODO there is work in the CI to create the new SQS message for this struct
type MessageSQS struct {
	Job                  string
	Java                 bool         `json:"java,omitempty"`
	Go                   bool         `json:"go,omitempty"`
	Publishing           *Groups      `json:"publishing,omitempty"`
	Web                  *Groups      `json:"web,omitempty"`
	PublishingCantabular *Groups      `json:"publishing_cantabular,omitempty"`
	WebCantabular        *Groups      `json:"web_cantabular,omitempty"`
	Healthcheck          *Healthcheck `json:"healthcheck,omitempty"`
	Revision             string
}

// Groups represents the publishing or web group for the MessageSQS
type Groups struct {
	Mount           bool     `json:"mount,omitempty"`
	DistinctHosts   bool     `json:"distinct_hosts,omitempty"`
	Volumes         []string `json:"volumes,omitempty"`
	UsernsMode      bool     `json:"userns_mode,omitempty"`
	CommandLineArgs []string `json:"command_line_args,omitempty"`
	CPU             int
	Memory          int
	TaskCount       int
	HeapMemory      string `json:"heap_memory,omitempty"`
}

// Healthcheck represents the healthcheck config
type Healthcheck struct {
	Enabled bool   `json:"enabled,omitempty"`
	Path    string `json:"path,omitempty"`
}
