package authz

import "testing"

func TestAuthorOnly(t *testing.T) {
	if !AuthorOnly("a", "a") {
		t.Fatal("author should be allowed")
	}
	if AuthorOnly("b", "a") || AuthorOnly("", "a") || AuthorOnly("a", "") {
		t.Fatal("non-author should be rejected")
	}
}
