---
name: add-go-module
description: Standardized workflow for adding a new Go module to GopherPaw. Use when the user asks to add a new feature, create a new package, implement a new component, or build a new module in the internal/ directory.
---

# Add Go Module Workflow

## When to Use

- Adding a new package under `internal/`
- Implementing a new feature module (e.g. new channel, new tool, new LLM provider)
- Building a new subsystem from CoPaw reference

## 5-Step Standard Workflow

Copy this checklist and track progress:

```
Add Module Progress:
- [ ] Step 1: Analyze CoPaw reference
- [ ] Step 2: Update contract docs (Contract First!)
- [ ] Step 3: Define Go interfaces
- [ ] Step 4: Implement + write edge-case tests
- [ ] Step 5: Wire up in cmd/ or parent module
```

### Step 1: Analyze CoPaw Reference

Read the corresponding Python module in `copaw-source/src/copaw/`:

1. Identify the module's responsibility and boundaries
2. List public APIs (methods that other modules call)
3. List dependencies (what other modules does it import?)
4. Note any patterns worth preserving or improving

Output: brief summary of what this module does and key design decisions.

### Step 2: Update Contract Docs

**Before writing any code**, update the contract documents:

1. **`docs/architecture_spec.md`**: Add the new module to the architecture diagram, define its layer placement and dependency relationships
2. **`docs/api_spec.md`**: Add the Go interface definitions for this module

Ask the user to review and approve the spec changes before proceeding.

### Step 3: Define Go Interfaces

Create the interface file in the correct package:

```go
// internal/newmodule/newmodule.go

// Package newmodule provides [one-line description].
package newmodule

import "context"

// SomeInterface defines the contract for [purpose].
type SomeInterface interface {
    Method(ctx context.Context, arg Type) (Result, error)
}
```

Rules:
- Interface defined at the **consumer** side
- Implementation at the **provider** side
- Keep interfaces small (1-3 methods)
- Use `context.Context` for I/O operations

### Step 4: Implement + Test

Implement the interface with a concrete struct. Write tests simultaneously.

Test requirements (per testing-tdd rule):
- Table-driven tests
- Edge cases: nil, empty, oversized, concurrent, timeout
- Mock all external dependencies via interfaces
- Target >= 80% coverage for core logic

File layout:
```
internal/newmodule/
├── newmodule.go       # Interface + types
├── impl.go            # Implementation
├── impl_test.go       # Tests
└── options.go         # Functional options (if needed)
```

### Step 5: Wire Up

Connect the new module to the application:

1. Add configuration fields in `internal/config/` if needed
2. Instantiate in `cmd/gopherpaw/main.go` with dependency injection
3. Pass to consumers via constructor parameters
4. Verify the full chain works with an integration-level test

## Post-Completion

After all steps are done:
1. Run `go vet ./...` and `go test ./...`
2. Verify contract docs match the implementation
3. Update `CONTEXT.md` if the module changes project status/progress
