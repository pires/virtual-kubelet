package scheduler

import (
	"sync"

	"github.com/mesos/mesos-go/api/v1/lib"

	corev1 "k8s.io/api/core/v1"
)

type mesosPod struct {
	pod      *corev1.Pod
	agentId  *mesos.AgentID
	executor *mesos.ExecutorInfo
	tasks    []mesos.TaskInfo
}

type MesosPodMap struct {
	items map[string]*mesosPod
	mutex sync.RWMutex
}

// Creates a new concurrent map.
func NewMesosPodMap() *MesosPodMap {
	return &MesosPodMap{
		items: make(map[string]*mesosPod),
		mutex: sync.RWMutex{},
	}
}

// Sets the given value under the specified key.
func (this *MesosPodMap) Set(key string, value *mesosPod) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.items[key] = value
}

// Removes an element from the map.
func (this *MesosPodMap) Remove(key string) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	delete(this.items, key)
}

// Retrieves an element from map under given key.
func (this *MesosPodMap) Get(key string) (*mesosPod, bool) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	v, ok := this.items[key]
	return v, ok
}

// Retrieves an element from map under given key.
// If it exists, removes it from map.
func (this *MesosPodMap) GetAndRemove(key string) (*mesosPod, bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	v, ok := this.items[key]
	if ok {
		delete(this.items, key)
	}
	return v, ok
}

// Removes an element from the map by key and value.
func (this *MesosPodMap) RemoveWithValue(key string, value *mesosPod) (*mesosPod, bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	v, ok := this.items[key]
	if ok && v == value {
		delete(this.items, key)
	}
	return v, ok
}

// Iteratively removes all elements from the map that contain a value.
func (this *MesosPodMap) IterRemoveWithValue(value *mesosPod) uint {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	var counter uint = 0
	for k, v := range this.items {
		if v == value {
			delete(this.items, k)
			counter++
		}
	}

	return counter
}

// Returns the number of elements within the map.
func (this *MesosPodMap) Count() int {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	return len(this.items)
}

// Looks up an item under specified key
func (this *MesosPodMap) Has(key string) bool {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	// See if element is within shard.
	_, ok := this.items[key]
	return ok
}

// Checks if map is empty.
func (this *MesosPodMap) IsEmpty() bool {
	return this.Count() == 0
}

// Returns a slice containing all map keys
func (this *MesosPodMap) Keys() []string {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	keys := make([]string, 0, len(this.items))
	for k, _ := range this.items {
		keys = append(keys, k)
	}
	return keys
}

// Returns a slice containing all map values
func (this *MesosPodMap) Values() []*mesosPod {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	values := make([]*mesosPod, 0, len(this.items))
	for _, v := range this.items {
		values = append(values, v)
	}
	return values
}

// Returns a <strong>snapshot</strong> (copy) of current map items which could be used in a for range loop.
// One <strong>CANNOT</strong> change the contents of this map by means of this method, since it returns only a copy.
func (this *MesosPodMap) Iter() map[string]*mesosPod {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	results := make(map[string]*mesosPod, len(this.items))
	for k, v := range this.items {
		results[k] = v
	}

	return results
}
