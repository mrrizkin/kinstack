package system

type Shortcut struct {
	Key    string `json:"key"`
	Action string `json:"action"`
}

type PageProps struct {
	Surface   string     `json:"surface"`
	Topics    []string   `json:"topics"`
	Shortcuts []Shortcut `json:"shortcuts"`
}

type NotFoundPageProps struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}
