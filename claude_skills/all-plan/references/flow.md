# All Plan

Collaborative planning with selected mounted CLIs for comprehensive solution design. Codex serves as the primary coordinator.

**Usage**: For complex features or architectural decisions requiring diverse perspectives.

---

## Input Parameters

From `$ARGUMENTS`:
- `requirement`: User's initial requirement or feature request
- `context`: Optional project context or constraints

---

## Execution Flow

### Phase 0: Participant Selection (Required)

Before any clarification, determine which CLIs will participate and who is the final decider.

1) **Check mounted CLIs** using the `mounted` skill.
2) **Ask the user to choose**:
   - Which AIs to include (subset of mounted CLIs)
   - Which AI is the final judge/arbiter
3) **Proceed only after explicit user selection.**

Record as:
```
participants: [list of chosen AIs]
arbiter: [chosen AI]
```

Define helper:
```
dispatch_to_participants(prompt, label):
  for each provider in participants:
    ask <provider> <<'EOF'
    [prompt]
    EOF
  save each response as "{provider}_{label}"
```

### Phase 1: Requirement Refinement & Project Analysis

**1.1 Structured Clarification (Option-Based)**

Use the **5-Dimension Planning Readiness Model** to ensure comprehensive requirement capture.

#### Readiness Dimensions (100 pts total)

| Dimension | Weight | Focus | Priority |
|-----------|--------|-------|----------|
| Problem Clarity | 30pts | What problem? Why solve it? | 1 |
| Functional Scope | 25pts | What does it DO? Key features | 2 |
| Success Criteria | 20pts | How to verify done? | 3 |
| Constraints | 15pts | Time, resources, compatibility | 4 |
| Priority/MVP | 10pts | What first? Phased delivery? | 5 |

#### Clarification Flow

```
ROUND 1:
  1. Parse initial requirement
  2. Identify 2 lowest-confidence dimensions (use Priority order for ties)
  3. Present 2 questions with options (1 per dimension)
  4. User selects options
  5. Update dimension scores based on answers
  6. Display Scorecard to user

IF readiness_score >= 80: Skip Round 2, proceed to 1.2
ELSE:
  ROUND 2:
    1. Re-identify 2 lowest-scoring dimensions
    2. Ask 2 more questions
    3. Update scores
    4. Proceed regardless (with gap summary)

QUICK-START OVERRIDE:
  - User can select "Proceed anyway" at any point
  - All dimensions marked as "assumption" in summary
```

#### Option Bank Reference

**Problem Clarity (30pts)**
```
Question: "What type of problem are you solving?"
Options:
  A. "Specific bug or defect with clear reproduction" → 27pts
  B. "New feature with defined business value" → 27pts
  C. "Performance/optimization improvement" → 24pts
  D. "General improvement or refactoring" → 18pts
  E. "Not sure yet - need exploration" → 9pts (flag)
  F. "Other: ___" → 12pts (flag)
```

**Functional Scope (25pts)**
```
Question: "What is the scope of functionality?"
Options:
  A. "Single focused component/module" → 23pts
  B. "Multiple related components" → 20pts
  C. "Cross-cutting system change" → 18pts
  D. "Unclear - need codebase analysis" → 10pts (flag)
  E. "Other: ___" → 10pts (flag)
```

**Success Criteria (20pts)**
```
Question: "How will you verify success?"
Options:
  A. "Automated tests (unit/integration/e2e)" → 18pts
  B. "Performance benchmarks with targets" → 18pts
  C. "Manual testing with checklist" → 14pts
  D. "User feedback/acceptance" → 12pts
  E. "Not defined yet" → 6pts (flag)
  F. "Other: ___" → 8pts (flag)
```

**Constraints (15pts)**
```
Question: "What are the primary constraints?"
Options:
  A. "Time-sensitive (deadline driven)" → 14pts
  B. "Must maintain backward compatibility" → 14pts
  C. "Resource/budget limited" → 12pts
  D. "Security/compliance critical" → 14pts
  E. "No specific constraints" → 10pts
  F. "Other: ___" → 8pts (flag)
```

**Priority/MVP (10pts)**
```
Question: "What is the delivery approach?"
Options:
  A. "MVP first, iterate later" → 9pts
  B. "Full feature, single release" → 9pts
  C. "Phased rollout planned" → 9pts
  D. "Exploratory - scope TBD" → 5pts (flag)
  E. "Other: ___" → 5pts (flag)
```

#### Gap Classification Rules

