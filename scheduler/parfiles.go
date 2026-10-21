package scheduler

import "sync"

// RunParallelFiles processes multiple images in parallel. Every image is
// handled start to finish by a single goroutine that takes it from a shared
// queue.
func RunParallelFiles(config Config) {
	defer reportTimings()

	queue := NewTaskQueue(loadTasks(config))
	workers := config.ThreadCount
	if workers > queue.Len() {
		workers = queue.Len()
	}

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for {
				task, ok := queue.Pop()
				if !ok {
					return
				}
				processTask(task, 1)
			}
		}()
	}
	wg.Wait()
}
