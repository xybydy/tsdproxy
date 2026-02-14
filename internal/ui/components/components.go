// SPDX-FileCopyrightText: 2025 Fatih Ka. <xybydy@gmail.com>
// SPDX-License-Identifier: MIT

package components

func IconURL(name string) string {
	if name == "" {
		name = "tsdproxy"
	}
	return "/icons/" + name + ".svg"
}
