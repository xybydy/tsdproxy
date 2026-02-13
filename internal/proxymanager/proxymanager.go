// SPDX-FileCopyrightText: 2025 Paulo Almeida <almeidapaulopt@gmail.com>
// SPDX-License-Identifier: MIT

package proxymanager

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/almeidapaulopt/tsdproxy/internal/config"
	"github.com/almeidapaulopt/tsdproxy/internal/model"
	"github.com/almeidapaulopt/tsdproxy/internal/proxyproviders"
	"github.com/almeidapaulopt/tsdproxy/internal/proxyproviders/tailscale"
	"github.com/almeidapaulopt/tsdproxy/internal/targetproviders"
	"github.com/almeidapaulopt/tsdproxy/internal/targetproviders/docker"
	"github.com/almeidapaulopt/tsdproxy/internal/targetproviders/list"
)

type (
	ProxyList          map[string]*Proxy
	TargetProviderList map[string]targetproviders.TargetProvider
	ProxyProviderList  map[string]proxyproviders.Provider

	// ProxyManager struct stores data that is required to manage all proxies
	ProxyManager struct {
		Proxies ProxyList

		log zerolog.Logger

		TargetProviders TargetProviderList
		ProxyProviders  ProxyProviderList

		statusSubscribers map[chan model.ProxyEvent]*subscriber

		// eventWorkerPool limits concurrent event handler goroutines
		eventWorkerPool chan struct{}

		mtx sync.RWMutex
	}

	subscriber struct {
		ch       chan model.ProxyEvent
		lastSeen time.Time
	}
)

var (
	ErrProxyProviderNotFound  = errors.New("proxyProvider not found")
	ErrTargetProviderNotFound = errors.New("targetProvider not found")
)

// NewProxyManager function creates a new ProxyManager.
func NewProxyManager(logger zerolog.Logger) *ProxyManager {
	pm := &ProxyManager{
		Proxies:           make(ProxyList),
		TargetProviders:   make(TargetProviderList),
		ProxyProviders:    make(ProxyProviderList),
		statusSubscribers: make(map[chan model.ProxyEvent]*subscriber),
		eventWorkerPool:   make(chan struct{}, 50), // Max 50 concurrent event handlers
		log:               logger.With().Str("module", "proxymanager").Logger(),
	}

	// Start cleanup routine for stale subscribers
	//go pm.startSubscriberCleanup()

	return pm
}

// Start method starts the ProxyManager.
func (pm *ProxyManager) Start() {
	// Add Providers
	pm.addProxyProviders()
	pm.addTargetProviders()

	// Do not start without providers
	if len(pm.ProxyProviders) == 0 {
		pm.log.Error().Msg("No Proxy Providers found")
		return
	}

	if len(pm.TargetProviders) == 0 {
		pm.log.Error().Msg("No Target Providers found")
		return
	}
}

// StopAllProxies method shuts down all proxies.
func (pm *ProxyManager) StopAllProxies() {
	pm.log.Info().Msg("Shutdown all proxies")

	pm.mtx.RLock()
	proxyIDs := make([]string, 0, len(pm.Proxies))
	for id := range pm.Proxies {
		proxyIDs = append(proxyIDs, id)
	}
	pm.mtx.RUnlock()

	wg := sync.WaitGroup{}
	wg.Add(len(proxyIDs))

	for _, id := range proxyIDs {
		go func(proxyID string) {
			defer wg.Done()
			pm.removeProxy(proxyID)
		}(id)
	}

	wg.Wait()
}

// WatchEvents method watches for events from all target providers.
func (pm *ProxyManager) WatchEvents(ctx context.Context) {
	var wg sync.WaitGroup

	for _, provider := range pm.TargetProviders {
		wg.Add(1)
		go func(provider targetproviders.TargetProvider) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					pm.log.Error().Interface("panic", r).Msg("event watcher panicked")
				}
			}()

			eventsChan := make(chan targetproviders.TargetEvent, 100)
			errChan := make(chan error, 1)

			// Start provider's event watcher
			go provider.WatchEvents(ctx, eventsChan, errChan)

			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-eventsChan:
					if !ok {
						pm.log.Debug().Msg("events channel closed, stopping watcher")
						return
					}
					// Use worker pool to limit concurrent event handlers
					select {
					case pm.eventWorkerPool <- struct{}{}: // Acquire semaphore
						go func(e targetproviders.TargetEvent) {
							defer func() { <-pm.eventWorkerPool }() // Release semaphore
							pm.HandleProxyEvent(e)
						}(event)
					case <-ctx.Done():
						return
					}
				case err, ok := <-errChan:
					if !ok {
						pm.log.Debug().Msg("error channel closed, stopping watcher")
						return
					}
					pm.log.Err(err).Msg("Error watching events")
				}
			}
		}(provider)
	}

	// Wait for all watchers to finish when context is canceled
	go func() {
		<-ctx.Done()
		pm.log.Debug().Msg("Context canceled, waiting for watchers to finish")
		wg.Wait()
		pm.log.Debug().Msg("All watchers finished")
	}()
}

