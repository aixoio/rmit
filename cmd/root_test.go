package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetCurrentBranch(t *testing.T) {
	// Test that function doesn't panic when git is not available
	// This is a basic smoke test since we can't easily mock git in unit tests
	branch, err := getCurrentBranch()
	if err != nil {
		// If git is not available, we expect an error
		if !strings.Contains(err.Error(), "git is not installed") {
			t.Errorf("Unexpected error: %v", err)
		}
	} else {
		// If git is available, branch should not be empty
		if branch == "" {
			t.Error("Expected non-empty branch name")
		}
	}
}

func TestTrackCodeChanges(t *testing.T) {
	// Test with a sample git diff
	diff := `diff --git a/example.txt b/example.txt
index 1234567..abcdef0 100644
--- a/example.txt
+++ b/example.txt
@@ -1,3 +1,4 @@
 line 1
-line 2
+modified line 2
+new line
 line 3`

	changes, err := trackCodeChanges(diff)
	if err != nil {
		t.Fatalf("trackCodeChanges failed: %v", err)
	}

	if len(changes) == 0 {
		t.Error("Expected changes to be detected")
	}

	// Check that changes contain additions and deletions
	foundAddition := false
	foundDeletion := false
	for _, change := range changes {
		if strings.Contains(change, "+modified line 2") {
			foundAddition = true
		}
		if strings.Contains(change, "-line 2") {
			foundDeletion = true
		}
	}

	if !foundAddition {
		t.Error("Expected to find addition in changes")
	}
	if !foundDeletion {
		t.Error("Expected to find deletion in changes")
	}
}

func TestGetProjectInfo(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "rmit_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Test with no project files
	info, err := getProjectInfo()
	if err != nil {
		t.Fatalf("getProjectInfo failed: %v", err)
	}
	if !strings.Contains(info, "Unknown project type") {
		t.Errorf("Expected 'Unknown project type' in info, got: %s", info)
	}

	// Test with Go project
	err = os.WriteFile("go.mod", []byte("module test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	info, err = getProjectInfo()
	if err != nil {
		t.Fatalf("getProjectInfo failed: %v", err)
	}
	if !strings.Contains(info, "Go") {
		t.Errorf("Expected 'Go' in project info, got: %s", info)
	}

	// Test with Node.js project
	os.Remove("go.mod")
	err = os.WriteFile("package.json", []byte(`{"name": "test", "dependencies": {"react": "^18.0.0"}}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	info, err = getProjectInfo()
	if err != nil {
		t.Fatalf("getProjectInfo failed: %v", err)
	}
	if !strings.Contains(info, "React") {
		t.Errorf("Expected 'React' in project info, got: %s", info)
	}

	// Test with Python project
	os.Remove("package.json")
	err = os.WriteFile("pyproject.toml", []byte("[tool.poetry]\nname = \"test\""), 0644)
	if err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	info, err = getProjectInfo()
	if err != nil {
		t.Fatalf("getProjectInfo failed: %v", err)
	}
	if !strings.Contains(info, "Python") {
		t.Errorf("Expected 'Python' in project info, got: %s", info)
	}

	// Test with Rust project
	os.Remove("pyproject.toml")
	err = os.WriteFile("Cargo.toml", []byte("[package]\nname = \"test\""), 0644)
	if err != nil {
		t.Fatalf("Failed to create Cargo.toml: %v", err)
	}

	info, err = getProjectInfo()
	if err != nil {
		t.Fatalf("getProjectInfo failed: %v", err)
	}
	if !strings.Contains(info, "Rust") {
		t.Errorf("Expected 'Rust' in project info, got: %s", info)
	}
}

func TestGetProjectInfoWithSubdirs(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "rmit_test_subdir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Create frontend subdirectory with Next.js config
	err = os.Mkdir("frontend", 0755)
	if err != nil {
		t.Fatalf("Failed to create frontend dir: %v", err)
	}

	err = os.WriteFile(filepath.Join("frontend", "next.config.js"), []byte("module.exports = {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create next.config.js: %v", err)
	}

	info, err := getProjectInfo()
	if err != nil {
		t.Fatalf("getProjectInfo failed: %v", err)
	}
	if !strings.Contains(info, "Next.js") {
		t.Errorf("Expected 'Next.js' in project info, got: %s", info)
	}
}

func TestGetProjectInfoWithDocker(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "rmit_test_docker")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Create Dockerfile
	err = os.WriteFile("Dockerfile", []byte("FROM alpine"), 0644)
	if err != nil {
		t.Fatalf("Failed to create Dockerfile: %v", err)
	}

	info, err := getProjectInfo()
	if err != nil {
		t.Fatalf("getProjectInfo failed: %v", err)
	}
	if !strings.Contains(info, "Dockerized") {
		t.Errorf("Expected 'Dockerized' in project info, got: %s", info)
	}
}
