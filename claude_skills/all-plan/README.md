# All-Plan Skill

Collaborative planning with selected mounted CLIs for comprehensive solution design. Codex acts as coordinator.

## Usage

```
/all-plan <your requirement or feature request>
```

Example:
```
/all-plan Design a caching layer for the API with Redis
```

## How It Works

**6-Phase Collaborative Design Process:**

1. **Participant Selection** - Choose which AIs to include + final arbiter (via `mounted`)
2. **Requirement Refinement** - Multi-round clarification, participant-driven questions
3. **Parallel Independent Design** - All selected AIs design independently
4. **Comparative Analysis** - Merge and compare insights from all participants
5. **Iterative Refinement** - All participants review; arbiter decides
6. **Final Output** - Layered, actionable implementation plan

## Key Features

- **Participant Selection**: User chooses participating AIs and final arbiter
- **Multi-round Clarification**: Questions gathered from all participants, merged by Codex
- **Optional Web Research**: Triggered when requirements depend on external info
- **Ask-Only Dispatch**: Uses `ask <provider>` for all participants
- **Layered Plans**: Phases → steps → subtasks with dependencies and risks

## When to Use

- Complex features requiring diverse perspectives
- Architectural decisions with multiple valid approaches
- High-stakes implementations needing thorough validation

## Output

A comprehensive plan including:
- Goal and architecture with rationale
- Layered implementation plan (phases → steps → subtasks)
- Dependencies and risks
- Acceptance criteria
- Design contributors from each selected AI
