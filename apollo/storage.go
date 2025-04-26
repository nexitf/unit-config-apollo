package apollo

import (
	"errors"
	"sync"

	agollo "github.com/apolloconfig/agollo/v4"
	config "github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/storage"
)

const (
	defaultServer    = "http://127.0.0.1:8080"
	defaultCluster   = "default"
	defaultNamespace = "application"
)

var (
	ErrDuplicateKeyName = errors.New("duplicate key name")
)

type Option func(*config.AppConfig)

// WithCluster
func WithCluster(cluster string) Option {
	return func(conf *config.AppConfig) { conf.Cluster = cluster }
}

// WithNamespace
func WithNamespace(namespace string) Option {
	return func(conf *config.AppConfig) { conf.NamespaceName = namespace }
}

// WithServer
func WithServer(server string) Option {
	return func(conf *config.AppConfig) { conf.IP = server }
}

// WithSecret
func WithSecret(secret string) Option {
	return func(conf *config.AppConfig) { conf.Secret = secret }
}

// WithLabel
func WithLabel(label string) Option {
	return func(conf *config.AppConfig) { conf.Label = label }
}

// WithMustStart
func WithMustStart() Option {
	return func(conf *config.AppConfig) { conf.MustStart = true }
}

// WithSyncServerTimeout
func WithSyncServerTimeout(seconds int) Option {
	return func(conf *config.AppConfig) { conf.SyncServerTimeout = seconds }
}

// WithBackup
func WithBackup(backup bool) Option {
	return func(conf *config.AppConfig) { conf.IsBackupConfig = backup }
}

// WithBackupWithPath
func WithBackupWithPath(backup bool, path string) Option {
	return func(conf *config.AppConfig) { conf.IsBackupConfig, conf.BackupConfigPath = backup, path }
}

type Apollo struct {
	mutex   sync.Mutex
	storage map[string]func(value string)
	client  agollo.Client
	config  *config.AppConfig
}

// NewApolloStorage
func NewApolloStorage(appID string, opts ...Option) (a *Apollo, err error) {
	a = &Apollo{
		config:  new(config.AppConfig),
		storage: make(map[string]func(value string)),
	}
	// Set options
	for _, setOpt := range opts {
		setOpt(a.config)
	}

	// Option: config
	a.config.AppID = appID
	if a.config.Cluster == "" {
		a.config.Cluster = defaultCluster
	}
	if a.config.NamespaceName == "" {
		a.config.NamespaceName = defaultNamespace
	}
	if a.config.IP == "" {
		a.config.IP = defaultServer
	}

	a.client, err = agollo.StartWithConfig(func() (*config.AppConfig, error) {
		return a.config, nil
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
	value, err := a.client.GetConfigCache(a.config.NamespaceName).Get(name)
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
