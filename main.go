package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env"
	"github.com/jpweber/cole/configuration"
	"github.com/jpweber/cole/notifier"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/jpweber/cole/dmtimer"
)

const (
	version = "v0.2.0"
)

var (
	ns   = notifier.NotificationSet{}
	conf = configuration.Conf{}
)

func init() {
	// Log as text. Color with tty attached
	log.SetFormatter(&log.TextFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	// log.SetLevel(log.WarnLevel)
}

func main() {

	versionPtr := flag.Bool("v", false, "Version")
	configFile := flag.String("c", "", "Path to Configuration File")

	// Once all flags are declared, call `flag.Parse()`
	// to execute the command-line parsing.
	flag.Parse()
	if *versionPtr == true {
		fmt.Println(version)
		os.Exit(0)
	}

	log.Println("Starting application...")

	// if no config file parameter was passed try env vars.
	if *configFile == "" {
		log.Info("Using ENV Vars for configuration")
		conf = configuration.Conf{}
		if err := env.Parse(&conf); err != nil {
			log.Fatal("Unable to parse envs: ", err)
		}
	} else {
		// read from config file
		log.Info("Reading from config file")
		conf = configuration.ReadConfig(*configFile)
	}

	// DEBUG
	// fmt.Printf("%+v", conf)

	// init first timer at launch of service
	// TODO:
	// figure out a way to start another timer after this alert fires.
	// we want this to continue to go off as long as the dead man
	// switch is not being tripped.

	// init notificaiton set
	ns = notifier.NotificationSet{
		Config: conf,
		Timers: dmtimer.DmTimers{},
	}

	// HTTP Handlers
	http.HandleFunc("/ping/", logger(ping))
	http.HandleFunc("/id", logger(genID))
	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, version)
	})
	http.Handle("/metrics", promhttp.Handler())

	// Server Lifecycle
	//To setup a insecure server for http without any tls validation
	/*
		s := &http.Server{
			Addr:         ":8080",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		go func() {
			log.Fatal(s.ListenAndServe())
		}()
	*/

	//To setup a https secure server with certificate validation.
	//To setup a https secure server with TLS 1.2
	// cfg := &tls.Config{
	// 	MinVersion:               tls.VersionTLS12,
	// 	PreferServerCipherSuites: true,
	// 	CipherSuites: []uint16{
	// 		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	// 		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	// 		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	// 		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	// 	},
	// }

	s := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	//err := http.ListenAndServeTLS(":443", "/usr/local/share/ca-certificates/https-server.crt", "/usr/local/share/ca-certificates/https-server.key", nil);
	err := s.ListenAndServe()

	if err != nil {
		log.Fatal(err)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Info("Shutdown signal received, exiting...")

	s.Shutdown(context.Background())
}
