package scheduler

// TaskQueue is a FIFO queue of image tasks whose removals are protected by a
// TAS lock, so that many goroutines can safely take tasks from it.
type TaskQueue struct {
	lock  TASLock
	tasks []ImageTask
	head  int // Index of the next task to hand out
}

// NewTaskQueue returns a queue that hands out tasks in the given order.
func NewTaskQueue(tasks []ImageTask) *TaskQueue {
	return &TaskQueue{tasks: tasks}
}

// Len returns the number of tasks that were put into the queue.
// It is only safe to call before any goroutine starts taking tasks.
func (q *TaskQueue) Len() int {
	return len(q.tasks)
}

// Pop removes and returns the next task. The boolean is false if the queue is
// empty.
func (q *TaskQueue) Pop() (ImageTask, bool) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.head == len(q.tasks) {
		return ImageTask{}, false
	}
	task := q.tasks[q.head]
	q.head++
	return task, true
}
