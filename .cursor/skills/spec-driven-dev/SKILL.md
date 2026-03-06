---
name: spec-driven-dev
description: Four-stage spec-driven development workflow for building new features. Use when the user asks to start building a new feature, begin a development sprint, or wants a structured approach to implement a major component.
---

# Spec-Driven Development (四阶段确认流)

## When to Use

- Starting development of a new major feature or component
- User says "let's build X" or "implement the Y module"
- Any feature that touches multiple packages or changes architecture

## Four-Stage Confirmation Flow

Each stage requires human confirmation before proceeding to the next.

```
Development Progress:
- [ ] Stage 1: Requirement Clarification
- [ ] Stage 2: Technical Design
- [ ] Stage 3: Plan-Driven Execution
- [ ] Stage 4: Living Doc Backfill
```

### Stage 1: Requirement Clarification (需求澄清)

**Goal**: Transform vague wishes into a concrete feature list.

Actions:
1. Analyze the CoPaw reference implementation (if applicable)
2. List all sub-features and capabilities
3. Identify edge cases and potential risks
4. Output a structured feature checklist

Output format:

```markdown
## Feature: [Name]

### Core Capabilities
- [ ] Capability 1: [description]
- [ ] Capability 2: [description]

### Dependencies
- Requires: [existing module X]
- Blocked by: [nothing / module Y not yet implemented]

### Risks & Open Questions
- Risk 1: [description]
- Question 1: [need user input on ...]
```

**STOP**: Present to user for review. Only proceed after confirmation.

### Stage 2: Technical Design (技术详设)

**Goal**: Produce an Implementation Plan that defines the architecture.

Actions:
1. Update `docs/architecture_spec.md` with new module placement
2. Design Go interfaces for the new feature
3. Update `docs/api_spec.md` with interface definitions
4. Create a Mermaid architecture/sequence diagram
5. List all files to be created/modified

Output format:

```markdown
## Implementation Plan: [Feature Name]

### Architecture Impact
[Mermaid diagram showing where this fits]

### Interface Definitions
[Go interface code blocks]

### Files to Create
- `internal/xxx/xxx.go` - [purpose]
- `internal/xxx/xxx_test.go` - [test scope]

### Files to Modify
- `cmd/gopherpaw/main.go` - [what changes]

### Execution Order
1. [First task]
2. [Second task]
...
```

**STOP**: Present to user for review. This is the "Tech Review" gate. Only proceed after approval.

### Stage 3: Plan-Driven Execution (计划驱动执行)

**Goal**: Implement feature according to the approved plan.

Execute the plan step by step:

1. Create interfaces first (contract-first)
2. Write test stubs for each interface method (TDD)
3. Implement each component
4. Run tests after each component
5. Wire up the full chain
6. Run all tests: `go test ./...`

Track progress using the execution order from Stage 2. Mark each step as complete.

Rules during execution:
- Do NOT deviate from the approved plan without user confirmation
- If you discover a design flaw, STOP and report it before changing the plan
- Write tests alongside (or before) implementation code

### Stage 4: Living Doc Backfill (活文档回填)

**Goal**: Ensure all documentation matches the final implementation.

Actions:
1. Verify `docs/architecture_spec.md` matches actual imports and dependencies
2. Verify `docs/api_spec.md` matches actual interface definitions in code
3. Fix any discrepancies (code is the source of truth)
4. Add changelog entry to relevant docs
5. Update `CONTEXT.md` with new progress status

Output: summary of documentation changes made.

## Key Principles

- **Human gates**: Never skip Stage 1->2 or Stage 2->3 transitions without user approval
- **Contract first**: Specs are updated before code is written
- **Test alongside**: Tests are written during Stage 3, not deferred
- **Code is truth**: In Stage 4, if docs and code conflict, update docs to match code
