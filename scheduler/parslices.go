package scheduler

// RunParallelSlices processes one image at a time, but splits the work of
// every effect on that image across ThreadCount goroutines.
func RunParallelSlices(config Config) {
	defer reportTimings()

	queue := NewTaskQueue(loadTasks(config))
	for {
		task, ok := queue.Pop()
		if !ok {
			return
		}
		processTask(task, config.ThreadCount)
	}
}
