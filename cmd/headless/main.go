package main

import (
	"flag"
	"fmt"
	"internet-monitor/internal/app"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var Version = "dev"

func main() {
	dir := flag.String("dir", envOr("IM_DIR", app.Dir()), "folder for config.json, case.json and logs")
	listen := flag.String("listen", envOr("IM_LISTEN", "127.0.0.1"), "address the dashboard listens on; 0.0.0.0 opens it to your network")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(Version)
		return
	}

	a, err := app.New(app.Options{Version: Version, Dir: *dir, ListenHost: *listen})
	if err != nil {
		log.Fatalf("startup failed: %v", err)
	}
	a.Run()

	fmt.Printf("Internet Monitor %s is running (data in %s)\n", Version, *dir)
	fmt.Println("Open the dashboard with this link (it contains your access key, keep it private):")
	fmt.Println("  " + a.Server.LaunchURL())
	if a.Server.LANMode() {
		for _, u := range a.Server.LANLaunchURLs() {
			fmt.Println("  " + u)
		}
		fmt.Println("The dashboard is reachable from your network. Only do this on a network you trust.")
	}
	a.Log.AppLog("HEADLESS dashboard %s (listen %s)", a.Server.URL(), *listen)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	a.Stop()
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
