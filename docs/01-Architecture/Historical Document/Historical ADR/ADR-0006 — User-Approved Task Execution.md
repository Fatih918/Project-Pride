# ADR-0006 — User-Approved Task Execution

**Status:** Accepted

---

## Context

PRIDE is designed to automate tasks across the operating system, external services, AI providers, and the plugin ecosystem.

Automation without transparency can reduce user trust, make actions difficult to understand, and increase the risk of unintended behavior.

A consistent execution model is therefore required.

---

## Decision

PRIDE adopts **User-Approved Task Execution** as the standard execution model.

Before executing meaningful actions on behalf of a user, the **Planning** component must generate a **Task Tree** describing the intended execution.

Execution begins only after explicit user approval.

---

## Principles

The generated Task Tree should clearly describe:

* What will happen
* Why it is necessary
* Which Plugins will participate
* Which permissions will be requested
* The expected outcome

The objective is transparency rather than hidden automation.

---

## Human Approval

Meaningful actions require explicit user approval before execution.

Examples include:

* Installing Plugins
* Installing software
* Importing data
* Exporting data
* Modifying system configuration
* Executing shell commands
* Accessing protected resources
* Connecting external accounts

Approval always occurs **after Planning completes** and **before execution begins**.

---

## Approval Scope

User approval applies only to the generated **Task Tree**.

If execution requires Tasks outside the approved Task Tree,

or if the Task Tree changes,

Planning must generate a new Task Tree and request user approval again before execution may continue.

---

## Execution

Once approved,

the Executor may execute the approved Task Tree without requesting confirmation for every individual Task.

The Executor must never perform actions outside the approved Task Tree.

---

## Execution Summary

After execution completes,

PRIDE should provide a summary including:

* Completed Tasks
* Failed Tasks
* Skipped Tasks
* Generated files
* Configuration changes
* Recommendations when appropriate

---

## Rationale

User approval increases transparency while preserving automation.

Users remain informed and retain authority over meaningful system actions without sacrificing convenience.

The Task Tree also provides predictable execution behavior across the entire PRIDE Core.

---

## Consequences

### Positive

* Greater transparency
* Consistent execution
* Improved user trust
* Reduced accidental actions
* Predictable platform behavior
* Stable execution lifecycle

### Trade-offs

* Certain workflows require an additional approval step.
* Planning introduces implementation complexity.
* Plugins should clearly describe intended actions.

These trade-offs are considered acceptable in exchange for safer and more understandable automation.

---

## Alternatives Considered

### Immediate Execution

Allowing Plugins or AI agents to execute actions immediately after receiving a request.

**Rejected.**

This reduces transparency and increases the possibility of unintended behavior.

---

### Confirmation Before Every Task

Requesting approval before every Task inside the Task Tree.

**Rejected.**

This unnecessarily interrupts workflows and creates poor user experience.

---

## Related Foundation

* Transparency
* Human Authority
* Ethics
* Privacy

---

## Related ADR

* ADR-0003 — Event-Driven Architecture
* ADR-0004 — Permission-Based Security Model
* ADR-0005 — Provider Abstraction Layer

---

This decision establishes the standard execution model for the PRIDE Core.

Planning is responsible for producing a complete Task Tree.

The user is responsible for approving that Task Tree.

The Executor is responsible for executing only the approved Task Tree.

Automation should always remain understandable, reviewable, and ultimately controlled by the user.