| Dimension Score | Classification | Handling |
|-----------------|----------------|----------|
| ≥70% of weight | ✓ Defined | Include in Design Brief |
| 50-69% of weight | ⚠️ Assumption | Carry forward as risk |
| <50% of weight | 🚫 Gap | Flag in brief, may need validation |

Example thresholds:
- Problem Clarity: ≥21 Defined, 15-20 Assumption, <15 Gap
- Functional Scope: ≥18 Defined, 13-17 Assumption, <13 Gap
- Success Criteria: ≥14 Defined, 10-13 Assumption, <10 Gap
- Constraints: ≥11 Defined, 8-10 Assumption, <8 Gap
- Priority/MVP: ≥7 Defined, 5-6 Assumption, <5 Gap

#### Clarification Summary Output

After clarification, generate:

```
CLARIFICATION SUMMARY
=====================
Readiness Score: [X]/100

Dimensions:
- Problem Clarity: [X]/30 [✓/⚠️/🚫]
- Functional Scope: [X]/25 [✓/⚠️/🚫]
- Success Criteria: [X]/20 [✓/⚠️/🚫]
- Constraints: [X]/15 [✓/⚠️/🚫]
- Priority/MVP: [X]/10 [✓/⚠️/🚫]

Assumptions & Gaps:
- [Dimension]: [assumption or gap description]
- [Dimension]: [assumption or gap description]

Proceeding to project analysis...
```

Save as `clarification_summary`.

**1.1.1 Optional Web Research (Smart Clarification)**

Use web research when the requirement depends on:
- external products/services/pricing
- latest APIs/standards
- domain norms/benchmarks

Search prompt template:
```
Goal: Identify realistic options, constraints, and unknowns for the user's requirement.
Return:
- 3-5 viable approaches (with sources)
- Critical constraints/limitations
- Common pitfalls or tradeoffs
```

Summarize findings into `clarification_summary` as:
```
Research Findings:
- Options: ...
- Constraints: ...
- Risks: ...
```

**1.2 Analyze Project Context**

Use available tools to understand:
- Existing codebase structure (Glob, Grep, Read)
- Current architecture patterns
- Dependencies and tech stack
- Related existing implementations

**1.3 Research (if needed)**

If the requirement involves:
- New technologies or frameworks
- Industry best practices
- Performance benchmarks
- Security considerations

Use WebSearch to gather relevant information.

**1.4 Formulate Complete Brief**

Create a comprehensive design brief incorporating clarification results:

```
DESIGN BRIEF
============
Readiness Score: [X]/100

Problem: [clear problem statement]
Context: [project context, tech stack, constraints]

Requirements:
- [requirement 1]
- [requirement 2]
- [requirement 3]

Success Criteria:
- [criterion 1]
- [criterion 2]

Assumptions (from clarification):
- [assumption 1]
- [assumption 2]

Gaps to Validate:
- [gap 1]
- [gap 2]

Research Findings: [if applicable]
```

Save as `design_brief`.

**1.5 Participant Clarification Round (Required)**

Send the current context to **all selected participants** and ask for clarification needs.

Prompt template:
```
You are a planning partner. Based on the requirement and current context:

KNOWN:
[known_facts]

UNKNOWN / AMBIGUOUS:
[unknowns]

CONSTRAINTS:
[constraints]

Provide:
1) 3-5 MUST-ASK clarification questions (each with 1-sentence rationale)
2) Optional assumptions (if user won't answer)
3) Risks if assumptions are wrong

Be specific. Avoid generic or boilerplate questions.
```

Use `dispatch_to_participants(prompt, "clarify_round_1")`.

**1.6 Codex Synthesis & User Questions**

Codex merges all participant questions, de-duplicates, and asks the user a short prioritized list (3-7 items). Save as `clarification_questions_round_1`.

---

### Phase 2: Iterative Clarification Loop (Multi-round)

Repeat until requirements are sufficiently clear.

For each round:
1) Incorporate user answers into `clarification_summary`.
2) Send updated context to **all selected participants** for additional clarification needs.
3) Codex merges and asks the next set of prioritized questions (3-5).
4) If user says "proceed", mark remaining gaps as assumptions and exit the loop.

Use `dispatch_to_participants(prompt, "clarify_round_N")` each round.

**Clarification Round Template (Codex → User)**
```
TOP QUESTIONS (please answer):
1) [question] (why it matters: ...)
2) [question] (why it matters: ...)
3) [question] (why it matters: ...)

If you want to proceed now, say: "Proceed with assumptions".
```

**Clarification Loop Stop Criteria**
- Requirements are actionable
- No blocking unknowns remain
- Remaining unknowns are recorded as assumptions

