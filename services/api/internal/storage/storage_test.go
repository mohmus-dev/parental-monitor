package storage

import "testing"

func TestCreateParentAssignsUniqueIDAndRejectsDuplicateEmail(t *testing.T) {
	store := NewMemoryStore()

	p1, err := store.CreateParentWithPassword("John", "john@example.com", "Secret123!")
	if err != nil {
		t.Fatalf("expected first parent registration to succeed: %v", err)
	}
	if p1.ID == "" {
		t.Fatalf("expected generated parent id to be non-empty")
	}
	if p1.PasswordHash == "" {
		t.Fatalf("expected password hash to be generated")
	}

	_, err = store.CreateParentWithPassword("John", "john@example.com", "Secret456!")
	if err == nil {
		t.Fatalf("expected duplicate email to be rejected")
	}

	p2, err := store.CreateParentWithPassword("Jane", "jane@example.com", "Secret456!")
	if err != nil {
		t.Fatalf("expected second parent registration to succeed: %v", err)
	}
	if p1.ID == p2.ID {
		t.Fatalf("expected unique ids for different parents")
	}
}

func TestCreateParentRequiresPassword(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.CreateParentWithPassword("John", "john@example.com", "")
	if err == nil {
		t.Fatalf("expected empty password to be rejected")
	}
}

func TestCreateParentRejectsWeakPassword(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.CreateParentWithPassword("John", "john@example.com", "weak")
	if err == nil {
		t.Fatalf("expected weak password to be rejected")
	}
}

func TestCreateChildLinksParentAndRejectsDuplicateDeviceID(t *testing.T) {
	store := NewMemoryStore()
	parent, err := store.CreateParentWithPassword("John", "john@example.com", "Secret123!", "Parent App", "windows", "parent-device-001")
	if err != nil {
		t.Fatalf("expected parent registration to succeed: %v", err)
	}

	child, err := store.CreateChild(parent.ID, "Emma", 12, "device-001", "Child App", "android")
	if err != nil {
		t.Fatalf("expected child creation to succeed: %v", err)
	}
	if child.ParentID != parent.ID {
		t.Fatalf("expected child to be linked to parent")
	}
	if child.DeviceID != "device-001" {
		t.Fatalf("expected child device_id to match")
	}
	if child.Platform != "android" {
		t.Fatalf("expected child platform to be stored")
	}
	if parent.Platform != "windows" || parent.AppName != "Parent App" || parent.DeviceID != "parent-device-001" {
		t.Fatalf("expected parent install metadata to be stored")
	}

	_, err = store.CreateChild(parent.ID, "Liam", 9, "device-001", "Child App", "ios")
	if err == nil {
		t.Fatalf("expected duplicate device id to be rejected")
	}
}

func TestDeleteParentDeletesAssociatedChildren(t *testing.T) {
	store := NewMemoryStore()
	parent, err := store.CreateParentWithPassword("John", "john@example.com", "Secret123!", "Parent App", "windows", "parent-device-001")
	if err != nil {
		t.Fatalf("expected parent registration to succeed: %v", err)
	}

	_, err = store.CreateChild(parent.ID, "Emma", 12, "device-001", "Child App", "android")
	if err != nil {
		t.Fatalf("expected child creation to succeed: %v", err)
	}

	if err := store.DeleteParent(parent.ID); err != nil {
		t.Fatalf("expected parent deletion to succeed: %v", err)
	}

	if _, ok := store.GetParentByID(parent.ID); ok {
		t.Fatalf("expected parent to be removed")
	}
	if children := store.ListChildrenByParent(parent.ID); len(children) != 0 {
		t.Fatalf("expected all children for the deleted parent to be removed, got %d", len(children))
	}
}

func TestDeleteChildrenByIDRemovesOnlySelectedChildren(t *testing.T) {
	store := NewMemoryStore()
	parent, err := store.CreateParentWithPassword("John", "john@example.com", "Secret123!", "Parent App", "windows", "parent-device-001")
	if err != nil {
		t.Fatalf("expected parent registration to succeed: %v", err)
	}

	c1, err := store.CreateChild(parent.ID, "Emma", 12, "device-001", "Child App", "android")
	if err != nil {
		t.Fatalf("expected first child creation to succeed: %v", err)
	}
	c2, err := store.CreateChild(parent.ID, "Liam", 9, "device-002", "Child App", "ios")
	if err != nil {
		t.Fatalf("expected second child creation to succeed: %v", err)
	}

	if err := store.DeleteChildren(parent.ID, []string{c1.ID}); err != nil {
		t.Fatalf("expected selected child deletion to succeed: %v", err)
	}

	children := store.ListChildrenByParent(parent.ID)
	if len(children) != 1 || children[0].ID != c2.ID {
		t.Fatalf("expected only the selected child to be removed")
	}
}
