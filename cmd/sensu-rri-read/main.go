package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/DENICeG/go-rriclient/pkg/rri"
	"github.com/danielb42/whiteflag"
)

var (
	timeBegin     = time.Now()
	rriClient     *rri.Client
	fails         int
	domainToCheck string
	regacc        string
	password      string
	rriServer     string
)

func main() {
	whiteflag.Alias("d", "domain", "sets the domain to use in CHECK order")
	whiteflag.Alias("r", "regacc", "sets the regacc to use")
	whiteflag.Alias("p", "password", "sets the password to use")
	whiteflag.Alias("s", "server", "sets the RRI server to use")

	domainToCheck = whiteflag.GetString("domain")
	regacc = whiteflag.GetString("regacc")
	password = whiteflag.GetString("password")
	rriServer = whiteflag.GetString("server") + ":51131"

	run()
}

func run() {
	var err error
	log.SetOutput(os.Stderr)
	log.SetPrefix("UTC ")
	log.SetFlags(log.Ltime | log.Lmsgprefix | log.LUTC)

	if rriClient != nil {
		rriClient.Logout() // nolint:errcheck
	}

	rriClient, err = rri.NewClient(rriServer, nil)
	if err != nil {
		printFailMetricsAndExit("could not connect to RRI server:", err.Error())
	}

	err = rriClient.Login(regacc, password)
	if err != nil {
		printFailMetricsAndExit("login failed:", err.Error())
	}

	timeLoginDone := time.Now()

	rriQuery := rri.NewCheckDomainQuery(domainToCheck)
	rriResponse, err := rriClient.SendQuery(rriQuery)
	if err != nil {
		printFailMetricsAndExit("SendQuery() failed:", err.Error())
	}

	durationLogin := timeLoginDone.Sub(timeBegin).Milliseconds()
	durationOrder := time.Now().Sub(timeLoginDone).Milliseconds() // nolint:gosimple
	durationTotal := durationLogin + durationOrder

	if rriResponse.IsSuccessful() {
		log.Printf("OK: RRI login + order: %dms + %dms = %dms\n\n", durationLogin, durationOrder, durationTotal)
		fmt.Printf("extmon,service=%s,ordertype=%s %s=%d,%s=%d,%s=%d,%s=%d %d\n",
			"rri",
			"CHECK",
			"available", 1,
			"login", durationLogin,
			"order", durationOrder,
			"total", durationTotal,
			timeBegin.Unix())
	} else {
		printFailMetricsAndExit("invalid response from RRI")
	}

	rriClient.Logout() // nolint:errcheck
	os.Exit(0)
}

func printFailMetricsAndExit(errors ...string) {

	if fails < 3 {
		fails++
		run()
	}

	errStr := "ERROR:"

	for _, err := range errors {
		errStr += " " + err
	}

	log.Printf("%s\n\n", errStr)

	fmt.Printf("extmon,service=%s,ordertype=%s %s=%d,%s=%d,%s=%d,%s=%d %d\n",
		"rri",
		"CHECK",
		"available", 0,
		"login", 0,
		"order", 0,
		"total", 0,
		timeBegin.Unix())

	if rriClient != nil {
		rriClient.Logout() // nolint:errcheck
	}

	os.Exit(2)
}
