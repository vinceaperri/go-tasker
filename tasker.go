package tasker

import (
	"fmt"
	"sync"
)

type task struct {
	name  string       // Unique task identifier.
	deps  []*task      // Tasks that must succeed before this task is run.
	task  func() error // The task itself.
	done  bool         // Prevents running a task more than once.
	err   error        // Stores error on failure.
	mux   *sync.Mutex  // Controls access to this task.
}

func (t *task) lock() {
	t.mux.Lock()
}

func (t *task) unlock() {
	t.mux.Unlock()
}

type Tasker struct {
	tasks     map[string]*task // Map of tasks indexed by name.
	semaphore chan bool        // Semaphore implemented as a boolean channel.
}

// NewTasker returns a new Tasker that will run up to n number of tasks
// simultaneously.
//
// Returns an error if n is not positive.
func NewTasker(n int) (*Tasker, error) {
	if n <= 0 {
		return nil, fmt.Errorf("n must be positive: %d", n)
	}
	tr := &Tasker{make(map[string]*task), make(chan bool, n)}
	return tr, nil
}

// Add adds a task to call fn. deps is a list of task dependencies to run
// before this task and may be nil. They must have already been added to
// eliminate the possibility of a cyclic dependency among the tasks.
//
// An error is returned if a task with that same name has already been added
// or deps contains an unknown task.
func (tr *Tasker) Add(name string, deps []string, fn func() error) error {
	if _, ok := tr.tasks[name]; ok {
		return fmt.Errorf("task already added: %s", name)
	}
	task_deps := make([]*task, 0)
	for _, dep := range deps {
		t, ok := tr.tasks[dep]
		if !ok {
			return fmt.Errorf("task not found: %s", dep)
		}
		task_deps = append(task_deps, t)
	}
	tr.tasks[name] = &task{name, task_deps, fn, false, nil, &sync.Mutex{}}
	return nil
}

// wait signals that a task is running and blocks until it may be run.
func (tr *Tasker) wait() {
	tr.semaphore <- true
}

// signal signals that a task is done.
func (tr *Tasker) signal() {
	<-tr.semaphore
}

// runTask is called recursivley as a goroutine to run tasks in parallel. It
// runs all dependencies before running the provided task. The first error it
// encounters will be send through err_ch, be it from a dependency or the task
// itself. It will not run the provided task if any dependency fails.
//
// It initially takes the task's lock and sets a flag so that a task is not run
// in any other goroutine. Other goroutines will wait for the lock, then see
// that the task has already been executed, and return whatever error it had
// produced.
//
// It further limits the number of consecutive tasks as defined by the size of
// the Tasker's semaphore.
func (tr *Tasker) runTask(t *task, err_ch chan error) {
	t.lock()
	defer t.unlock()

	// Don't run this task if it has been handled by another goroutine and send
	// its error, which may be an error from running the task itself or from
	// running one of its dependencies.
	if t.done {
		err_ch <- t.err
		return
	}

	// Set this task as done.
	t.done = true

	// Run all dependencies first. Do not continue with the current task if one
	// fails. If that happens, this task will inherit its error from the first
	// one that failed.
	dep_err_ch := make(chan error)
	for _, dep := range t.deps {
		go tr.runTask(dep, dep_err_ch)
	}
	for _ = range t.deps {
		t.err = <-dep_err_ch

		// Do not run this task if one of its dependencies fail.
		if t.err != nil {
			err_ch <- t.err
			return
		}
	}

	// Limit the number of consecutive tasks.
	tr.wait()
	defer tr.signal()

	t.err = t.task()
	err_ch <- t.err
}

// Run runs all tasks registered through Add in parallel. It should only be
// called once. Invokations after the first are simple expensive no-ops.
//
// All tasks are only run once, even if two or more other tasks depend on it.
// A task will not run if any dependency fails.
func (tr *Tasker) Run() error {
	err_ch := make(chan error)
	for _, t := range tr.tasks {
		go tr.runTask(t, err_ch)
	}

	// Wait for all tasks to finish. Return the first error encountered.
	var err error
	for _ = range tr.tasks {
		e := <-err_ch
		if err == nil {
			err = e
		}
	}
	return err
}
