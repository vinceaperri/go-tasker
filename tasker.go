package tasker

import (
	"errors"
	"fmt"
	"sync"
)

// A Task is a function called with no arguments that returns an error. If
// variable information is required, consider providing a closure.
type Task func() error

// task info holds run-time information related to a task identified by
// task_info.name.
type task_info struct {
	task Task        // The task itself.
	done bool        // Prevents running a task more than once.
	err  error       // Stores error on failure.
	mux  *sync.Mutex // Controls access to this task.

	// Elements used in Tarjan's Algorithm.
	index   int
	lowlink int
}

func (ti *task_info) lock() {
	ti.mux.Lock()
}

func (ti *task_info) unlock() {
	ti.mux.Unlock()
}

func new_task_info(task Task) *task_info {
	return &task_info{task, false, nil, &sync.Mutex{}, -1, -1}
}

type Tasker struct {
	// Map of task_info's indexed by task name.
	tis map[string]*task_info

	// Map of tasks names their dependencies. Its keys are identical to tis'.
	deps map[string][]string

	// Semaphore implemented as a boolean channel. See wait and signal.
	semaphore chan bool
}

// wait signals that a task is running and blocks until it may be run.
func (tr *Tasker) wait() {
	tr.semaphore <- true
}

// signal signals that a task is done.
func (tr *Tasker) signal() {
	<-tr.semaphore
}

// NewTasker returns a new Tasker that will run up to n number of tasks
// simultaneously.
//
// Returns an error if n is not positive.
func NewTasker(n int) (*Tasker, error) {
	if n <= 0 {
		return nil, fmt.Errorf("n must be positive: %d", n)
	}
	tr := &Tasker{
		make(map[string]*task_info),
		make(map[string][]string),
		make(chan bool, n),
	}
	return tr, nil
}

// Add adds a task to call a function. name is the unique name for the task.
// deps is a list of the names of tasks to run before this the one being added.

// Any tasks specified in deps must be added before the Tasker can be run. deps
// may be nil, but task may not.
//
// An error is returned if name is not unique.
func (tr *Tasker) Add(name string, deps []string, task Task) error {
	if _, ok := tr.tis[name]; ok {
		return fmt.Errorf("task already added: %s", name)
	}

	// Prevent the basic cyclic dependency of one from occuring.
	for _, dep := range deps {
		if name == dep {
			return errors.New("task must not add itself as a dependency")
		}
	}

	tr.tis[name] = new_task_info(task)
	tr.deps[name] = deps
	return nil
}

func (tr *Tasker) strong_connect(name string, index *int, S *string_stack) {
	ti := tr.tis[name]
	ti.index = *index
	ti.lowlink = *index
	*index++
	S.push(name)
}

// verify returns an error if any task dependencies haven't been added or any
// cycles exist among the tasks.
func (tr *Tasker) verify() error {
	for name, deps := range tr.deps {
		for _, dep := range deps {
			if _, ok := tr.tis[dep]; !ok {
				return NewDepNotFoundError(name, dep)
			}
		}
	}

	cycles := tarjan_algo(tr.deps)
	if len(cycles) > 0 {
		return CycleError(cycles)
	}

	return nil
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
func (tr *Tasker) runTask(name string, err_ch chan error) {
	ti := tr.tis[name]

	ti.lock()
	defer ti.unlock()

	// Don't run this task if it has been handled by another goroutine and send
	// its error, which may be an error from running the task itself or from
	// running one of its dependencies.
	if ti.done {
		err_ch <- ti.err
		return
	}

	// Set this task to done.
	ti.done = true

	// Run all dependencies first. Do not continue with the current task if one
	// fails. If that happens, this task will inherit its error from the first
	// one that failed.
	deps := tr.deps[name]
	dep_err_ch := make(chan error)
	for _, dep := range deps {
		go tr.runTask(dep, dep_err_ch)
	}
	for _ = range deps {
		ti.err = <-dep_err_ch

		// Do not run this task if one of its dependencies fail.
		if ti.err != nil {
			err_ch <- ti.err
			return
		}
	}

	// Limit the number of consecutive tasks.
	tr.wait()
	defer tr.signal()

	ti.err = ti.task()
	err_ch <- ti.err
}

// Run runs all tasks registered through Add in parallel. It should only be
// called once. Invokations after the first are simpley expensive no-ops.
//
// All tasks are only run once, even if two or more other tasks depend on it.
// A task will not run if any dependency fails.
func (tr *Tasker) Run() error {
	if err := tr.verify(); err != nil {
		return err
	}

	err_ch := make(chan error)
	for name, _ := range tr.tis {
		go tr.runTask(name, err_ch)
	}

	// Wait for all tasks to finish. Return the first error encountered.
	var err error
	for _ = range tr.tis {
		e := <-err_ch
		if err == nil {
			err = e
		}
	}
	return err
}
