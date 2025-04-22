package apollo

import (
	"strconv"
	"strings"
	"sync"

	"github.com/nexitf/unit/plugin"
)

type Strings struct {
	base
	mutex sync.RWMutex
	value []string
}

// Get returns the value of the Strings.
func (s *Strings) Get() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.value
}

// bind
func (s *Strings) bind(opts ...plugin.BindOption) (unused []plugin.BindOption) {
	for _, setOpt := range opts {
		if !setOpt(s) {
			unused = append(unused, setOpt)
		}
	}
	return
}

// update updates the value of the String.
func (s *Strings) update(value string) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.value = strings.Split(value, ",")
	return
}

type Int64s struct {
	base
	mutex sync.RWMutex
	value []int64
}

// Get returns the value of the Int64s.
func (i *Int64s) Get() []int64 {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.value
}

// bind
func (i *Int64s) bind(opts ...plugin.BindOption) (unused []plugin.BindOption) {
	for _, setOpt := range opts {
		if !setOpt(i) {
			unused = append(unused, setOpt)
		}
	}
	return
}

// update updates the value of the String.
func (i *Int64s) update(value string) (err error) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	var vv []int64
	var ss = strings.Split(value, ",")
	for _, s := range ss {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		vv = append(vv, v)
	}
	i.value = vv
	return nil
}
