package fs 

import (
	"os" 
	"testing"
)

func TestExpandPath(t *testing.T){
	p := "path:to:something" 
	expected := p 
	result := ExpandPath(p) 
	if result != expected{
		t.Errorf("expected %s, but got %s", expected, result)
	}

	// Test case 2: p contains an @ symbol 
	p = "user@example.com" 
	expected = p 
	result = ExpandPath(p) 
	if result != expected{
		t.Errorf("expected %s, but got %s", expected, result) 
	}

	// Test case 3: p starts with ~/ 
	p = "~/path/to/something" 
	expected = os.Getenv("HOME") + "/path/to/something" 
	result = ExpandPath(p) 
	if result != expected{
		t.Errorf("expected %s, but got %s", expected, result) 
	}

	// Test case 4: p starts with ~\ 
	p = "~\\path\\to\\something" 
	expected = os.Getenv("HOME") + "\\path\\to\\something" 
	result = ExpandPath(p) 
	if result != expected{
		t.Errorf("expected %s, but got %s", expected, result) 
	}

	// Test case 5: p does not match any conditions 
	p = "path/to/something" 
	expected = p 
	result = ExpandPath(p) 
	if result != expected{
		t.Errorf("expected %s, but got %s", expected, result)
	}
}

func TestRemoveDir(t *testing.T){
	// Test case 1: path exists 
	path := "testdir" 
	err := os.Mkdir(path, 0755) 
	if err != nil{
		t.Errorf("failed to create test directory: %v", err) 
	}
	err = RemoveDir(path) 
	if err != nil{
		t.Errorf("failed to remove directory: %v", err) 
	}

	// Test case 2: path does not exist 
	path = "nonexistantDir" 
	err = RemoveDir(path) 
	if err == nil{
		t.Errorf("expected an error, but got nil") 
	}
}

func TestHomeDir(t *testing.T){
	// test case 1: HOME environment variable is set 
	expected := os.Getenv("HOME") 
	result := homeDir() 
	if result != expected{
		t.Errorf("expected %s, but got %s", expected, result) 
	}

	// Test case 2: Home environment variable is not set 
	os.Unsetenv("HOME") 
	expected = "" 
	result = homeDir() 
	if result != expected{
		t.Errorf("expected %s, but got %s", expected, result )
	}
}