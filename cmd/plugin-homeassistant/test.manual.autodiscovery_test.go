//go:build manual

// TestManual_AutoDiscovery verifies that HA can discover the plugin via mDNS
// and complete the hello handshake. Run with:
//
//	go test -tags manual -v -run TestManual_AutoDiscovery -timeout 120s ./cmd/plugin-homeassistant/
//
// Then open the HA integrations dashboard and verify the Slidebolt popup appears.
// If you submit it, HA will connect and the test will log the connection.
package main

import (
	"log"
	"testing"
	"time"

	"github.com/slidebolt/plugin-homeassistant/app"
)

func TestManual_AutoDiscovery(t *testing.T) {
	srv := app.NewServerForTest(app.Config{Port: "0", Token: "", SystemUUID: "test-uuid", SystemMAC: "E0:00:00:00:00:FF"})
	port, err := srv.Start()
	if err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer srv.Stop()

	log.Printf("TestManual_AutoDiscovery: WebSocket server on port %d", port)
	log.Printf("TestManual_AutoDiscovery: mDNS advertising _slidebolt._tcp")
	log.Printf("TestManual_AutoDiscovery: open HA → Settings → Devices & Services and look for Slidebolt")
	log.Printf("TestManual_AutoDiscovery: waiting 60s for HA connections...")

	time.Sleep(60 * time.Second)

	log.Printf("TestManual_AutoDiscovery: done")
}
