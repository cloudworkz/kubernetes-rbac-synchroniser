package main

import (
	"testing"

	"google.golang.org/api/admin/directory/v1"
)

func TestUniq(t *testing.T) {
	uniqUserList1 := uniq(getFakeMembers())
	list1Length := len(uniqUserList1)
	if list1Length != 1 {
		t.Errorf("Uniq was incorrect, got: %d, want: %d.", list1Length, 1)
	}

	var uniqUserList2 []*admin.Member
	var member1 = new(admin.Member)
	member1.Email = "member1@example.com"
	var member2 = new(admin.Member)
	member2.Email = "member2@example.com"
	var member3 = new(admin.Member)
	member3.Email = "member3@example.com"
	uniqUserList2 = append(uniqUserList2, member1)
	uniqUserList2 = append(uniqUserList2, member2)
	uniqUserList2 = append(uniqUserList2, member1)
	uniqUserList2 = append(uniqUserList2, member3)
	uniqUserList2 = append(uniqUserList2, member2)
	uniqUserList2 = uniq(uniqUserList2)
	list2Length := len(uniqUserList2)
	if list2Length != 3 {
		t.Errorf("Uniq was incorrect, got: %d, want: %d.", list2Length, 3)
	}
	if uniqUserList2[0].Email != member1.Email {
		t.Errorf("Uniq sort was incorrect, got: %q, want: %q.", uniqUserList2[0].Email, member1.Email)
	}
	if uniqUserList2[1].Email != member2.Email {
		t.Errorf("Uniq sort was incorrect, got: %q, want: %q.", uniqUserList2[1].Email, member2.Email)
	}
	if uniqUserList2[2].Email != member3.Email {
		t.Errorf("Uniq sort was incorrect, got: %q, want: %q.", uniqUserList2[2].Email, member3.Email)
	}
}
