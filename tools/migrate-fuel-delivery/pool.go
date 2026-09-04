package main

import (
	"fmt"
	"sync"

	"gorm.io/gorm"
)

// Insert concurrency for large Django tables (drivers, trucks, transporters,
// ILO lines, comp lines, loadings, attachments, …). Some exceed 300k rows.
//
// Do not spawn one goroutine per row. That would open hundreds of thousands of
// MSSQL sessions (the pool maxes at 100), deadlock unique indexes
// (TransactionID, seals, CompartmentalizationID+TankID+Index), and still wait
// on the single pgx reader. The cost of the old loop was N+1 round-trips
// (idByDjango + snapshot First + Create), not the Go for-loop itself.
//
// Stages stay serial: parents must exist before children. Inside one table:
// one Postgres reader, in-memory DjangoID maps, bounded workers, CreateInBatches.
var (
	insertWorkers = 8
	insertBatch   = 200
)

func clampCopyPool(workers, batch int) (int, int) {
	if workers < 1 {
		workers = 1
	}
	if workers > 32 {
		workers = 32
	}
	if batch < 1 {
		batch = 1
	}
	if batch > 1000 {
		batch = 1000
	}
	return workers, batch
}

type workerPool[T any] struct {
	ch chan []T
	wg sync.WaitGroup
}

func startPool[T any](workers int, fn func([]T)) *workerPool[T] {
	if workers < 1 {
		workers = 1
	}
	p := &workerPool[T]{ch: make(chan []T, workers)}
	p.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer p.wg.Done()
			for batch := range p.ch {
				fn(batch)
			}
		}()
	}
	return p
}

func (p *workerPool[T]) submit(batch []T) {
	if p == nil || len(batch) == 0 {
		return
	}
	p.ch <- batch
}

func (p *workerPool[T]) wait() {
	if p == nil {
		return
	}
	close(p.ch)
	p.wg.Wait()
}

type batchWriter[T any] struct {
	name  string
	size  int
	buf   []T
	pool  *workerPool[T]
	added int
}

func newBatchWriter[T any](name string, workers, batch int, fn func([]T)) *batchWriter[T] {
	workers, batch = clampCopyPool(workers, batch)
	return &batchWriter[T]{
		name: name,
		size: batch,
		buf:  make([]T, 0, batch),
		pool: startPool(workers, fn),
	}
}

func (w *batchWriter[T]) add(row T) {
	w.buf = append(w.buf, row)
	w.added++
	if w.added%10000 == 0 {
		fmt.Printf("  %s queued %d …\n", w.name, w.added)
	}
	if len(w.buf) >= w.size {
		w.flush()
	}
}

func (w *batchWriter[T]) flush() {
	if len(w.buf) == 0 {
		return
	}
	w.pool.submit(w.buf)
	w.buf = make([]T, 0, w.size)
}

func (w *batchWriter[T]) close() int {
	w.flush()
	w.pool.wait()
	if w.added > 0 {
		fmt.Printf("  %s inserted %d\n", w.name, w.added)
	}
	return w.added
}

func gormInsert[T any](dest *gorm.DB, remember func([]T)) func([]T) {
	return func(batch []T) {
		if dest == nil || len(batch) == 0 {
			return
		}
		if err := dest.CreateInBatches(batch, len(batch)).Error; err != nil {
			var ok []T
			for i := range batch {
				if err := dest.Create(&batch[i]).Error; err != nil {
					continue
				}
				ok = append(ok, batch[i])
			}
			if remember != nil && len(ok) > 0 {
				remember(ok)
			}
			return
		}
		if remember != nil {
			remember(batch)
		}
	}
}