---

### Phase 3: Parallel Independent Design (All Participants)

Send the design brief to **all selected participants** for independent design.

**3.1 Dispatch to Claude**

```bash
ask claude <<'EOF'
Design a solution for this requirement:

[design_brief]

Provide:
- Goal (1 sentence)
- Architecture approach
- Implementation steps (3-7 key steps)
- Technical considerations
- Potential risks
- Acceptance criteria (max 3)

Be specific and concrete.
EOF
```

Save response as `claude_design` (only if Claude is selected).

**3.2 Dispatch to Gemini**

```bash
ask gemini <<'EOF'
Design a solution for this requirement:

[design_brief]

Provide:
- Goal (1 sentence)
- Architecture approach
- Implementation steps (3-7 key steps)
- Technical considerations
- Potential risks
- Acceptance criteria (max 3)

Be specific and concrete.
EOF
```

Wait for response. Save as `gemini_design` (only if Gemini is selected).

**3.3 Dispatch to OpenCode**

```bash
ask opencode <<'EOF'
Design a solution for this requirement:

[design_brief]

Provide:
- Goal (1 sentence)
- Architecture approach
- Implementation steps (3-7 key steps)
- Technical considerations
- Potential risks
- Acceptance criteria (max 3)

Be specific and concrete.
EOF
```

Wait for response. Save as `opencode_design` (only if OpenCode is selected).

**3.4 Codex's Independent Design**

While waiting for responses, create YOUR own design (do not look at others yet):
- Goal (1 sentence)
- Architecture approach
- Implementation steps (3-7 key steps)
- Technical considerations
- Potential risks
- Acceptance criteria (max 3)

Save as `codex_design`.

---

### Phase 4: Collect & Analyze All Designs

**4.1 Collect All Responses**

Gather designs for the selected participants:
- Claude design → `claude_design` (if selected)
- Gemini design → `gemini_design` (if selected)
- OpenCode design → `opencode_design` (if selected)
- Codex design → `codex_design`

**4.2 Comparative Analysis**

Analyze designs from the selected participants (including Codex):

Create a comparison matrix:
```
DESIGN COMPARISON
=================

1. Goals Alignment
   - Common goals across all designs
   - Unique perspectives from each

2. Architecture Approaches
   - Overlapping patterns
   - Divergent approaches
   - Pros/cons of each

3. Implementation Steps
   - Common steps (high confidence)
   - Unique steps (need evaluation)
   - Missing steps in some designs

4. Technical Considerations
   - Shared concerns
   - Unique insights from each CLI
   - Critical issues identified

5. Risk Assessment
   - Commonly identified risks
   - Unique risks from each perspective
   - Risk mitigation strategies

6. Acceptance Criteria
   - Overlapping criteria
   - Additional criteria to consider
```

Save as `comparative_analysis`.

---

### Phase 5: Iterative Refinement with All Participants

**5.1 Draft Merged Design**

Based on comparative analysis, create initial merged design:
```
MERGED DESIGN (Draft v1)
========================
Goal: [synthesized goal]

Architecture: [best approach from analysis]

Implementation Steps:
1. [step 1]
2. [step 2]
3. [step 3]
...

Technical Considerations:
- [consideration 1]
- [consideration 2]

Risks & Mitigations:
- Risk: [risk 1] → Mitigation: [mitigation 1]
- Risk: [risk 2] → Mitigation: [mitigation 2]

Acceptance Criteria:
- [criterion 1]
- [criterion 2]
- [criterion 3]

Open Questions:
- [question 1]
- [question 2]
```

Save as `merged_design_v1`.

**Merged Design Output Template (for participants)**
```
Goal:

Layered Plan:
Phase 1:
  - Step 1:
    - Subtasks:
  - Step 2:
    - Subtasks:
Phase 2:
  - Step 3:
    - Subtasks:

Dependencies:
- ...

Risks:
- ...

New Clarification Points:
- [if any]
```

**5.2 Discussion Round 1 - Review & Critique (All Participants)**

Send `comparative_analysis` + `merged_design_v1` to **all selected participants** for critique.
Use `dispatch_to_participants(prompt, "review_round_1")`.

```bash
ask {arbiter} <<'EOF'
Review this merged design based on all CLI inputs:

COMPARATIVE ANALYSIS:
[comparative_analysis]

MERGED DESIGN v1:
[merged_design_v1]

Analyze:
1. Does this design capture the best ideas from all perspectives?
2. Are there any conflicts or contradictions?
3. What's missing or unclear?
4. Are the implementation steps logical and complete?
5. Are risks adequately addressed?

Provide specific recommendations for improvement.
EOF
```

