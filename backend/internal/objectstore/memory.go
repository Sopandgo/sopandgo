package objectstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

// Memory is an in-memory Store for tests in any package. Its failure
// switches make the next calls fail until they are switched off again.
type Memory struct {
	root string

	mu         sync.Mutex
	objects    map[string]memObject
	lock       bool
	failPut    error
	failList   error
	failDelete error
}

type memObject struct {
	body         []byte
	lastModified time.Time
}

// NewMemory returns an empty store whose Root is root.
func NewMemory(root string) *Memory {
	return &Memory{root: strings.Trim(root, "/"), objects: map[string]memObject{}}
}

// Opener returns an Opener that always opens m.
func (m *Memory) Opener() Opener {
	return func(context.Context) (Store, error) { return m, nil }
}

// FailPut makes Put return err; nil switches it off. Likewise FailList and
// FailDelete.
func (m *Memory) FailPut(err error)    { m.mu.Lock(); m.failPut = err; m.mu.Unlock() }
func (m *Memory) FailList(err error)   { m.mu.Lock(); m.failList = err; m.mu.Unlock() }
func (m *Memory) FailDelete(err error) { m.mu.Lock(); m.failDelete = err; m.mu.Unlock() }

// SetObjectLock sets what ObjectLockEnabled answers.
func (m *Memory) SetObjectLock(on bool) { m.mu.Lock(); m.lock = on; m.mu.Unlock() }

// SetLastModified changes an object's last-modified time, for retention
// tests. It fails when there is no such object.
func (m *Memory) SetLastModified(key string, t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.objects[key]
	if !ok {
		return fmt.Errorf("no object %s", key)
	}
	o.lastModified = t
	m.objects[key] = o
	return nil
}

// Keys returns the keys under prefix, sorted.
func (m *Memory) Keys(prefix string) []string {
	objs, _ := m.list(prefix)
	keys := make([]string, len(objs))
	for i, o := range objs {
		keys[i] = o.Key
	}
	return keys
}

func (m *Memory) Root() string { return m.root }

func (m *Memory) Put(_ context.Context, key string, body io.Reader, _ string) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failPut != nil {
		return m.failPut
	}
	m.objects[key] = memObject{body: b, lastModified: time.Now().UTC()}
	return nil
}

func (m *Memory) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.objects[key]
	if !ok {
		return nil, errors.New("not found")
	}
	b := o.body
	if len(b) > MaxGetSize {
		b = b[:MaxGetSize]
	}
	return append([]byte(nil), b...), nil
}

func (m *Memory) List(_ context.Context, prefix string) ([]Object, error) {
	m.mu.Lock()
	failList := m.failList
	m.mu.Unlock()
	if failList != nil {
		return nil, failList
	}
	return m.list(prefix)
}

func (m *Memory) list(prefix string) ([]Object, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var objs []Object
	for k, o := range m.objects {
		if strings.HasPrefix(k, prefix) {
			objs = append(objs, Object{Key: k, LastModified: o.lastModified})
		}
	}
	sort.Slice(objs, func(i, j int) bool { return objs[i].Key < objs[j].Key })
	return objs, nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failDelete != nil {
		return m.failDelete
	}
	delete(m.objects, key)
	return nil
}

func (m *Memory) ObjectLockEnabled(context.Context) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lock, nil
}
