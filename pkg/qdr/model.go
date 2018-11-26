/*
 * Copyright 2018, EnMasse authors.
 * License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).
 */

package qdr

const TypeNameConnector string = "org.apache.qpid.dispatch.connector"
const TypeNameLinkRoute string = "org.apache.qpid.dispatch.router.config.linkRoute"

type RouterResource interface {
    GetType() string
    GetName() string
}

type typeAndNameInfo struct {
    _type string
    _name string
}

func (r typeAndNameInfo) GetType() string {
    return r._type
}

func (r typeAndNameInfo) GetName() string {
    return r._name
}

func TypeAndName(Type string, Name string) RouterResource {
    return typeAndNameInfo{_type: Type, _name: Name}
}

type NamedResource struct {
    Name string `json:"name"`
}

func (r NamedResource) GetName() string {
    return r.Name
}

// link route

type LinkRoute struct {
    NamedResource
    Connection string `json:"connection"`
    Direction  string `json:"direction"`
    Pattern    string `json:"pattern"`
}

func (r LinkRoute) GetType() string {
    return TypeNameLinkRoute
}

// connector

type Connector struct {
    NamedResource
    Host         string `json:"host"`
    Port         string `json:"port"` // yes, port is a string, as it could be a named port
    Role         string `json:"role"`
    SASLUsername string `json:"saslUsername"`
    SASLPassword string `json:"saslPassword"`
}

func (r Connector) GetType() string {
    return TypeNameConnector
}
