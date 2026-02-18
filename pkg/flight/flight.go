package flight

import (
	"sync"
	"sync/atomic"
	"time"
	"weak"
)

// Cache provides a generic, concurrency-safe cache that combines "singleflight"
// request coalescing with a hybrid strong/weak reference expiration policy.
//
// It ensures that only one execution of the work function happens per key
// at a time (coalescing). Cached values are held via a strong reference
// for a configurable duration (TTL). Once the TTL expires, the cache downgrade
// to a weak reference, allowing the Go Garbage Collector to reclaim the memory
// if strictly necessary, while still serving the value if it remains in memory.
type Cache[K comparable, V any] struct {
	// finished holds completed results. Each entry keeps a strong reference
	// until its deadline passes, after which only the weak pointer remains.
	finished map[K]*entry[V]
	fmu      *sync.RWMutex

	pending map[K]*job[V]
	pmu     *sync.Mutex

	work func(K) (V, error)

	// ttl stores the strong-hold duration in nanoseconds.
	// <= 0 means infinite (never drop the strong reference).
	ttl *atomic.Int64
}

type entry[V any] struct {
	w        weak.Pointer[V]
	strong   *V        // non-nil while within the strong-hold window
	deadline time.Time // zero => infinite
}

type job[V any] struct {
	val  V
	err  error
	done chan struct{}
}

// NewCache creates a new Cache instance with a default strong-hold TTL of 1 hour.
//
// work is the function used to retrieve data when it is not present in the cache.
// This function will be called concurrently if multiple keys are requested,
// but only once per specific key at a time.
func NewCache[K comparable, V any](work func(K) (V, error)) Cache[K, V] {
	var ttl atomic.Int64
	ttl.Store(int64(time.Hour))
	return Cache[K, V]{
		finished: make(map[K]*entry[V]),
		fmu:      new(sync.RWMutex),
		pending:  make(map[K]*job[V]),
		pmu:      new(sync.Mutex),
		work:     work,
		ttl:      &ttl,
	}
}

// Expiry sets the duration for which the cache maintains a strong reference to values.
//
// If d > 0, values are strongly held for duration d, after which they become
// candidates for garbage collection (weak references).
// If d <= 0, the cache maintains a strong reference indefinitely (never GC'd).
func (p *Cache[K, V]) Expiry(d time.Duration) {
	if d <= 0 {
		p.ttl.Store(0)
		return
	}
	p.ttl.Store(int64(d))
}

// Get retrieves the value for the given key.
//
// 1. If the value is cached and strongly held, it returns immediately.
// 2. If the value is weakly held (expired but not GC'd), it is promoted and returned.
// 3. If the value is missing or GC'd, it initiates the 'work' function.
//
// Concurrent calls for the same key join the same "in-flight" request (singleflight),
// ensuring the work function is executed only once per key.
func (p *Cache[K, V]) Get(k K) (V, error) {
	// Try finished (with lazy cleanup) and coalesce concurrent work.
	p.pmu.Lock()

	// Fast path: check finished.
	if e, ok := p.loadEntry(k); ok {
		if v, ok := p.tryEntry(e); ok {
			p.pmu.Unlock()
			return v, nil
		}
		// If the weak value is gone, remove the entry so the miss below computes.
		p.fmu.Lock()
		if cur, ok := p.finished[k]; ok && cur == e && e.w.Value() == nil {
			delete(p.finished, k)
		}
		p.fmu.Unlock()
	}

	// Join existing in-flight job if any.
	if pending, ok := p.pending[k]; ok {
		p.pmu.Unlock()
		<-pending.done
		return pending.val, pending.err
	}

	// Create new job.
	j := &job[V]{done: make(chan struct{})}
	p.pending[k] = j
	p.pmu.Unlock()

	// Execute work.
	j.val, j.err = p.work(k)
	if j.err == nil {
		p.storeEntry(k, j.val)
	}

	// Complete job.
	p.pmu.Lock()
	close(j.done)
	delete(p.pending, k)
	p.pmu.Unlock()

	return j.val, j.err
}

