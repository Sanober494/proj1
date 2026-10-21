package scheduler

// RunSequential processes all images one after another on a single thread.
func RunSequential(config Config) {
	defer reportTimings()

	for _, task := range loadTasks(config) {
		processTask(task, 1)
	}
}
