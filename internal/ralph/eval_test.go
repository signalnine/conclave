package ralph

import (
	"testing"
)

func TestExtractFilePaths_NodeJS(t *testing.T) {
	output := `FAIL src/routes/users.test.js
  ● POST /users › returns 201

    TypeError: Cannot read properties of undefined
      at Object.<anonymous> (src/routes/users.js:45:12)
      at processTicksAndRejections (node:internal/process/task_queues:95:5)

  ● GET /users › returns list

    Error: expected 200 got 404
      at Object.<anonymous> (src/routes/users.js:12:5)
      at Object.<anonymous> (src/middleware/auth.js:8:3)`

	paths := ExtractFilePathsFromOutput(output)
	expected := map[string]bool{
		"src/routes/users.js":      true,
		"src/middleware/auth.js":    true,
		"src/routes/users.test.js": true,
	}
	for _, p := range paths {
		if !expected[p] {
			t.Errorf("unexpected path: %s", p)
		}
		delete(expected, p)
	}
	for p := range expected {
		t.Errorf("missing path: %s", p)
	}
}

func TestExtractFilePaths_Python(t *testing.T) {
	output := `FAILED tests/test_api.py::test_create_user
  File "src/api/users.py", line 23, in create_user
    raise ValueError("invalid email")
  File "src/api/validators.py", line 8, in validate_email
    return re.match(pattern, email)`

	paths := ExtractFilePathsFromOutput(output)
	expected := map[string]bool{
		"src/api/users.py":      true,
		"src/api/validators.py": true,
		"tests/test_api.py":     true,
	}
	for _, p := range paths {
		if !expected[p] {
			t.Errorf("unexpected path: %s", p)
		}
		delete(expected, p)
	}
	for p := range expected {
		t.Errorf("missing path: %s", p)
	}
}

func TestExtractFilePaths_Go(t *testing.T) {
	output := "--- FAIL: TestCreateUser (0.01s)\n    users_test.go:34: expected 201 got 500\ngoroutine 1 [running]:\nmain.createUser(...)\n\tcmd/server/handlers.go:45\nmain.validateInput(...)\n\tinternal/validation/input.go:12"

	paths := ExtractFilePathsFromOutput(output)
	expected := map[string]bool{
		"cmd/server/handlers.go":       true,
		"internal/validation/input.go": true,
	}
	for _, p := range paths {
		if !expected[p] {
			t.Errorf("unexpected path: %s", p)
		}
		delete(expected, p)
	}
	for p := range expected {
		t.Errorf("missing path: %s", p)
	}
}

func TestExtractFilePaths_FiltersNodeModules(t *testing.T) {
	output := `Error: foo
      at Object.<anonymous> (node_modules/express/lib/router.js:12:5)
      at Object.<anonymous> (src/app.js:3:10)`

	paths := ExtractFilePathsFromOutput(output)
	for _, p := range paths {
		if p == "node_modules/express/lib/router.js" {
			t.Error("should filter node_modules")
		}
	}
	found := false
	for _, p := range paths {
		if p == "src/app.js" {
			found = true
		}
	}
	if !found {
		t.Error("should include src/app.js")
	}
}

func TestExtractFilePaths_Empty(t *testing.T) {
	paths := ExtractFilePathsFromOutput("")
	if len(paths) != 0 {
		t.Errorf("expected empty, got %v", paths)
	}
}
