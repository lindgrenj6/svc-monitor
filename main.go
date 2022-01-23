package main

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

var (
	// all of the svcs to monitor
	// hash represents a service in form:
	// K: subdomain
	// V: systemd svc
	SVCS = map[string]string{
		"git":      "container-gitea",
		"transfer": "container-transfersh",
		"blog":     "container-ghost",
		"echo":     "container-httpin",
		"lounge":   "container-thelounge",
		"hc":       "container-hc",
	}

	// global waitgroup to keep track of running sub-shells.
	wg = sync.WaitGroup{}
)

func main() {
	// guarantee everything will run. without this it just exits immediately
	wg.Add(len(SVCS))

	// do them all at once. really hammer caddy!
	for subdomain, service := range SVCS {
		go checkService(subdomain, service)
	}

	wg.Wait()
}

func checkService(subdomain, service string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		"http://"+subdomain+".lindgren.tech",
		nil,
	)
	if err != nil {
		fmt.Printf("Error creating req for %v, exiting routine.\n", subdomain)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil || (resp != nil && resp.StatusCode > 299) {
		if resp != nil {
			fmt.Printf("Got response status code of %v\n", resp.StatusCode)
		}

		fmt.Printf("e: %v\nk", err)

		wg.Add(1)
		go restartSvc(service)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("No issues with %v\n", subdomain)
	wg.Done()
}

func restartSvc(svc string) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx,
		"ssh", "caterpie",
		"systemctl", "restart", "--user", svc).CombinedOutput()
	if err != nil {
		fmt.Printf("Ran into error restarting service %v: %v\n", svc, err)
		fmt.Println(string(out))
	}

	err = exec.CommandContext(ctx, "tg", "restarted", svc).Run()
	if err != nil {
		fmt.Printf("Ran into error notifying about restart: %v", err)
	}

	wg.Done()
}
