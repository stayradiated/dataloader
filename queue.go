package dataloader

import (
	"fmt"
	"sync"
	"time"
)

type LoaderQueueItem struct {
	Key     interface{}
	Resolve func(interface{}, error)
}

type LoaderQueue struct {
	mutex *sync.Mutex
	items []LoaderQueueItem
}

func newLoaderQueue() *LoaderQueue {
	return &LoaderQueue{
		mutex: &sync.Mutex{},
		items: make([]LoaderQueueItem, 0),
	}
}

func (q *LoaderQueue) Push(item LoaderQueueItem) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.items = append(q.items, item)
}

func (q *LoaderQueue) DumpItems() []LoaderQueueItem {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	items := q.items
	q.items = make([]LoaderQueueItem, 0)
	return items
}

func (q *LoaderQueue) Size() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return len(q.items)
}

func enqueuePostFutureJob(f func()) {
	<-time.After(time.Millisecond)
	f()
}

// Private: given the current state of a Loader instance, perform a batch load
// from its current queue.
func dispatchQueue(loader *DataLoader) {
	// Take the current loader queue, replacing it with an empty queue.
	items := loader.queue.DumpItems()

	// Collect all keys to be loaded in this dispatch
	keys := make([]interface{}, len(items))
	for i, item := range items {
		keys[i] = item.Key
	}

	// Call the provided batchLoadFn for this loader with the loader queue's keys.
	values, err := loader.batchLoadFn(keys)

	if err != nil {
		failedDispatch(loader, items, err)
		return
	}

	if len(values) != len(keys) {
		failedDispatch(loader, items, fmt.Errorf(
			"DataLoader must be constructed with a function which accepts "+
				"Array<key> and returns Future<Array<value>>, but the function did "+
				"not return a Future of an Array of the same length as the Array "+
				"of keys. \n\nKeys:\n%s \n\nValues:\n%s", keys, values))
		return
	}

	// Step through the values, resolving or rejecting each Future in the
	// loaded queue.
	for i, item := range items {
		switch value := values[i].(type) {
		case error:
			item.Resolve(nil, value)
		default:
			item.Resolve(value, nil)
		}
	}
}

// Private: do not cache individual loads if the entire batch dispatch fails,
// but still reject each request so they do not hang.
func failedDispatch(loader *DataLoader, items []LoaderQueueItem, err error) {
	for _, item := range items {
		loader.Clear(item.Key)
		item.Resolve(nil, err)
	}
}