// HandleProxyEvent method handles events from a targetprovider
func (pm *ProxyManager) HandleProxyEvent(event targetproviders.TargetEvent) {
	switch event.Action {
	case targetproviders.ActionStartProxy:
		pm.eventStart(event)
	case targetproviders.ActionStopProxy:
		pm.eventStop(event)
	case targetproviders.ActionRestartProxy:
		pm.eventStop(event)
		pm.eventStart(event)
	}
}

// SubscribeStatusEvents return a channel of proxy events.
// This events are sent by Proxies and Ports.
func (pm *ProxyManager) SubscribeStatusEvents() <-chan model.ProxyEvent {
	ch := make(chan model.ProxyEvent, 100) // Buffered channel to prevent blocking

	pm.mtx.Lock()
	pm.statusSubscribers[ch] = &subscriber{
		ch:       ch,
		lastSeen: time.Now(),
	}
	pm.mtx.Unlock()

	return ch
}

// UnsubscribeStatusEvents remove the channel subscrived in SubscribeStatusEvents
func (pm *ProxyManager) UnsubscribeStatusEvents(ch chan model.ProxyEvent) {
	pm.mtx.Lock()
	delete(pm.statusSubscribers, ch)
	pm.mtx.Unlock()
	close(ch)
}

func (pm *ProxyManager) GetProxies() ProxyList {
	pm.mtx.RLock()
	defer pm.mtx.RUnlock()

	return pm.Proxies
}

func (pm *ProxyManager) GetProxy(name string) (*Proxy, bool) {
	pm.mtx.RLock()
	defer pm.mtx.RUnlock()

	proxy, ok := pm.Proxies[name]

	return proxy, ok
}

// broadcastStatusEvents broadcasts proxy status event to all SubscribeStatusEvents
func (pm *ProxyManager) broadcastStatusEvents(event model.ProxyEvent) {
	pm.mtx.RLock()
	subscribers := make([]chan model.ProxyEvent, 0, len(pm.statusSubscribers))
	for ch := range pm.statusSubscribers {
		subscribers = append(subscribers, ch)
	}
	pm.mtx.RUnlock()

	// Send without holding lock to prevent deadlock
	for _, ch := range subscribers {
		select {
		case ch <- event:
			// Update lastSeen timestamp for active subscriber
			pm.mtx.Lock()
			if sub, ok := pm.statusSubscribers[ch]; ok {
				sub.lastSeen = time.Now()
			}
			pm.mtx.Unlock()
		default:
			// Channel full, skip this subscriber to prevent blocking
			pm.log.Warn().Msg("Subscriber channel full, skipping event broadcast")
		}
	}
}

// addTargetProviders method adds TargetProviders from configuration file.
func (pm *ProxyManager) addTargetProviders() {
	for name, provider := range config.Config.Docker {
		p, err := docker.New(pm.log, name, provider)
		if err != nil {
			pm.log.Error().Err(err).Msg("Error creating Docker provider")
			continue
		}

		pm.addTargetProvider(p, name)
	}
	for name, file := range config.Config.Lists {
		p, err := list.New(pm.log, name, file)
		if err != nil {
			pm.log.Error().Err(err).Msg("Error creating Files provider")
			continue
		}

		pm.addTargetProvider(p, name)
	}
}

// addProxyProviders method adds ProxyProviders from configuration file.
func (pm *ProxyManager) addProxyProviders() {
	pm.log.Debug().Msg("Setting up Tailscale Providers")
	// add Tailscale Providers
	for name, provider := range config.Config.Tailscale.Providers {
		if p, err := tailscale.New(pm.log, name, provider); err != nil {
			pm.log.Error().Err(err).Msg("Error creating Tailscale provider")
		} else {
			pm.log.Debug().Str("provider", name).Msg("Created Proxy provider")
			pm.addProxyProvider(p, name)
		}
	}
}

// addTargetProvider method adds a TargetProvider to the ProxyManager.
func (pm *ProxyManager) addTargetProvider(provider targetproviders.TargetProvider, name string) {
	pm.mtx.Lock()
	defer pm.mtx.Unlock()

	pm.TargetProviders[name] = provider
}

