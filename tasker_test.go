package tasker

import (
	"errors"
	"testing"
)

func good_task() error {
	return nil
}

func bad_task() error {
	return errors.New("")
}

type test_task struct {
	name   string
	deps   []string
	task   Task
	called bool
}

func new_good_test_task(name string, deps []string) *test_task {
	return &test_task{name, deps, good_task, false}
}

func new_bad_test_task(name string, deps []string) *test_task {
	return &test_task{name, deps, bad_task, false}
}

// add adds a task to a Tasker with a Task that wraps around the actual task in
// order to set called when the Task gets called.
func (tt *test_task) add(tr *Tasker) error {
	return tr.Add(tt.name, tt.deps, func() error {
		tt.called = true
		return tt.task()
	})
}

func test_run(t *testing.T, n int, tts []*test_task) error {
	tr, err := NewTasker(n)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range tts {
		err = tt.add(tr)
		if err != nil {
			t.Fatal(err)
		}
	}
	return tr.Run()
}

func test_run_ok(t *testing.T, n int, tts []*test_task) {
	err := test_run(t, n, tts)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range tts {
		if !tt.called {
			// Print out all of these errors.
			t.Errorf("%s not called", tt.name)
		}
	}
}

func test_run_task_error(t *testing.T, n int, tts[]*test_task) {
	err := test_run(t, n, tts)
	if err == nil {
		t.Fatalf("No error occurred")
	}
}

func test_run_cycle_error(t *testing.T, n int, tts []*test_task) {
	err := test_run(t, n, tts)
	if _, ok := err.(CycleError); !ok {
		t.Fatal(err)
	}
}

func test_run_dep_not_found_error(t *testing.T, n int, tts []*test_task) {
	err := test_run(t, n, tts)
	if _, ok := err.(*DepNotFoundError); !ok {
		t.Fatal(err)
	}
}

func TestRunOne(t *testing.T) {
	test_run_ok(t, 1, []*test_task{
		new_good_test_task("foo", nil),
	})
}

func TestRunThreeIndependent(t *testing.T) {
	test_run_ok(t, 1, []*test_task{
		new_good_test_task("foo", nil),
		new_good_test_task("bar", nil),
		new_good_test_task("baz", nil),
	})
}

func TestRunTwoIndependentGraphs(t *testing.T) {
	test_run_ok(t, 1, []*test_task{
		new_good_test_task("1", []string{"11", "12"}),

		new_good_test_task("11", []string{"111", "112"}),
		new_good_test_task("111", nil),
		new_good_test_task("112", nil),

		new_good_test_task("12", []string{"121", "122"}),
		new_good_test_task("121", nil),
		new_good_test_task("122", nil),

		new_good_test_task("2", []string{"21", "22"}),

		new_good_test_task("21", []string{"211", "212"}),
		new_good_test_task("211", nil),
		new_good_test_task("212", nil),

		new_good_test_task("22", []string{"221", "222"}),
		new_good_test_task("221", nil),
		new_good_test_task("222", nil),
	})
}

func TestRunFailEarly(t *testing.T) {
	called_tts := []*test_task {
		new_good_test_task("11", []string{"111", "112"}),
		new_good_test_task("111", nil),
		new_good_test_task("112", nil),
		new_good_test_task("12", []string{"121", "122"}),
		new_good_test_task("121", nil),
		new_good_test_task("122", nil),
	}
	all_tts := []*test_task{
		new_bad_test_task("1", []string{"11", "12"}),
		// 11 and 12 are in must_run
		new_good_test_task("21", []string{"211", "212"}),
		new_good_test_task("211", nil),
		new_good_test_task("212", nil),
		new_good_test_task("22", []string{"221", "222"}),
		new_good_test_task("221", nil),
		new_good_test_task("222", nil),
	}
	all_tts = append(all_tts, called_tts...)

	test_run_task_error(t, 1, all_tts)

	for _, tt := range called_tts {
		if !tt.called {
			// Print out all of these errors.
			t.Errorf("%s not called", tt.name)
		}
	}
}


func TestErrorCycleDetectionTwo(t *testing.T) {
	test_run_cycle_error(t, 1, []*test_task{
		new_good_test_task("foo", []string{"bar"}),
		new_good_test_task("bar", []string{"foo"}),
	})
}

func TestErrorCycleDetectionThree(t *testing.T) {
	test_run_cycle_error(t, 1, []*test_task{
		new_good_test_task("quid", []string{"pro"}),
		new_good_test_task("pro", []string{"quo"}),
		new_good_test_task("quo", []string{"quid"}),
	})
}

func TestErrorDepNotFoundOne(t *testing.T) {
	test_run_dep_not_found_error(t, 1, []*test_task{
		new_good_test_task("foo", []string{"bar"}),
	})
}

func TestErrorDepNotFoundThree(t *testing.T) {
	test_run_dep_not_found_error(t, 1, []*test_task{
		new_good_test_task("foo", []string{"bar", "baz", "boo"}),
	})
}

func TestErrorDepNotFoundSome(t *testing.T) {
	test_run_dep_not_found_error(t, 1, []*test_task{
		new_good_test_task("foo", []string{"bar", "baz", "boo"}),
		new_good_test_task("bar", nil),
	})
}

func TestErrorRunTwice(t *testing.T) {
	tr, err := NewTasker(1)
	if err != nil {
		t.Fatal(err)
	}
	tt := new_good_test_task("foo", nil)
	err = tt.add(tr)
	if err != nil {
		t.Fatal(err)
	}
	err = tr.Run()
	if err != nil {
		t.Fatal(err)
	}
	err = tr.Run()
	if err == nil {
		t.Fatal(err)
	}
	if err.Error() != "tasker: already run" {
		t.Fatal(err)
	}
}
