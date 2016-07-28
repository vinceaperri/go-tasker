package tasker

import (
	"testing"
	"errors"
)

func test_pop(t *testing.T, s *string_stack, right_e string, right_err error) {
	e, err := s.pop()

	// Wrong element returned.
	if e != right_e {
		t.Errorf("%#v != %#v", e, right_e)
	}

	// Error returned when it shouldn't.
	if err == nil && right_err != nil {
		t.Errorf("%#v != %#v", err, right_err)
	}

	// Error not returned when it should or wrong error returned.
	if err != nil && (right_err == nil || err.Error() != right_err.Error()) {
		t.Errorf("%#v != %#v", err, right_err)
	}
}

func TestSinglePopError(t *testing.T) {
	s := new_string_stack()
	test_pop(t, s, "", errors.New("Stack is empty"))
}

func TestErrorSinglePopPushPopToError(t *testing.T) {
	s := new_string_stack()
	test_pop(t, s, "", errors.New("Stack is empty"))
	s.push("foo")
	test_pop(t, s, "foo", nil)
	test_pop(t, s, "", errors.New("Stack is empty"))
	test_pop(t, s, "", errors.New("Stack is empty"))
}

func TestSinglePushPopToError(t *testing.T) {
	s := new_string_stack()
	s.push("foo")
	test_pop(t, s, "foo", nil)
	test_pop(t, s, "", errors.New("Stack is empty"))
	test_pop(t, s, "", errors.New("Stack is empty"))
}

func TestMultiPushPopToError(t *testing.T) {
	s := new_string_stack()
	s.push("foo")
	s.push("bar")
	s.push("baz")
	test_pop(t, s, "baz", nil)
	test_pop(t, s, "bar", nil)
	test_pop(t, s, "foo", nil)
	test_pop(t, s, "", errors.New("Stack is empty"))
	test_pop(t, s, "", errors.New("Stack is empty"))
}

func TestMultiPushPopToErrorSequence(t *testing.T) {
	s := new_string_stack()
	s.push("foo")
	s.push("bar")
	s.push("baz")
	test_pop(t, s, "baz", nil)
	test_pop(t, s, "bar", nil)
	test_pop(t, s, "foo", nil)
	test_pop(t, s, "", errors.New("Stack is empty"))
	test_pop(t, s, "", errors.New("Stack is empty"))
	s.push("quid")
	s.push("pro")
	s.push("quo")
	test_pop(t, s, "quo", nil)
	test_pop(t, s, "pro", nil)
	test_pop(t, s, "quid", nil)
	test_pop(t, s, "", errors.New("Stack is empty"))
	test_pop(t, s, "", errors.New("Stack is empty"))
	s.push("kn")
	s.push("o")
	s.push("ck")
	test_pop(t, s, "ck", nil)
	test_pop(t, s, "o", nil)
	test_pop(t, s, "kn", nil)
	test_pop(t, s, "", errors.New("Stack is empty"))
	test_pop(t, s, "", errors.New("Stack is empty"))
}