// Force bypasses the cache and executes the work function to refresh the value.
//
// If a job for this key is currently in-flight, Force waits for it to complete,
// then immediately starts a new job to ensure freshness.
func (p *Cache[K, V]) Force(k K) (V, error) {
	var j *job[V]
	for {
		p.pmu.Lock()
		if existing, ok := p.pending[k]; ok {
			p.pmu.Unlock()
			<-existing.done
			continue
		}
		newJob := &job[V]{done: make(chan struct{})}
		p.pending[k] = newJob
		j = newJob
		p.pmu.Unlock()
		break
	}

	j.val, j.err = p.work(k)
	if j.err == nil {
		p.storeEntry(k, j.val)
	}

	p.pmu.Lock()
	close(j.done)
	delete(p.pending, k)
	p.pmu.Unlock()

	return j.val, j.err
}

// Work directly executes the configured work function for the key, ignoring
// caching and singleflight mechanics completely.
func (p *Cache[K, V]) Work(k K) (V, error) {
	return p.work(k)
}

// Delete removes the key from the cache entirely.
// Subsequent calls to Get will trigger a new fetch.
func (p *Cache[K, V]) Delete(k K) {
	p.fmu.Lock()
	delete(p.finished, k)
	p.fmu.Unlock()
}

// Expire manually forces the immediate expiration of the strong reference
// for the given key. The value remains in the cache as a weak reference
// (if not yet garbage collected), but the strong hold is released.
func (p *Cache[K, V]) Expire(k K) {
	p.fmu.Lock()
	if e, ok := p.finished[k]; ok {
		e.strong = nil
		// Set deadline to a past time to ensure loadEntry logic sees it as expired
		// should it check the time, though strong=nil is the primary mechanism.
		e.deadline = time.Unix(0, 1)
	}
	p.fmu.Unlock()
}

// Reset clears all cached results.
//
// In-flight requests (pending jobs) are unaffected, but their results
// will repopulate the empty cache when they complete.
func (p *Cache[K, V]) Reset() {
	p.fmu.Lock()
	// Create a new map to instantly clear all references.
	p.finished = make(map[K]*entry[V])
	p.fmu.Unlock()
}

// Set manually sets a value in the cache, useful for when the value is generated outside the work function.
func (p *Cache[K, V]) Set(k K, val V) {
	p.storeEntry(k, val)
}

// --- internals ---

func (p *Cache[K, V]) ttlDur() time.Duration {
	return time.Duration(p.ttl.Load())
}

func (p *Cache[K, V]) loadEntry(k K) (*entry[V], bool) {
	p.fmu.RLock()
	e, ok := p.finished[k]
	p.fmu.RUnlock()
	if !ok {
		return nil, false
	}

	// If the strong-hold window elapsed, drop the strong pointer.
	if !e.deadline.IsZero() && time.Now().After(e.deadline) {
		p.fmu.Lock()
		// Re-check under write lock to avoid racing another dropper.
		if cur, ok := p.finished[k]; ok && cur == e && e.strong != nil && time.Now().After(e.deadline) {
			e.strong = nil
		}
		p.fmu.Unlock()
	}
	return e, true
}

func (p *Cache[K, V]) tryEntry(e *entry[V]) (V, bool) {
	if vp := e.w.Value(); vp != nil {
		return *vp, true
	}
	var zero V
	return zero, false
}

func (p *Cache[K, V]) storeEntry(k K, val V) {
	// Allocate a dedicated heap cell so the weak pointer refers to a stable address.
	v := new(V)
	*v = val

	e := &entry[V]{w: weak.Make(v)}
	if d := p.ttlDur(); d > 0 {
		e.deadline = time.Now().Add(d)
		e.strong = v // keep alive until deadline
	} else {
		// Infinite duration => keep strong ref permanently.
		e.deadline = time.Time{}
		e.strong = v
	}

	p.fmu.Lock()
	p.finished[k] = e
	p.fmu.Unlock()
}
