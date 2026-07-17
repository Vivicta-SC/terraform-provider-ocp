// Copyright @ Vivicta. All Rights Reserved. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"net/http"
)

// InsecureSkipVerify returns true if TLS verification is disabled.
// ponytail: test helper helper to verify TLS configuration from client.
func (c *OCPClient) InsecureSkipVerify() bool {
	if c.http == nil {
		return false
	}
	if transport, ok := c.http.Transport.(*http.Transport); ok {
		if transport.TLSClientConfig != nil {
			return transport.TLSClientConfig.InsecureSkipVerify
		}
	}
	return false
}
