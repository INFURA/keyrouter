package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/gorilla/mux"
	"github.com/hashicorp/logutils"
	"github.com/pelletier/go-toml"

	"github.com/INFURA/keyrouter/consistent"
	"github.com/INFURA/keyrouter/server"
)

type Config struct {
	Services []struct {
		Name  string
		Nodes consistent.Members
	}
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	app      = kingpin.New("keyrouter", "A simple microservice for consistent hashing of service entries")
	conf     = app.Flag("services", "location of services.toml").Default("services.toml").String()
	bind     = app.Flag("address", "address to bind to").Default(":8080").TCP()
	logLevel = app.Flag("log-level", "minimum log level").Default("INFO").String()
)

func main() {
	app.Version(version + "-" + commit).Author("INFURA")
	_ = kingpin.MustParse(app.Parse(os.Args[1:]))
	address := (*bind).String()
	router := mux.NewRouter()
	srv := server.New()

	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"SPAM", "DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel(*logLevel),
		Writer:   os.Stdout,
	}
	log.SetOutput(filter)

	loader := func() {
		b, err := ioutil.ReadFile(*conf)
		if err != nil {
			app.FatalUsage("cannot read config file: %v", err)
		}

		cfg := Config{}
		err = toml.Unmarshal(b, &cfg)
		if err != nil {
			log.Fatalf("[FATAL] error reading config: %v", err)
		}

		for _, s := range cfg.Services {
			log.Printf("[DEBUG] Populating Service %s", s.Name)
			added, removed, err := srv.PopulateService(s.Name, s.Nodes)
			if err != nil {
				log.Fatalf("[FATAL] error updating service %s: %v", s.Name, err)
			}
			if len(added) > 0 {
				log.Printf("[DEBUG] Added: %v", added)
			}
			if len(removed) > 0 {
				log.Printf("[DEBUG] Removed: %v", removed)
			}
		}
	}

	loader()

	hups := make(chan os.Signal, 1)
	signal.Notify(hups, syscall.SIGHUP)

	go func() {
		for range hups {
			log.Printf("[INFO] Reloading config")
			loader()
		}
	}()

	router.PathPrefix("/service/").Handler(http.StripPrefix("/service/", srv.Handler()))

	log.Printf("[INFO] Ready to serve at %v", address)
	s := http.Server{
		Addr:         address,
		ReadTimeout:  125 * time.Second,
		WriteTimeout: 125 * time.Second,
		Handler:      router,
	}
	err := s.ListenAndServe()
	if err != nil {
		log.Fatalf("[FATAL] %v", err)
	}
}
