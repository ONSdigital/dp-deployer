package message

// MessageSQS represents a message that has been consumed.
type MessageSQS struct {
	Job         string
	Java        bool         `json:"java,omitempty"`
	Go          bool         `json:"go,omitempty"`
	Publishing  *Groups      `json:"publishing,omitempty"`
	Web         *Groups      `json:"web,omitempty"`
	Healthcheck *healthcheck `json:"healthcheck,omitempty"`
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
	TaskCount       string
	HeapMemory      string `json:"heap_memory,omitempty"`
}

type healthcheck struct {
	Enabled bool   `json:"enabled,omitempty"`
	Path    string `json:"path,omitempty"`
}
