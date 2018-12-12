package flickraccess

import "testing"

func TestSimpleJsonMerge(t *testing.T) {

	s1 := `{"a":"1"}`
	s2 := `{"b":"2"}`

	s3, _ := mergeJson(s1, s2)

	assertEquals(`{"a":"1","b":"2"}`, s3, t)
}

func assertEquals(expected string, actual string, t *testing.T) {
	if actual != expected {
		t.Errorf("Test failed, expected: '%s', got:  '%s'", expected, actual)
	}
}