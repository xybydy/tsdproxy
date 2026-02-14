// SPDX-FileCopyrightText: 2025 Paulo Almeida <almeidapaulopt@gmail.com>
// SPDX-License-Identifier: MIT

package consts

import "time"

// HTTP Headers
const (
	HeaderUsername      = "X-tsdproxy-username"
	HeaderDisplayName   = "x-tsdproxy-displayName"
	HeaderProfilePicURL = "x-tsdproxy-profilePicUrl"
)

// Concurrency and Buffer Sizes
const (
	// MaxConcurrentEventHandlers limits concurrent event processing goroutines
	MaxConcurrentEventHandlers = 50

	// EventChannelBufferSize is the buffer size for event channels
	EventChannelBufferSize = 100

	// ErrorChannelBufferSize is the buffer size for error channels
	ErrorChannelBufferSize = 1

	// StatusEventChannelBufferSize is the buffer size for status event subscribers
	StatusEventChannelBufferSize = 100

	// SignalChannelBufferSize prevents signal loss during rapid signal arrival
	SignalChannelBufferSize = 10
)

// Time Intervals
const (
	// ClientCleanupInterval is the interval for cleaning up stale SSE clients
	ClientCleanupInterval = 5 * time.Minute

	// ContainerReconcileInterval is the interval for reconciling container state
	ContainerReconcileInterval = 5 * time.Minute

	// StaleClientThreshold is the duration after which a client is considered stale
	StaleClientThreshold = 15 * time.Minute
)
