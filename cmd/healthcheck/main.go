// SPDX-FileCopyrightText: 2025 Fatih Ka. <xybydy@gmail.com>
// SPDX-License-Identifier: MIT

package main

import (
	"net/http"
	"os"
)

func main() {
	h, err := http.Get("http://127.0.0.1:8080/health/ready/")
	if err != nil {
		os.Exit(1)
	}
	h.Body.Close()
}
