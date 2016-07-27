package tasker

import (
	"errors"
	"testing"
)

type test_task struct {
	name   string
	deps   []string
	task   Task
	called bool
}

func new_test_task(name string, deps []string, task Task) *test_task {
	return &test_task{name, deps, task, false}
}

func (tt *test_task) get_task() Task {
	return func() error {
		tt.called = true
		return tt.task()
	}
}

func good_task() error {
	return nil
}

func bad_task() error {
	return errors.New("")
}

func run(t *testing.T, n int, tts []*test_task) error {
	tr, err := NewTasker(n)
	if err != nil {
		t.Error(err)
	}
	for _, tt := range tts {
		err = tr.Add(tt.name, tt.deps, tt.get_task())
		if err != nil {
			t.Error(err)
		}
	}
	return tr.Run()
}

func test_run_ok(t *testing.T, n int, tts []*test_task) {
	err := run(t, n, tts)
	if err != nil {
		t.Error(err)
	}
	for _, tt := range tts {
		if !tt.called {
			t.Errorf("%s not called", tt.name)
		}
	}
}

func test_run_cycle_error(t *testing.T, n int, tts []*test_task) {
	err := run(t, n, tts)
	if _, ok := err.(CycleError); !ok {
		t.Error(err)
	}
}

func test_run_dep_not_found_error(t *testing.T, n int, tts []*test_task) {
	err := run(t, n, tts)
	if _, ok := err.(*DepNotFoundError); !ok {
		t.Error(err)
	}
}

func TestRunOne(t *testing.T) {
	test_run_ok(t, 1, []*test_task{
		new_test_task("foo", nil, good_task),
	})
}

func TestRunThreeIndependent(t *testing.T) {
	test_run_ok(t, 1, []*test_task{
		new_test_task("foo", nil, good_task),
		new_test_task("bar", nil, good_task),
		new_test_task("baz", nil, good_task),
	})
}

func TestErrorCycleDetectionTwo(t *testing.T) {
	test_run_cycle_error(t, 1, []*test_task{
		new_test_task("foo", []string{"bar"}, good_task),
		new_test_task("bar", []string{"foo"}, good_task),
	})
}

func TestErrorCycleDetectionThree(t *testing.T) {
	test_run_cycle_error(t, 1, []*test_task{
		new_test_task("quid", []string{"pro"}, good_task),
		new_test_task("pro", []string{"quo"}, good_task),
		new_test_task("quo", []string{"quid"}, good_task),
	})
}

func TestErrorDepNotFoundOne(t *testing.T) {
	test_run_dep_not_found_error(t, 1, []*test_task{
		new_test_task("foo", []string{"bar"}, good_task),
	})
}

func TestErrorDepNotFoundThree(t *testing.T) {
	test_run_dep_not_found_error(t, 1, []*test_task{
		new_test_task("foo", []string{"bar", "baz", "boo"}, good_task),
	})
}

func TestErrorDepNotFoundSome(t *testing.T) {
	test_run_dep_not_found_error(t, 1, []*test_task{
		new_test_task("foo", []string{"bar", "baz", "boo"}, good_task),
		new_test_task("bar", nil, good_task),
	})
}
