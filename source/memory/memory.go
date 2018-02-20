package memory

import (
	"crypto/md5"
	"fmt"
	"sync"
	"time"

	"github.com/micro/go-config/source"
	"github.com/pborman/uuid"
)

type Source struct {
	sync.RWMutex
	ChangeSet *source.ChangeSet
	Watchers  map[string]*Watcher
}

func (s *Source) Read() (*source.ChangeSet, error) {
	s.RLock()
	cs := &source.ChangeSet{
		Timestamp: s.ChangeSet.Timestamp,
		Data:      s.ChangeSet.Data,
		Checksum:  s.ChangeSet.Checksum,
		Source:    s.ChangeSet.Source,
	}
	s.RUnlock()
	return cs, nil
}

func (s *Source) Watch() (source.Watcher, error) {
	w := &Watcher{
		Id:      uuid.NewUUID().String(),
		Updates: make(chan *source.ChangeSet, 100),
		Source:  s,
	}

	s.Lock()
	s.Watchers[w.Id] = w
	s.Unlock()
	return w, nil
}

// Update allows manual updates of the config data.
func (s *Source) Update(data []byte) {
	// hash the file
	h := md5.New()
	h.Write(data)
	checksum := fmt.Sprintf("%x", h.Sum(nil))

	s.Lock()
	// update changeset
	s.ChangeSet = &source.ChangeSet{
		Timestamp: time.Now(),
		Data:      data,
		Checksum:  checksum,
		Source:    "memory",
	}

	// update watchers
	for _, w := range s.Watchers {
		select {
		case w.Updates <- s.ChangeSet:
		default:
		}
	}
	s.Unlock()
}

func (s *Source) String() string {
	return "memory"
}

func NewSource(opts ...source.Option) *Source {
	var options source.Options
	for _, o := range opts {
		o(&options)
	}

	var data []byte

	if options.Context != nil {
		d, ok := options.Context.Value(dataKey{}).([]byte)
		if ok {
			data = d
		}
	}

	s := &Source{
		Watchers: make(map[string]*Watcher),
	}
	s.Update(data)
	return s
}
