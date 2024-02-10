package executils

import "sync"

func BatchExec[T any](vals []T, fn func(T)) {
	batchSize := len(vals)
	var wg sync.WaitGroup
	workerPool := make(chan struct{}, batchSize)
	defer close(workerPool)
	for i := 0; i < batchSize; i++ {
		workerPool <- struct{}{}
	}

	idxCh := make(chan int, batchSize)
	defer close(idxCh)

	go func() {
		for idx := range idxCh {
			<-workerPool

			go func(idx int) {
				defer func() {
					workerPool <- struct{}{}
					wg.Done()
				}()
				fn(vals[idx])
			}(idx)
		}
	}()

	for idx := range vals {
		wg.Add(1)
		idxCh <- idx
	}

	wg.Wait()
}
