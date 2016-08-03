# tasker

tasker is a library for running tasks in parallel with an arbitrary dependency
specification.

## Usage

Setup a tasker to run with only up to 5 tasks at a time:

```go
tr := tasker.NewTasker(5)
```

or, if you're feeling ambitious, configure it with no limitation on the number
of tasks it can run at a time:

```go
tr := tasker.NewTasker(-1)
```

Add a few tasks, making sure that all dependencies are also added. The ordering
of these calls doesn't matter.

```go
tr.Add("d", nil, func() error { fmt.Println("d"); return nil })
tr.Add("a", []string{"b", "c"}, func() error { fmt.Println("a"); return nil })
tr.Add("b", nil, func() error { fmt.Println("b"); return nil })
tr.Add("c", []string{"d"}, func() error { fmt.Println("c"); return nil })
```

Putting it all together:

```go
package main

import (
	"log"
	"github.com/perriv/go-tasker"
)

func print_task(msg string) tasker.Task {
	return func() error {
		log.Println(msg)
		return nil
	}
}

func main() {
	tr, err := tasker.NewTasker(-1)
	if err != nil {
		log.Fatal(err)
	}

	err = tr.Add("d", nil, print_task("d"))
	if err != nil {
		log.Fatal(err)
	}

	err = tr.Add("a", []string{"b", "c"}, print_task("a"))
	if err != nil {
		log.Fatal(err)
	}

	err = tr.Add("b", nil, print_task("b"))
	if err != nil {
		log.Fatal(err)
	}

	err = tr.Add("c", []string{"d"}, print_task("c"))
	if err != nil {
		log.Fatal(err)
	}

	if err = tr.Run(); err != nil {
		log.Fatal(err)
	}
}
```

This program's output would be something like:

```
2016/08/03 13:03:08 d
2016/08/03 13:03:08 c
2016/08/03 13:03:08 b
2016/08/03 13:03:08 a
```

Although c and b could have just as easily been switched:

```
2016/08/03 13:03:09 d
2016/08/03 13:03:09 b
2016/08/03 13:03:09 c
2016/08/03 13:03:09 a
```

In case you were wondering, log is used above instead of fmt in the tasks
because it's thread-safe.

## Invalid Dependency Graphs

Since dependencies don't have to be defined in a call to `Add`, two problems
can arise that the user must be mindful of before calling `Run`.

The first problem is that all dependencies might not have been added. This was
mentioned earlier, but it worth repeating here. `Run` checks for this and
doesn't run unless all dependencies are defined.

The second problem is that there might be a dependency cycle among the tasks.
`Add` takes care of the obvious self-dependent cycle of "a" depending on "a",
but cycles involving more than one task can't be detected as easily. Instead,
`Run` checks for these multi-task cycles using
[Tarjan's Algorithm](https://en.wikipedia.org/wiki/Tarjan%27s_strongly_connected_components_algorithm)
for linear performance. It's been modified slightly to return multi-task cycles
only, since tasks with no dependencies are valid and tasks that depend on
themselves are taken care of by `Add`.

## Documentation

Source code documentation can be found on
[godoc.org](http://godoc.org/github.com/perriv/go-tasker).
