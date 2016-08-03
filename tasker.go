package tasker

import (
	"errors"
	"fmt"
	"sync"
)

type CycleError [][]string

func (ce CycleError) Error() string {
	msg := "tasker: "
	if len(ce) > 1 {
		msg += "cycles"
	} else {
		msg += "cycle"
	}
	msg += " detected: "
	for i, c := range ce {
		for j, e := range c {
			msg += e
			if j < len(c)-1 {
				msg += " -> "
			}
		}
		if i < len(ce)-1 {
			msg += ", "
		}
	}
	return msg
}


type DepNotFoundError struct {
	v string
	w string
}

func NewDepNotFoundError(v, w string) *DepNotFoundError {
	return &DepNotFoundError{v, w}
}

func (dnfe *DepNotFoundError) Error() string {
	return fmt.Sprintf("tasker: %s not found, required by %s", dnfe.w, dnfe.v)
}


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

	// Elements used in cycle detection.
	index    int
	lowlink  int
	on_stack bool
}

func (ti *task_info) lock() {
	ti.mux.Lock()
}

func (ti *task_info) unlock() {
	ti.mux.Unlock()
}

func new_task_info(task Task) *task_info {
	return &task_info{task, false, nil, &sync.Mutex{}, -1, -1, false}
}


type Tasker struct {
	// Map of task_info's indexed by task name.
	tis map[string]*task_info

	// Map of tasks names their dependencies. Its keys are identical to tis'.
	dep_graph map[string][]string

	// Semaphore implemented as a buffered boolean channel. May be nil.
	// See wait and signal.
	semaphore chan bool

	// Elements used in cycle detection.
	index int
	stack *string_stack
	cycles [][]string

	// Indicates whether Run has been called.
	was_run bool
}

// wait signals that a task is running and blocks until it may be run.
func (tr *Tasker) wait() {
	if tr.semaphore != nil {
		tr.semaphore <- true
	}
}

// signal signals that a task is done.
func (tr *Tasker) signal() {
	if tr.semaphore != nil {
		<-tr.semaphore
	}
}

// NewTasker returns a new Tasker that will run up to n number of tasks
// simultaneously. If n is -1, there is no such restriction.
//
// Returns an error if n is invalid.
func NewTasker(n int) (*Tasker, error) {
	if n < -1 || n == 0 {
		return nil, fmt.Errorf("n must be positive or -1: %d", n)
	}

	var semaphore chan bool
	if n > 0 {
		semaphore = make(chan bool, n)
	} // else semaphore is nil

	tr := &Tasker{
		make(map[string]*task_info),
		make(map[string][]string),
		semaphore,
		-1,
		new_string_stack(),
		make([][]string, 0),
		false,
	}
	return tr, nil
}

// Add adds a task to call a function. name is the unique name for the task.
// deps is a list of the names of tasks to run before this the one being added.
//
// name may not be the empty string.
//
// Any tasks specified in deps must be added before the Tasker can be run. deps
// may be nil, but task may not.
//
// An error is returned if name is not unique.
func (tr *Tasker) Add(name string, deps []string, task Task) error {
	if name == "" {
		return errors.New("name is empty")
	}
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
	tr.dep_graph[name] = deps
	return nil
}

// find_cycles implements Tarjan's Algorithm to construct a list of strongly
// connected components, or cycles, in the directed graph of tasks and their
// dependencies. It sets tr.cycles to a list of lists of task names. Each task
// name list denotes a strongly connected component of more than one vertex.
//
// It is called with the empty string, but called recursively with a task name.
func (tr *Tasker) find_cycles(v string) {
	if v == "" {
		// Initialize algorithm's state.
		tr.index = 0
		tr.stack = new_string_stack()
		tr.cycles = make([][]string, 0)
		for v := range tr.dep_graph {
			ti := tr.tis[v]
			ti.index = -1
			ti.lowlink = -1
			ti.on_stack = false
		}

		// Find all cycles.
		for v := range tr.dep_graph {
			if tr.tis[v].index == -1 {
				// v has not yet been visited.
				tr.find_cycles(v)
			}
		}
	} else {
		// Visit v: Set its index and lowlink and push it onto the stack.
		v_ti := tr.tis[v]
		v_ti.index = tr.index
		v_ti.lowlink = tr.index
		tr.index++
		tr.stack.push(v)
		v_ti.on_stack = true

		// Recursively consider dependencies of v.
		for _, w := range tr.dep_graph[v] {
			w_ti := tr.tis[w]
			if w_ti.index == -1 {

				// w has not yet been visited.
				tr.find_cycles(w)

				// v's lowlink is the smallest index of any
				// recursive dependency of v. If w's lowlink is
				// smaller than v's, it follows that v's
				// lowlink must be set to w's, since w is a
				// dependency of v.
				if w_ti.lowlink < v_ti.lowlink {
					v_ti.lowlink = w_ti.lowlink
				}
			} else if w_ti.on_stack {
				if w_ti.index != w_ti.lowlink {
					panic("w's index and lowlink differ, how!?")
				}
				// w's presence on the stack means that it is
				// in the current scc. It's index is equal to
				// its lowlink because we are in one of its
				// recursive calls.
				if w_ti.index < v_ti.lowlink {
					v_ti.lowlink = w_ti.index
				}
			}
		}

		if v_ti.lowlink == v_ti.index {
			scc := make([]string, 0)
			for {
				w, err := tr.stack.pop()
				if err != nil {
					panic(err)
				}
				tr.tis[w].on_stack = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}

			// Ignore sccs that only include itself, since
			// technically a root node with no dependencies is an
			// scc, and in the Add function we make sure that a
			// task never depends on itself.
			if len(scc) > 1 {
				tr.cycles = append(tr.cycles, scc)
			}
		}
	}
}

// verify returns an error if any task dependencies haven't been added or any
// cycles exist among the tasks.
func (tr *Tasker) verify() error {
	for name, deps := range tr.dep_graph {
		for _, dep := range deps {
			if _, ok := tr.tis[dep]; !ok {
				return NewDepNotFoundError(name, dep)
			}
		}
	}
	tr.find_cycles("")
	if len(tr.cycles) > 0 {
		return CycleError(tr.cycles)
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
	deps := tr.dep_graph[name]
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

// runTasks runs a list of tasks using runTask and waits for them to finish.
func (tr *Tasker) runTasks(names... string) error {
	err_ch := make(chan error)
	for _, name := range names {
		go tr.runTask(name, err_ch)
	}

	// Wait for all tasks to finish. Return the first error encountered.
	var err error
	for _ = range names {
		e := <-err_ch
		if err == nil {
			err = e
		}
	}
	return err
}

// Run runs a list of tasks registered through Add in parallel. If not tasks
// are provided, then all tasks are run.
//
// All tasks are only run once, even if two or more other tasks depend on it.
// A task will not run if any dependency fails.
//
// The last error from a task is returned. Otherwise, Run returns
// nil.
func (tr *Tasker) Run(names... string) error {
	if tr.was_run {
		return errors.New("tasker: already run")
	}

	if err := tr.verify(); err != nil {
		return err
	}

	if len(names) == 0 {
		names = make([]string, 0)
		for name, _ := range tr.tis {
			names = append(names, name)
		}
	} else {
		// Validate the provided tasks.
		for _, name := range names {
			if _, ok := tr.tis[name]; !ok {
				return fmt.Errorf("tasker: task not found: %s", name)
			}
		}
	}

	// This function must not be called again at this point.
	tr.was_run = true

	return tr.runTasks(names...)
}
