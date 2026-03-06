---
name: architecture-review
description: Review code for architecture compliance against GopherPaw's layered architecture rules. Use when the user asks to review, audit, or check code for architecture violations, dependency issues, or design pattern compliance.
---

# Architecture Compliance Review

## When to Use

- After implementing a new module (post-implementation check)
- Before merging significant changes
- When user asks "does this code follow our architecture?"
- Periodic health check of the codebase

## Review Checklist

```
Review Progress:
- [ ] Check 1: Layer dependency compliance
- [ ] Check 2: Interface-first design
- [ ] Check 3: Error handling completeness
- [ ] Check 4: Test coverage and edge cases
- [ ] Check 5: Global state and concurrency safety
- [ ] Check 6: Contract document alignment
```

### Check 1: Layer Dependency Compliance

Verify that imports follow the dependency rules:

```
Allowed:
  cmd/ --> internal/* (any)
  channels/ --> agent/ (interface only)
  agent/ --> llm/, memory/, tools/ (interface only), config/
  scheduler/ --> agent/ (interface only), config/
  llm/, memory/, tools/ --> config/ (only)

Forbidden:
  llm/ --> agent/ (infra must not import domain)
  memory/ --> agent/ (infra must not import domain)
  tools/ --> channels/ (tools must not import interface layer)
  Any circular dependencies
```

How to check:
1. Read the import statements of each `.go` file in the target package
2. Verify each import is allowed by the dependency matrix above
3. Flag any violations with the specific file and import path

### Check 2: Interface-First Design

Verify that consumers depend on interfaces, not concrete types:

- [ ] `internal/agent/` defines interfaces for `LLMProvider`, `MemoryStore`, `Tool`
- [ ] `internal/llm/` implements `LLMProvider` without agent/ importing llm/ directly
- [ ] Constructor functions accept interfaces, not concrete structs
- [ ] No type assertions to concrete types outside of factory/wire-up code

### Check 3: Error Handling Completeness

- [ ] No ignored error returns (no `_ = someFunc()` for functions returning errors)
- [ ] Errors wrapped with context: `fmt.Errorf("operation desc: %w", err)`
- [ ] Sentinel errors defined at package level for expected failure modes
- [ ] No `panic()` in library code (only acceptable in `main()` for unrecoverable startup errors)

### Check 4: Test Coverage and Edge Cases

- [ ] Every exported function has at least one test
- [ ] Tests use table-driven pattern
- [ ] Edge cases covered: nil input, empty input, large input, timeout, concurrent access
- [ ] External dependencies mocked via interfaces
- [ ] No test interdependencies (each test is independent)

### Check 5: Global State and Concurrency

- [ ] No package-level mutable variables (except `sync.Once` for initialization)
- [ ] Shared state protected by `sync.Mutex` or `sync.RWMutex`
- [ ] goroutines have clear shutdown mechanism via `context.Context`
- [ ] Channels are properly closed by the sender
- [ ] No goroutine leaks (every goroutine has an exit path)

### Check 6: Contract Document Alignment

- [ ] All interfaces in code exist in `docs/api_spec.md`
- [ ] Architecture diagram in `docs/architecture_spec.md` matches actual package structure
- [ ] No undocumented packages in `internal/`

## Output Format

```markdown
# Architecture Review Report

## Summary
[PASS/FAIL] - [N] violations found

## Violations

### [Severity: Critical/Warning/Info]
- **File**: `internal/xxx/yyy.go`
- **Check**: [which check failed]
- **Issue**: [description]
- **Fix**: [recommended fix]

## Recommendations
- [General improvement suggestions]
```

## Severity Levels

- **Critical**: Must fix before proceeding (dependency violations, missing error handling)
- **Warning**: Should fix soon (missing tests, undocumented interfaces)
- **Info**: Nice to have (naming suggestions, minor style issues)
