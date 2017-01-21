package service

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/aandryashin/selenoid/config"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
)

// Starter - interface to create session with cancellation ability
type Starter interface {
	StartWithCancel() (*url.URL, func(), error)
}

// Manager - interface to choose appropriate starter
type Manager interface {
	Find(s string, v *string) (Starter, bool)
}

// DefaultManager - struct for default implementation
type DefaultManager struct {
	IP        string
	Client    *client.Client
	Config    *config.Config
	LogConfig container.LogConfig
}

// Find - default implementation Manager interface
func (m *DefaultManager) Find(s string, v *string) (Starter, bool) {
	log.Printf("Locating the service for %s %s\n", s, *v)
	service, ok := m.Config.Find(s, v)
	if !ok {
		return nil, false
	}
	switch service.Image.(type) {
	case string:
		if m.Client == nil {
			return nil, false
		}
		log.Printf("Using docker service for %s %s\n", s, *v)
		return &Docker{m.IP, m.Client, service, m.LogConfig}, true
	case []interface{}:
		log.Printf("Using driver service for %s %s\n", s, *v)
		return &Driver{service}, true
	}
	return nil, false
}

func wait(u string, t time.Duration) error {
	done := make(chan struct{})
	go func() {
	loop:
		for {
			select {
			case <-time.After(50 * time.Millisecond):
				r, err := http.Head(u)
				if err == nil {
					r.Body.Close()
					done <- struct{}{}
				}
			case <-done:
				break loop
			}
		}
	}()
	select {
	case <-time.After(t):
		return fmt.Errorf("error: %s does not respond in %v", u, t)
	case <-done:
		close(done)
	}
	return nil
}
