package apollo

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/apolloconfig/agollo/v4"
	configuration "github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/storage"
	"github.com/nexitf/unit/plugin"
)

var (
	ErrPluginNotInited          = errors.New("plugin not inited, see 'github.com/nexitf/unit-plugin-apollo/apollo.Init()'")
	ErrInvalidUpdater           = errors.New("invalid updater")
	ErrUnrecognizedVariableType = errors.New("unrecognized variable type")
	ErrUnrecognizedBindOption   = errors.New("unrecognized bind option")
)

var (
	id   string
	plug *apolloPlugin
)

func init() {
	plug = &apolloPlugin{updaters: make(map[string]*configUpdater)}
	// Register plugin.
	id = plugin.Register(plug)
}

type ConfigBase struct {
}

// PluginID implements plugin.Resource.
func (base *ConfigBase) PluginID() string {
	return id
}

// Bind implements Config.
func (base *ConfigBase) Bind(opts ...plugin.BindOption) (unused []plugin.BindOption) {
	return opts
}

type Config interface {
	PluginID() string
	Bind(opts ...plugin.BindOption) (unused []plugin.BindOption)
	Update(value string) (err error)
}

// Config base methods
type configBase struct {
}

// PluginID implements plugin.Type.
func (base *configBase) PluginID() string {
	return id
}

// bind implements Config.
func (base *configBase) bind(opts ...plugin.BindOption) (unused []plugin.BindOption) {
	return opts
}

type config interface {
	PluginID() string
	bind(opts ...plugin.BindOption) (unused []plugin.BindOption)
	update(value string) (err error)
}

// WithVariableReady
func WithVariableReady(fn func()) plugin.BindOption {
	return func(varp plugin.Resource) (used bool) {
		up, used := varp.(*configUpdater)
		if used {
			up.readyFn = fn
		}
		return
	}
}

// WithVariableChange
func WithVariableChange(fn func()) plugin.BindOption {
	return func(varp plugin.Resource) (used bool) {
		up, used := varp.(*configUpdater)
		if used {
			up.changeFn = fn
		}
		return
	}
}

type configUpdater struct {
	configBase
	mutex    sync.RWMutex
	value    string
	uptime   time.Time
	varp     plugin.Resource
	bindFn   func(opts ...plugin.BindOption) (unused []plugin.BindOption)
	updateFn func(value string) (err error)
	readyFn  func()
	changeFn func()
	once     sync.Once
}

// Bind implements plugin.Updater.
func (up *configUpdater) Bind(varp plugin.Resource, opts ...plugin.BindOption) {
	if varp.PluginID() != id {
		panic(ErrUnrecognizedVariableType)
	}
	switch vp := varp.(type) {
	case config:
		up.bindFn = vp.bind
		up.updateFn = vp.update
	case Config:
		up.bindFn = vp.Bind
		up.updateFn = vp.Update
	default:
		panic(ErrUnrecognizedVariableType)
	}
	// Bind options
	unused := up.bind(opts...)
	if len(unused) > 0 {
		unused = up.bindFn(unused...)
	}
	if len(unused) > 0 {
		panic(ErrUnrecognizedBindOption)
	}
	up.varp = varp
}

// Snapshot implements plugin.Updater.
func (up *configUpdater) Snapshot() (snapshot plugin.Snapshot) {
	up.mutex.RLock()
	defer up.mutex.RUnlock()
	// Load snapshot
	snapshot.Time = up.uptime
	snapshot.Data = up.value
	return
}

// bind
func (up *configUpdater) bind(opts ...plugin.BindOption) (unused []plugin.BindOption) {
	for _, setOpt := range opts {
		if !setOpt(up) {
			unused = append(unused, setOpt)
		}
	}
	return
}

// update
func (up *configUpdater) update(value string) (err error) {
	up.mutex.Lock()
	defer up.mutex.Unlock()
	// Update
	err = up.updateFn(value)
	if err == nil {
		up.value = value
		up.uptime = time.Now()
		if up.readyFn != nil {
			up.once.Do(up.readyFn)
		}
		if up.changeFn != nil {
			up.changeFn()
		}
	}
	return
}

// stop
func (up *configUpdater) stop() {
}

type apolloPlugin struct {
	updaters map[string]*configUpdater
	conf     *configuration.AppConfig
	client   agollo.Client
}

// Name implements plugin.Plugin.
func (plug *apolloPlugin) Name() string {
	return "NexITF apollo plugin"
}

// About implements plugin.Plugin.
func (plug *apolloPlugin) About() (about plugin.About) {
	about.Name = plug.Name()
	about.Version = "1.0.0"
	about.Author = "Kami"
	about.Package = "github.com/nexitf/unit-plugin-apollo"
	return
}

// Init
func (plug *apolloPlugin) Init(conf *configuration.AppConfig) {
	plug.conf = conf
}

// Init inits the plugin.
func Init(conf *configuration.AppConfig) {
	plug.Init(conf)
}

// Run implements plugin.Plugin.
func (plug *apolloPlugin) Run(ctx context.Context) (err error) {
	if plug.conf == nil {
		return ErrPluginNotInited
	}
	// Start apollo
	client, err := agollo.StartWithConfig(func() (*configuration.AppConfig, error) {
		return plug.conf, nil
	})
	if err != nil {
		return
	}
	plug.client = client
	// Watch
	plug.client.AddChangeListener(plug)
	return
}

// Stop implements plugin.Plugin.
func (plug *apolloPlugin) Stop(ctx context.Context) (err error) {
	if plug.client == nil {
		return ErrPluginNotInited
	}
	for _, up := range plug.updaters {
		up.stop()
	}
	plug.client.RemoveChangeListener(plug)
	// CLose apollo
	plug.client.Close()
	return
}

// Bind implements plugin.Plugin.
func (plug *apolloPlugin) Bind(ctx context.Context, name string, updater plugin.Updater) (err error) {
	up, ok := updater.(*configUpdater)
	if !ok {
		return ErrInvalidUpdater
	}
	value, err := plug.client.GetConfigCache(plug.conf.NamespaceName).Get(name)
	if err != nil {
		return nil
	} else {
		up.update(value.(string))
	}
	plug.updaters[name] = up
	return
}

// NewUpdater implements plugin.Plugin.
func (plug *apolloPlugin) NewUpdater() (updater plugin.Updater) {
	return new(configUpdater)
}

// OnChange implements storage.ChangeListener.
func (plug *apolloPlugin) OnChange(event *storage.ChangeEvent) {
	for key, value := range event.Changes {
		if up, ok := plug.updaters[key]; ok {
			up.update(value.NewValue.(string))
		}
	}
}

// OnNewestChange implements storage.ChangeListener.
func (plug *apolloPlugin) OnNewestChange(event *storage.FullChangeEvent) {
}
