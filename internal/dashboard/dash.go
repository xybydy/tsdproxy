// SPDX-FileCopyrightText: 2025 Fatih Ka. <xybydy@gmail.com>
// SPDX-License-Identifier: MIT

package dashboard

import (
	"sync"
	"time"

	"github.com/xybydy/tsdproxy/internal/consts"
	"github.com/xybydy/tsdproxy/internal/core"
	"github.com/xybydy/tsdproxy/internal/model"
	"github.com/xybydy/tsdproxy/internal/proxymanager"
	"github.com/xybydy/tsdproxy/internal/ui/pages"
	"github.com/xybydy/tsdproxy/web"

	"github.com/rs/zerolog"
)

type Dashboard struct {
	Log        zerolog.Logger
	HTTP       *core.HTTPServer
	pm         *proxymanager.ProxyManager
	sseClients map[string]*sseClient
	mtx        sync.RWMutex
}

func NewDashboard(http *core.HTTPServer, log zerolog.Logger, pm *proxymanager.ProxyManager) *Dashboard {
	dash := &Dashboard{
		Log:  log.With().Str("module", "dashboard").Logger(),
		HTTP: http,
		pm:   pm,
	}
	dash.sseClients = make(map[string]*sseClient)

	go dash.streamProxyUpdates()
	go dash.startClientCleanup()

	return dash
}

// AddRoutes method add dashboard related routes to the http server
func (dash *Dashboard) AddRoutes() {
	dash.HTTP.Get("/stream", dash.streamHandler())
	dash.HTTP.Get("/", web.Static)
}

// index is the HandlerFunc to index page of dashboard
func (dash *Dashboard) renderList(ch chan SSEMessage) {
	dash.mtx.RLock()
	defer dash.mtx.RUnlock()

	// force remove elements of proxy-list inn case of client reconnect
	ch <- SSEMessage{
		Type:    EventRemoveMessage,
		Message: "#proxy-list>*",
	}

	for name, p := range dash.pm.Proxies {
		if p.Config.Dashboard.Visible {
			dash.renderProxy(ch, name, EventAppend)
		}
	}

	dash.streamSortList(ch)
}

func (dash *Dashboard) renderProxy(ch chan SSEMessage, name string, ev EventType) {
	p, ok := dash.pm.GetProxy(name)
	if !ok {
		return
	}

	status := p.GetStatus()

	url := p.GetURL()
	if status == model.ProxyStatusAuthenticating {
		url = p.GetAuthURL()
	}

	icon := p.Config.Dashboard.Icon
	if icon == "" {
		icon = model.DefaultDashboardIcon
	}

	label := p.Config.Dashboard.Label
	if label == "" {
		label = name
	}

	ports := make([]model.PortConfig, len(p.Config.Ports))
	i := 0
	for _, target := range p.Config.Ports {
		ports[i] = target
		i++
	}

	enabled := status == model.ProxyStatusAuthenticating || status == model.ProxyStatusRunning

	a := pages.ProxyData{
		Enabled:     enabled,
		Name:        name,
		URL:         url,
		ProxyStatus: status,
		Icon:        icon,
		Label:       label,
		Ports:       ports,
	}

	ch <- SSEMessage{
		Type: ev,
		Comp: pages.Proxy(a),
	}
}

// startClientCleanup periodically removes stale SSE clients
func (dash *Dashboard) startClientCleanup() {
	ticker := time.NewTicker(consts.ClientCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		dash.cleanupStaleClients()
	}
}

// cleanupStaleClients removes clients that haven't been active recently
func (dash *Dashboard) cleanupStaleClients() {
	dash.mtx.Lock()
	defer dash.mtx.Unlock()

	now := time.Now()
	staleThreshold := consts.StaleClientThreshold

	for sessionID, client := range dash.sseClients {
		if now.Sub(client.lastActive) > staleThreshold {
			dash.Log.Warn().Str("sessionID", sessionID).Msg("Removing stale SSE client")
			delete(dash.sseClients, sessionID)
			close(client.channel)
		}
	}
}
