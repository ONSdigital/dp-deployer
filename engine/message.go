package message

// MessageSQS represents a message that has been consumed.
type MessageSQS struct {
	Job         string
	Java        bool         `json:"java,omitempty"`
	Go          bool         `json:"go,omitempty"`
	Publishing  *group       `json:"publishing,omitempty"`
	Web         *group       `json:"web,omitempty"`
	Healthcheck *healthcheck `json:"healthcheck,omitempty"`
}

type groups struct {
	Mount           bool     `json:"mount,omitempty"`
	Volumes         []string `json:"volumes,omitempty"`
	UsernsMode      bool     `json:"userns_mode,omitempty"`
	CommandLineArgs []string `json:"command_line_args,omitempty"`
	CPU             string
	Memory          string
	TaskCount       string
	HeapMemory      string `json:"heap_memory,omitempty"`
}

type healthcheck struct {
	Enabled bool   `json:"enabled,omitempty"`
	Path    string `json:"path,omitempty"`
}
