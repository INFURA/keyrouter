package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/pkg/errors"

	"github.com/INFURA/keyrouter/consistent"
)

type serviceEntry struct {
	Name     string
	hashRing *consistent.HashRing
}

type server struct {
	mu       sync.RWMutex
	services map[string]serviceEntry
}

// New returns a new instance of the server
func New() *server {
	return &server{
		services: make(map[string]serviceEntry),
	}
}

// PopulateService adds member nodes to the service indetified by name
func (s *server) PopulateService(name string, nodes consistent.Members) (added consistent.Members, removed consistent.Members, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.services[name]; !ok {
		s.services[name] = serviceEntry{
			Name:     name,
			hashRing: consistent.NewHashRing(),
		}
	}

	added, removed, err = s.services[name].hashRing.Set(nodes)
	if err != nil {
		return nil, nil, errors.Wrap(err, "populate service failed")
	}
	return
}

// Handler returns an http.HandlerFunc which implements the server logic
func (s *server) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceName := r.URL.Path
		s.mu.RLock()
		service, ok := s.services[serviceName]
		s.mu.RUnlock()
		if !ok {
			http.Error(w, "unrecognized service: "+serviceName, http.StatusNotFound)
			return
		}

		// min, max, key arguments all required
		type Args struct {
			Key string
			Min int
			Max int
		}

		parse := func() (*Args, error) {
			args := Args{}

			if r.Header.Get("Content-Type") == "application/json" {
				if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
					return nil, err
				}
				return &args, nil
			}
			args.Key = r.FormValue("key")
			min, max := r.FormValue("min"), r.FormValue("max")

			var err error
			if args.Min, err = strconv.Atoi(min); err != nil {
				return nil, err
			}
			if args.Max, err = strconv.Atoi(max); err != nil {
				return nil, err
			}
			return &args, nil
		}

		args, err := parse()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if args.Min > args.Max {
			// 400
			http.Error(w, "max cannot be less than min", http.StatusBadRequest)
			return
		}

		for i := args.Max; i >= args.Min; i-- {
			members, e := service.hashRing.Get(args.Key, i)
			if e != nil {
				err = e
				continue
			}

			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(members)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
			return
		}

		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}
