// SPDX-FileCopyrightText: 2025 Fatih Ka. <xybydy@gmail.com>
// SPDX-License-Identifier: MIT
package model

type (
	ProxyStatus int

	ProxyEvent struct {
		ID      string
		Port    string
		AuthURL string
		Status  ProxyStatus
	}
)

const (
	ProxyStatusInitializing ProxyStatus = iota
	ProxyStatusStarting
	ProxyStatusAuthenticating
	ProxyStatusRunning
	ProxyStatusStopping
	ProxyStatusStopped
	ProxyStatusError
)

var proxyStatusStrings = []string{
	"Initializing",
	"Starting",
	"Authenticating",
	"Running",
	"Stopping",
	"Stopped",
	"Error",
}

func (s *ProxyStatus) String() string {
	return proxyStatusStrings[int(*s)]
}
