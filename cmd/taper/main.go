package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/harrylincoln/taper/internal/api"
	"github.com/harrylincoln/taper/internal/proxy"
	"github.com/harrylincoln/taper/internal/throttle"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	const proxyAddr = ":8807"
	const apiAddr = ":5507"

	// Define profiles
	profiles := []throttle.Profile{
		{Name: "Full", Level: 10, LatencyMs: 0, DownloadBytesPerSec: 0, UploadBytesPerSec: 0}, // 0 = unlimited
		{Name: "Good", Level: 8, LatencyMs: 80, DownloadBytesPerSec: 2_000_000, UploadBytesPerSec: 1_000_000},
		{Name: "OK", Level: 5, LatencyMs: 300, DownloadBytesPerSec: 300_000, UploadBytesPerSec: 150_000},
		{Name: "Bad", Level: 3, LatencyMs: 800, DownloadBytesPerSec: 64_000, UploadBytesPerSec: 32_000},
		{Name: "Terrible", Level: 1, LatencyMs: 2000, DownloadBytesPerSec: 16_000, UploadBytesPerSec: 8_000},
	}

	manager := throttle.NewManager(profiles, 10)

	proxySrv := proxy.NewServer(proxyAddr, manager)
	apiSrv := api.NewServer(apiAddr, manager)

	// Start proxy
	go func() {
		log.Println("Proxy listening on", proxyAddr)
		if err := proxySrv.Start(); err != nil {
			handleListenError("proxy", proxyAddr, err)
		}
	}()

	// Start API
	go func() {
		log.Println("API listening on", apiAddr)
		if err := apiSrv.Start(); err != nil {
			handleListenError("api", apiAddr, err)
		}
	}()

	printManualSetupInstructions(proxyAddr, apiAddr)

	<-ctx.Done()
	log.Println("Shutting down...")
	proxySrv.Shutdown()
	apiSrv.Shutdown()
}

func handleListenError(name, addr string, err error) {
	// Ignore "server closed" on shutdown

	var opErr *net.OpError
	if errors.As(err, &opErr) && errors.Is(opErr.Err, syscall.EADDRINUSE) {
		log.Fatalf("%s failed to start on %s: port already in use. Is another instance running?", name, addr)
	}

	log.Fatalf("%s failed to start on %s: %v", name, addr, err)
}

func printManualSetupInstructions(proxyAddr, apiAddr string) {
	// Only print macOS-specific stuff on macOS
	if runtime.GOOS != "darwin" {
		log.Println("Running on non-macOS; configure your system/browser to use the proxy at 127.0.0.1" + proxyAddr)
		return
	}

	log.Println("──────────────────────────────────────────────────────")
	log.Println(" Taper daemon is running.")
	log.Println("")
	log.Println(" To route your Mac traffic through this proxy:")
	log.Println("")
	log.Println(" 1) Find your network service name (e.g. \"Wi-Fi\"):")
	log.Println("      networksetup -listallnetworkservices")
	log.Println("")
	log.Println(" 2) Enable HTTP/HTTPS proxies for that service:")
	log.Printf("      networksetup -setwebproxy \"Wi-Fi\" 127.0.0.1 %s\n", proxyAddr[1:])
	log.Printf("      networksetup -setsecurewebproxy \"Wi-Fi\" 127.0.0.1 %s\n", proxyAddr[1:])
	log.Println("")
	log.Println("    (Replace \"Wi-Fi\" if your service is named differently.)")
	log.Println("")
	log.Println(" 3) To turn proxies OFF again later:")
	log.Println("      networksetup -setwebproxystate \"Wi-Fi\" off")
	log.Println("      networksetup -setsecurewebproxystate \"Wi-Fi\" off")
	log.Println("")
	log.Printf(" API for the Chrome extension is available at http://127.0.0.1%s\n", apiAddr)
	log.Println("──────────────────────────────────────────────────────")
}
