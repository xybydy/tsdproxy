// SPDX-FileCopyrightText: 2026 Fatih Ka. <xybydy@gmail.com>
// SPDX-License-Identifier: MIT

package model

import "context"

type (
	Whois struct {
		ID            string
		DisplayName   string
		Username      string
		ProfilePicURL string
	}
)

func (w *Whois) GetID() string {
	return w.ID
}

func (w *Whois) GetDisplayName() string {
	return w.DisplayName
}

func (w *Whois) GetUsername() string {
	return w.Username
}

func (w *Whois) GetProfilePicURL() string {
	return w.ProfilePicURL
}

func WhoisFromContext(ctx context.Context) (Whois, bool) {
	who, ok := ctx.Value(ContextKeyWhois).(Whois)

	return who, ok
}

func WhoisNewContext(ctx context.Context, who Whois) context.Context {
	return context.WithValue(ctx, ContextKeyWhois, who)
}
