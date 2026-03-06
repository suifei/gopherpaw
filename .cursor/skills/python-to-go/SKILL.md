---
name: python-to-go
description: Translate CoPaw Python modules to idiomatic Go implementations. Use when the user asks to convert, translate, rewrite, or port a Python module from copaw-source/ to Go, or when implementing a GopherPaw module based on CoPaw source.
---

# Python to Go Translation

## When to Use

- User asks to translate/convert/port a CoPaw Python module
- User asks to implement a GopherPaw module "based on" or "like" a CoPaw module
- User references a Python file in `copaw-source/` and wants Go equivalent

## Translation Workflow

Copy this checklist and track progress:

```
Translation Progress:
- [ ] Step 1: Read and understand the Python source
- [ ] Step 2: Extract the interface contract
- [ ] Step 3: Design Go interfaces
- [ ] Step 4: Implement in Go
- [ ] Step 5: Write tests (TDD - edge cases first)
- [ ] Step 6: Update contract docs
```

### Step 1: Read and Understand

Read the target Python file(s) in `copaw-source/src/copaw/`. Focus on:

- What problem does this module solve?
- What are the public methods (the contract)?
- What are the dependencies (imports from other CoPaw modules)?
- What are the side effects (I/O, network, state mutation)?

### Step 2: Extract Interface Contract

List all public methods with their signatures:

```
Python class: ClassName
  - method_name(arg1: type, arg2: type) -> return_type
  - another_method(arg: type) -> return_type
```

Identify which methods are essential vs. framework-specific (AgentScope).

### Step 3: Design Go Interfaces

Translate the extracted contract to Go interfaces. Apply these rules:

- Python class -> Go interface (at consumer side) + struct (at provider side)
- Keep interfaces small (1-3 methods)
- Use `context.Context` as first parameter for anything that does I/O
- Return `error` as last return value

### Step 4: Implement

- Place code in the correct `internal/` package per architecture spec
- Use composition (embedding) instead of inheritance
- Replace Python async with goroutines + context
- Replace Python exceptions with error returns

### Step 5: Write Tests

**Before or alongside implementation**, write tests covering:

- Normal path
- Empty/nil inputs
- Error conditions
- Concurrent access (if stateful)
- Timeout behavior

### Step 6: Update Docs

Update `docs/api_spec.md` with new interface definitions.

## Pattern Reference

For detailed Python-to-Go pattern mappings, see [reference.md](reference.md).
