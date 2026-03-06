---
name: copaw-analysis
description: Analyze CoPaw Python source modules to understand architecture, data flow, and dependencies for Go reimplementation. Use when the user asks to analyze, study, or understand a CoPaw module before implementing its Go equivalent.
---

# CoPaw Source Analysis

## When to Use

- Before implementing a GopherPaw module, to understand the CoPaw original
- When the user asks "how does CoPaw handle X?"
- When mapping CoPaw functionality to Go architecture

## Analysis Workflow

```
Analysis Progress:
- [ ] Step 1: Identify the target module
- [ ] Step 2: Read source files
- [ ] Step 3: Map architecture
- [ ] Step 4: Generate Go implementation recommendations
```

### Step 1: Identify Target Module

Use the module map in [reference.md](reference.md) to locate the relevant files. CoPaw has 8 major modules:

1. **Agent** - `agents/react_agent.py` (ReAct loop)
2. **Runner** - `app/runner/` (runtime, session, chat management)
3. **Channels** - `app/channels/` (messaging platforms)
4. **Tools** - `agents/tools/` (built-in capabilities)
5. **Memory** - `agents/memory/` (ReMe-based recall)
6. **Providers** - `providers/` (LLM backends)
7. **Config** - `config/` (Pydantic-based configuration)
8. **CLI** - `cli/` (command-line interface)

### Step 2: Read Source Files

For each target module, read:

1. The main implementation file(s)
2. The `__init__.py` for public API surface
3. Related test files in `tests/`
4. Any referenced utility modules

### Step 3: Map Architecture

Produce a structured analysis:

**Module Summary**
```markdown
## [Module Name]

### Purpose
[One paragraph: what problem does this solve?]

### Key Classes/Functions
- `ClassName` - [responsibility]
- `function_name()` - [what it does]

### Data Flow
[Mermaid sequence diagram or flowchart]

### Dependencies
- Internal: [other CoPaw modules used]
- External: [third-party libraries]

### State Management
- [What state is stored, where, and how]

### Error Handling
- [How errors are propagated and recovered]
```

### Step 4: Go Implementation Recommendations

Based on the analysis, recommend:

1. **Go package placement**: Which `internal/` package should hold this?
2. **Interface design**: What Go interfaces should be defined?
3. **Key differences**: What should NOT be directly translated?
4. **Risks**: What parts are complex or error-prone?
5. **Priority**: What to implement first for a working MVP?

## Module Map Reference

For detailed file paths and module relationships, see [reference.md](reference.md).