// addProxyProvider method adds	a ProxyProvider to the ProxyManager.
func (pm *ProxyManager) addProxyProvider(provider proxyproviders.Provider, name string) {
	pm.mtx.Lock()
	defer pm.mtx.Unlock()

	pm.ProxyProviders[name] = provider
}

// addProxy method adds a Proxy to the ProxyManager.
func (pm *ProxyManager) addProxy(proxy *Proxy) {
	pm.mtx.Lock()
	defer pm.mtx.Unlock()

	pm.Proxies[proxy.Config.Hostname] = proxy
}

// removeProxy method removes a Proxy from the ProxyManager.
func (pm *ProxyManager) removeProxy(hostname string) {
	pm.mtx.RLock()
	proxy, exists := pm.Proxies[hostname]
	pm.mtx.RUnlock()

	if !exists {
		return
	}

	proxy.Close()

	pm.mtx.Lock()
	defer pm.mtx.Unlock()

	delete(pm.Proxies, hostname)

	pm.log.Debug().Str("proxy", hostname).Msg("Removed proxy")
}

// eventStart method starts a Proxy from a event trigger
func (pm *ProxyManager) eventStart(event targetproviders.TargetEvent) {
	pm.log.Debug().Str("targetID", event.ID).Msg("Adding target")

	pcfg, err := event.TargetProvider.AddTarget(event.ID)
	if err != nil {
		pm.log.Error().Err(err).Str("targetID", event.ID).Msg("Error adding target")
		return
	}

	pm.newAndStartProxy(pcfg.Hostname, pcfg)
}

// eventStop method stops a Proxy from a event trigger
func (pm *ProxyManager) eventStop(event targetproviders.TargetEvent) {
	pm.log.Debug().Str("targetID", event.ID).Msg("Stopping target")

	proxy := pm.getProxyByTargetID(event.ID)
	if proxy == nil {
		pm.log.Error().Int("action", int(event.Action)).Str("target", event.ID).Msg("No proxy found for target")
		return
	}

	targetprovider := pm.TargetProviders[proxy.Config.TargetProvider]
	if err := targetprovider.DeleteProxy(event.ID); err != nil {
		pm.log.Error().Err(err).Msg("No proxy found for target")
		return
	}

	pm.removeProxy(proxy.Config.Hostname)
}

// getProxyByTargetID method returns a Proxy by TargetID.
func (pm *ProxyManager) getProxyByTargetID(targetID string) *Proxy {
	pm.mtx.RLock()
	defer pm.mtx.RUnlock()

	for _, p := range pm.Proxies {
		if p.Config.TargetID == targetID {
			return p
		}
	}
	return nil
}

// newAndStartProxy method creates a new proxy and starts it.
func (pm *ProxyManager) newAndStartProxy(name string, proxyConfig *model.Config) {
	pm.log.Debug().Str("proxy", name).Msg("Creating proxy")

	proxyProvider, err := pm.getProxyProvider(proxyConfig)
	if err != nil {
		pm.log.Error().Err(err).Msg("Error to get ProxyProvider")
		return
	}

	p, err := NewProxy(pm.log, proxyConfig, proxyProvider)
	if err != nil {
		pm.log.Error().Err(err).Msg("Error creating proxy")
		return
	}

	// any status change in proxy will be broadcasted
	p.onUpdate = func(event model.ProxyEvent) {
		pm.broadcastStatusEvents(event)
	}

	pm.addProxy(p)

	// broadcasts ProxyStatusInitializing
	pm.broadcastStatusEvents(model.ProxyEvent{
		ID:     p.Config.Hostname,
		Status: model.ProxyStatusInitializing,
	})

	p.Start()
}

// getProxyProvider method returns a ProxyProvider.
func (pm *ProxyManager) getProxyProvider(proxy *model.Config) (proxyproviders.Provider, error) {
	// return ProxyProvider defined in configurtion
	//
	if proxy.ProxyProvider != "" {
		p, ok := pm.ProxyProviders[proxy.ProxyProvider]
		if !ok {
			return nil, ErrProxyProviderNotFound
		}
		return p, nil
	}

	// return default ProxyProvider defined in TargetProvider
	targetProvider, ok := pm.TargetProviders[proxy.TargetProvider]
	if !ok {
		return nil, ErrTargetProviderNotFound
	}
	if p, ok := pm.ProxyProviders[targetProvider.GetDefaultProxyProviderName()]; ok {
		return p, nil
	}

	// return default ProxyProvider from global configurtion
	//
	if p, ok := pm.ProxyProviders[config.Config.DefaultProxyProvider]; ok {
		return p, nil
	}

	// return the first ProxyProvider
	//
	return nil, ErrProxyProviderNotFound
}