Save as `arbiter_review_1`.

**5.3 Discussion Round 2 - Resolve & Finalize (All Participants)**

Based on the arbiter's review, refine the design:

```bash
ask {arbiter} <<'EOF'
Refined design based on your feedback:

MERGED DESIGN v2:
[merged_design_v2]

Changes made:
- [change 1]
- [change 2]

Remaining concerns:
- [concern 1 if any]

Final approval or additional suggestions?
EOF
```

Save as `arbiter_review_2`.

---

### Phase 6: Final Output

**6.1 Finalize Design**

Incorporate feedback from all participants; arbiter has final decision.

**6.2 Save Plan Document**

Write the final plan to a markdown file:

**File path**: `plans/{feature-name}-plan.md`

Use this template:

```markdown
# {Feature Name} - Solution Design

> Generated by all-plan collaborative design process (Codex-led)

## Overview

**Goal**: [Clear, concise goal statement]

**Readiness Score**: [X]/100

**Generated**: [Date]

---

## Requirements Summary

### Problem Statement
[Clear problem description]

### Scope
[What's in scope and out of scope]

### Success Criteria
- [ ] [criterion 1]
- [ ] [criterion 2]
- [ ] [criterion 3]

### Constraints
- [constraint 1]
- [constraint 2]

### Assumptions
- [assumption 1 from clarification]
- [assumption 2 from clarification]

---

## Architecture

### Approach
[Chosen architecture approach with rationale]

### Key Components
- **[Component 1]**: [description]
- **[Component 2]**: [description]

### Data Flow
[If applicable, describe data flow]

---

## Implementation Plan

### Step 1: [Title]
- **Actions**: [specific actions]
- **Deliverables**: [what will be produced]
- **Dependencies**: [what's needed first]

### Step 2: [Title]
- **Actions**: [specific actions]
- **Deliverables**: [what will be produced]
- **Dependencies**: [what's needed first]

### Step 3: [Title]
- **Actions**: [specific actions]
- **Deliverables**: [what will be produced]
- **Dependencies**: [what's needed first]

[Continue for all steps...]

---

## Technical Considerations

- [consideration 1]
- [consideration 2]
- [consideration 3]

---

## Risk Management

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| [risk 1] | High/Med/Low | High/Med/Low | [strategy] |
| [risk 2] | High/Med/Low | High/Med/Low | [strategy] |

---

## Acceptance Criteria

- [ ] [criterion 1]
- [ ] [criterion 2]
- [ ] [criterion 3]

---

## Design Contributors

| CLI | Key Contributions |
|-----|-------------------|
| Codex | [contributions] |
| Claude | [contributions if selected] |
| Gemini | [contributions if selected] |
| OpenCode | [contributions if selected] |

---

## Appendix

### Clarification Summary
[Include the clarification summary from Phase 1.1]

### Alternative Approaches Considered
[Brief notes on approaches that were evaluated but not chosen]
```

**6.3 Output to User**

After saving the file, display to user:

```
PLAN COMPLETE
=============

✓ Plan saved to: plans/{feature-name}-plan.md

Summary:
- Goal: [1-sentence goal]
- Steps: [N] implementation steps
- Risks: [N] identified with mitigations
- Readiness: [X]/100

Next: Review the plan and proceed with implementation when ready.
```

---

## Principles

1. **Structured Clarification**: Use option-based questions to systematically capture requirements
2. **Readiness Scoring**: Quantify requirement completeness before proceeding
3. **True Independence**: All CLIs design independently without seeing others' work first
4. **Diverse Perspectives**: Leverage unique strengths of each selected CLI and dispatch to all participants whenever external input is needed
5. **Evidence-Based Synthesis**: Merge based on comparative analysis, not arbitrary choices
6. **Iterative Refinement**: Use arbiter discussion to validate and improve merged design
7. **Concrete Deliverables**: Output actionable plan document, not just discussion notes
8. **Attribution**: Acknowledge contributions from each CLI to maintain transparency
9. **Research When Needed**: Don't hesitate to use WebSearch for external knowledge
10. **Max 2 Iteration Rounds**: Avoid endless discussion; converge on practical solution
11. **Document Output**: Always save final plan as markdown file

---

## Notes

- This skill is designed for complex features or architectural decisions
- For simple tasks, use dual-design or direct implementation instead
- Use `ask <provider>` for all providers (no extra parameters)
- If any CLI is not available, proceed with available CLIs and note the absence
- Plans are saved to `plans/` directory with descriptive filenames
