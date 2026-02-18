// SPDX-FileCopyrightText: 2026 Fatih Ka. <xybydy@gmail.com>
// SPDX-License-Identifier: MIT

package targetproviders

import (
	"context"

	"github.com/xybydy/tsdproxy/internal/model"
)

type (
	// TargetProvider interface to be implemented by all target providers
	TargetProvider interface {
		WatchEvents(ctx context.Context, eventsChan chan TargetEvent, errChan chan error)
		GetDefaultProxyProviderName() string
		Close()
		AddTarget(id string) (*model.Config, error)
		RemoveTarget(id string)
		DeleteProxy(id string) error
	}
)

const (
	ActionStartProxy ActionType = iota + 1
	ActionStopProxy
	ActionRestartProxy
	ActionStartPort
	ActionStopPort
	ActionRestartPort
)

type (
	ActionType int

	TargetEvent struct {
		TargetProvider TargetProvider
		ID             string
		Action         ActionType
	}
)
