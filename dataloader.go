package dataloader

import (
	"fmt"
	"sync"
)

type BatchLoadFn func(keys []interface{}) ([]interface{}, error)

type CacheMap interface {
	Get(key interface{}) interface{}
	Set(key interface{}, value interface{})
	Delete(key interface{})
	Clear()
}

type Options struct {
	Batch      bool
	Cache      bool
	CacheKeyFn func(key interface{}) interface{}
	CacheMap   CacheMap
}

type DataLoader struct {
	mutex       *sync.Mutex
	batchLoadFn BatchLoadFn
	options     *Options
	futureCache CacheMap
	queue       *LoaderQueue
}

func NewDataLoader(fn BatchLoadFn, options *Options) *DataLoader {
	return &DataLoader{
		mutex:       &sync.Mutex{},
		batchLoadFn: fn,
		options:     options,
		futureCache: newMap(),
		queue:       newLoaderQueue(),
	}
}

/**
 * Loads a key, returning a `Future` for the value represented by that key.
 */
func (d *DataLoader) Load(key interface{}) (interface{}, error) {
	if key == nil {
		return nil, fmt.Errorf("The Load() function must be called with a value but got: %s", key)
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Determine options
	shouldBatch := d.options == nil || d.options.Batch == true
	shouldCache := d.options == nil || d.options.Cache == true
	cacheKey := key
	if d.options != nil && d.options.CacheKeyFn != nil {
		cacheKey = d.options.CacheKeyFn(key)
	}

	// If caching and there is a cache-hit, return cached Future.
	if shouldCache {
		cachedFuture := d.futureCache.Get(cacheKey)
		if cachedFuture != nil {
			return cachedFuture.(Future)()
		}
	}

	var resultValue interface{}
	var resultError error

	c := make(chan struct{}, 1)

	future := Future(func() (interface{}, error) {
		<-c
		return resultValue, resultError
	})

	// If caching, cache this future.
	if shouldCache {
		d.futureCache.Set(cacheKey, future)
	}

	d.queue.Push(LoaderQueueItem{
		Key: key,
		Resolve: func(value interface{}, err error) {
			defer close(c)
			resultValue = value
			resultError = err
		},
	})

	// Determine if a dispatch of this queue should be scheduled.
	// A single dispatch should be scheduled per queue at the time when the
	// queue changes from "empty" to "full".
	if d.queue.Size() == 1 {
		if shouldBatch {
			// If batching, schedule a task to dispatch the queue.
			go enqueuePostFutureJob(func() {
				dispatchQueue(d)
			})
		} else {
			// Otherwise dispatch the (queue of one) immediately.
			dispatchQueue(d)
		}
	}

	return future()
}

/**
* Loads multiple keys, promising an array of values:
 */

func (d *DataLoader) LoadMany(keys []interface{}) ([]interface{}, error) {
	return FutureAll(keys, d.Load)
}

/**
 * Clears the value at `key` from the cache, if it exists. Returns itself for
 * method chaining.
 */

func (d *DataLoader) Clear(key interface{}) {
	cacheKey := key
	if d.options != nil && d.options.CacheKeyFn != nil {
		cacheKey = d.options.CacheKeyFn(key)
	}
	d.futureCache.Delete(cacheKey)
}

/**
 * Clears the entire cache. To be used when some event results in unknown
 * invalidations across this particular `DataLoader`. Returns itself for
 * method chaining.
 */

func (d *DataLoader) ClearAll() {
	d.futureCache.Clear()
}

/**
 * Adds the provied key and value to the cache. If the key already exists, no
 * change is made. Returns itself for method chaining.
 */

func (d *DataLoader) Prime(key interface{}, value interface{}) {
	cacheKey := key
	if d.options != nil && d.options.CacheKeyFn != nil {
		cacheKey = d.options.CacheKeyFn(key)
	}

	// Only add the key if it does not already exist.
	if d.futureCache.Get(cacheKey) == nil {
		var future Future

		if err, ok := value.(error); ok {
			future = NewFutureError(err)
		} else {
			future = NewFutureValue(value)
		}

		d.futureCache.Set(cacheKey, future)
	}
}
