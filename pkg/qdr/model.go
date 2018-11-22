package qdr

type RouterResource struct {
    Type string `json:"type"`
    Name string `json:"name"`
}

type LinkRoute struct {
    RouterResource
    Connection string `json:"connection"`
    Direction string `json:"direction"`
    Pattern string `json:"pattern"`
}

type Connector struct {
    RouterResource
    Host string `json:"host"`
    Port string `json:"port"` // yes, port is a string, as it could be named port
    Role string `json:"role"`
    SASLUsername string `json:"saslUsername"`
    SASLPassword string `json:"saslPassword"`
}
