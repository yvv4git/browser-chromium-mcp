package domain

type Health struct {
	OK               bool   `json:"ok"`
	Engine           string `json:"engine"`
	Sessions         int    `json:"sessions"`
	BrowserConnected bool   `json:"browserConnected"`
}

type Tab struct {
	TabID    string `json:"tabId"`
	TargetID string `json:"targetId"`
	URL      string `json:"url"`
	Title    string `json:"title"`
}

type Snapshot struct {
	URL       string `json:"url"`
	Snapshot  string `json:"snapshot"`
	RefsCount int    `json:"refsCount"`
}

type Evaluate struct {
	OK     bool `json:"ok"`
	Result any  `json:"result"`
}

type Link struct {
	URL  string `json:"url"`
	Text string `json:"text"`
}

type Links struct {
	Links   []Link `json:"links"`
	Total   int    `json:"total"`
	Offset  int    `json:"offset"`
	Limit   int    `json:"limit"`
	HasMore bool   `json:"hasMore"`
}

type Resource struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Size   int    `json:"size"`
	Status int    `json:"status"`
}

type Stats struct {
	TabID       string   `json:"tabId"`
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	VisitedURLs []string `json:"visitedUrls"`
	ToolCalls   int      `json:"toolCalls"`
	RefsCount   int      `json:"refsCount"`
}

type TabInfo struct {
	TabID string `json:"tabId"`
	URL   string `json:"url"`
	Title string `json:"title"`
}

type ListTabs struct {
	Tabs []TabInfo `json:"tabs"`
}
