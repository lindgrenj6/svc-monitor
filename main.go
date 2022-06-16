package main

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

type systemdScope uint8

const (
	userScope = iota
	systemScope

	maxRetries = 5
)

type service struct {
	// url to run a GET request against
	url string
	// server where the svc is running
	server string
	// systemd service name
	service string
	// systemd scope, either user or system
	scope systemdScope
}

var (
	// all of the svcs to monitor
	svcs = []service{
		// caterpie services
		{url: "https://git.lindgren.tech",
			service: "container-gitea",
			server:  "caterpie",
			scope:   userScope},
		{url: "https://transfer.lindgren.tech",
			service: "container-transfersh",
			server:  "caterpie",
			scope:   userScope},
		{url: "https://blog.lindgren.tech",
			service: "container-ghost",
			server:  "caterpie",
			scope:   userScope},
		{url: "https://echo.lindgren.tech",
			service: "container-httpin",
			server:  "caterpie",
			scope:   userScope},
		{url: "https://lounge.lindgren.tech",
			service: "container-thelounge",
			server:  "caterpie",
			scope:   userScope},
		{url: "https://hc.lindgren.tech",
			service: "container-hc",
			server:  "caterpie",
			scope:   userScope},

		// loudred svcs
		{url: "http://loudred:32400/web/index.html",
			service: "container-plex",
			server:  "loudred",
			scope:   systemScope},
		{url: "http://loudred:7878",
			service: "container-radarr",
			server:  "loudred",
			scope:   systemScope},
		{url: "http://loudred:8989",
			service: "container-sonarr",
			server:  "loudred",
			scope:   systemScope},
	}

	// global waitgroup to keep track of running sub-shells.
	wg = sync.WaitGroup{}
)

func main() {
	// guarantee everything will run. without this it just exits immediately
	wg.Add(len(svcs))

	// check all of them at once, async baby!!!
	for i := range svcs {
		go func(service *service) {
			err := checkService(service)
			if err != nil {
				restartSvc(service)
			}
			wg.Done()
		}(&svcs[i])
	}

	wg.Wait()
}

func checkService(srv *service) error {
	// retry at least maxRetries times just in case wg flakes or something
	for retry := 1; ; retry++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("Error hitting %v: %v\n", srv.url, err)
		} else if resp != nil && resp.StatusCode > 299 {
			fmt.Printf("Got response status code %v for url %v \n", resp.StatusCode, srv.url)
		} else {
			// we must not have any errors, break out of the loop.
			break
		}

		if retry == maxRetries {
			return fmt.Errorf("max retries reached")
		}

		time.Sleep(3 * time.Second)
	}

	fmt.Printf("No issues with %v\n", srv.url)
	return nil
}

func restartSvc(svc *service) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// constructing the args to the `ssh` command that we're going to be sending.
	args := []string{"systemctl", "restart", svc.service}
	// handle user vs system scoped systemd units
	switch svc.scope {
	case systemScope:
		args = append([]string{"sudo"}, args...)
	case userScope:
		args = append(args, "--user")
	}

	// prepending the server name
	// cmd now looks like `ssh server (sudo|) systemctl restart container-xxxx (|--user)`
	args = append([]string{svc.server}, args...)

	output, err := exec.CommandContext(ctx, "ssh", args...).CombinedOutput()
	if err != nil {
		fmt.Printf("Ran into error restarting service %v: %v\n", svc, err)
		fmt.Println(string(output))
		telegramNotify("🔄 svc-monitor couldn't restart " + svc.service + ": " + string(output))
	} else {
		telegramNotify("🔄 svc-monitor restarted " + svc.service)
	}
}
