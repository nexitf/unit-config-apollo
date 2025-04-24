package apollo

import (
	"errors"
	"sync"

	agollo "github.com/apolloconfig/agollo/v4"
	configuration "github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/storage"
)

var (
	ErrDuplicateKeyName = errors.New("duplicate key name")
)

type Apollo struct {
	mutex   sync.Mutex
	storage map[string]func(value string)
	client  agollo.Client
	conf    *configuration.AppConfig
}

// NewApolloStorage
func NewApolloStorage(conf *configuration.AppConfig) (a *Apollo, err error) {
	a = &Apollo{
		conf:    conf,
		storage: make(map[string]func(value string)),
	}
	a.client, err = agollo.StartWithConfig(func() (*configuration.AppConfig, error) {
		return conf, nil
	})
	if err == nil {
		a.client.AddChangeListener(a)
	}
	return
}

// Name implements config.Storage.
func (a *Apollo) Name() string {
	return "Apollo"
}

// Version implements config.Storage.
func (a *Apollo) Version() string {
	return "1.0.0"
}

// Package implements config.Storage.
func (a *Apollo) Package() string {
	return "github.com/nexitf/unit-config-apollo"
}

// Watch implements config.Storage.
func (a *Apollo) Watch(name string, update func(value string)) (err error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if _, ok := a.storage[name]; ok {
		return ErrDuplicateKeyName
	}
	a.storage[name] = update
	value, err := a.client.GetConfigCache(a.conf.NamespaceName).Get(name)
	if err != nil {
		return nil
	} else {
		update(value.(string))
	}
	return
}

// Close
func (a *Apollo) Close() {
	a.client.RemoveChangeListener(a)
	a.client.Close()

}

// OnChange implements storage.ChangeListener.
func (a *Apollo) OnChange(event *storage.ChangeEvent) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	// Key changed
	for key, value := range event.Changes {
		if fn, ok := a.storage[key]; ok {
			fn(value.NewValue.(string))
		}
	}
}

// OnNewestChange implements storage.ChangeListener.
func (a *Apollo) OnNewestChange(event *storage.FullChangeEvent) {
}
