# Why Does PRIDE Have So Many Documents?

If you've just opened the PRIDE repository, you might wonder why there are so many documents before you even see the source code.

The short answer is simple.

Every document answers a different question.

None of them exist to repeat one another.

Instead, they form a chain where each document naturally leads to the next.

Understanding this relationship is far more important than reading every file in order.

---

# The Flow of Ideas

```text
Philosophy
    ↓
Manifesto
    ↓
Foundation
    ↓
Architecture Decisions (ADR)
    ↓
Workflow
    ↓
Implementation
```

This is not just a folder structure.

It is the evolution of an idea.

Every layer exists because the previous layer made it necessary.

---

# Philosophy

Everything begins with philosophy.

It defines how PRIDE views technology, people, and progress.

It does not describe features.

It does not describe architecture.

It defines the mindset that should remain true regardless of how technology changes.

---

# Manifesto

The Manifesto transforms philosophy into commitments.

If the Philosophy explains how PRIDE thinks, the Manifesto explains what PRIDE promises to become.

These commitments define the direction of the entire project.

---

# Foundation

The Foundation converts those commitments into principles.

Instead of describing ideals, it provides rules that guide future design decisions.

Whenever uncertainty exists, the Foundation becomes the reference point.

---

# Architecture Decision Records (ADR)

The ADR explains why architectural decisions were made.

It does not introduce new principles.

Instead, it records how the Foundation influenced real technical decisions.

Every significant architectural choice should be traceable back to the Foundation.

---

# Workflow

Once decisions exist, they must work together.

The Workflow explains how different parts of PRIDE interact during execution.

Rather than describing individual components, it explains how responsibilities are distributed across the system.

---

# Implementation

Implementation is where every previous document finally becomes software.

The code should never define the philosophy.

The philosophy should define the code.

---

# Continue Reading

The documentation is intentionally layered.

Instead of asking,

> "What should I read next?"

Try asking,

> "What question am I trying to answer?"

Each document exists to answer one specific question.

Together, they explain not only how PRIDE is built, but why it is built that way.
# How Every Document Connects

PRIDE's documentation is not a collection of independent files.

Each document exists because another document came before it.

Removing one layer makes every layer below it harder to understand.

This relationship forms the reasoning process behind the entire project.

---

# Philosophy → Manifesto

Everything begins with Philosophy.

The Philosophy defines how PRIDE views technology, humanity, and progress.

However, a philosophy alone cannot guide a project.

It must become a commitment.

That commitment becomes the Manifesto.

If the Philosophy answers,

> "How do we think?"

The Manifesto answers,

> "What do we promise?"

---

# Manifesto → Foundation

A commitment is still too broad to build software.

The Foundation transforms those commitments into principles that can guide future development.

Whenever a new feature, workflow, or architectural decision is proposed, it should remain consistent with these principles.

If the Manifesto answers,

> "What do we believe?"

The Foundation answers,

> "How should we build?"

---

# Foundation → ADR

Principles alone cannot explain individual technical decisions.

Every significant architectural choice should have a documented reason.

That reason is recorded inside an Architecture Decision Record (ADR).

Each ADR should be understandable because the Foundation already explains the principles behind it.

If the Foundation answers,

> "What principles guide us?"

The ADR answers,

> "Why was this decision made?"

---

# ADR → Workflow

An architectural decision is valuable only when it becomes part of a working system.

The Workflow explains how multiple architectural decisions cooperate during execution.

Instead of focusing on isolated components, it explains how responsibilities move throughout PRIDE.

If the ADR answers,

> "Why was this designed?"

The Workflow answers,

> "How does it work together?"

---

# Workflow → Implementation

The implementation is the final result of every previous document.

The source code should not invent new principles.

It should implement existing ones.

Whenever implementation changes, the Philosophy, Manifesto, Foundation, ADR, and Workflow should continue to remain consistent.

The code is not the beginning of PRIDE.

It is the final expression of everything defined before it.

---

# One Continuous Thought

PRIDE is intentionally documented from abstract ideas toward concrete implementation.

```text
Philosophy
    ↓
Manifesto
    ↓
Foundation
    ↓
ADR
    ↓
Workflow
    ↓
Implementation
```

Every document answers a different question.

Together, they explain not only *what* PRIDE is, but *why* it exists and *how* it should continue to evolve.
# Workflow

Software is more than a collection of components.

Components only become a system when they cooperate toward a shared objective.

PRIDE follows the same principle.

The Workflow exists to describe how every responsibility moves throughout the platform.

It is not an implementation guide.

It is not source code.

It is not an API reference.

Instead, it explains how every major architectural decision cooperates to transform a user's request into a completed result while remaining consistent with the Philosophy, Manifesto, Foundation, and Architecture Decision Records.

Every request follows the same workflow.

Regardless of whether the user asks a simple question, automates a complex task, generates code, edits a document, or controls an external application, the request always enters and leaves PRIDE through the same execution path.

This consistency is intentional.

PRIDE does not create different workflows for different features.

Instead, it extends capabilities while preserving one stable execution model.

This allows the Core to remain predictable while the ecosystem continues to evolve.

---

# One Workflow

PRIDE intentionally maintains a single execution workflow.

Plugins do not create their own workflows.

SDKs do not introduce alternative execution paths.

Providers do not change how the platform operates.

Only the implementation inside each component may evolve.

The workflow itself remains stable.

This ensures that every improvement made to one part of the system benefits every capability built upon it.

Stable execution creates predictable behavior.

Predictable behavior creates trust.

---

# The Complete Workflow

```text
User
   │
   ▼
UI
   │
   ▼
Memory Retrieval
   │
   ▼
sdk.ai.retrieve()
   │
   ▼
Provider
   │
   ▼
Relevant Memories
   │
   ▼
Context Builder
   │
   ▼
sdk.ai.context()
   │
   ▼
Provider
   │
   ▼
Complete Context
   │
   ▼
Planning (Core)
   │
   ▼
sdk.ai.plan()
   │
   ▼
Provider
   │
   ▼
Task Tree
   │
   ▼
Capability Registry
   │
   ▼
sdk.ai.match()
   │
   ▼
Provider
   │
   ▼
Selected Plugin
   │
   ▼
Permission Manager
   │
   ├───────────────┐
   │ Permission OK │ Permission Denied
   ▼               ▼
Executor          UI
   │
   ▼
Execute Current Task
   │
   ▼
Plugin
   │
   ├───────────────┐
   │ Need AI?      │ No
   ▼               ▼
sdk.ai.reason()   Execute
   │
   ▼
Provider
   │
   ▼
Plugin Result
   │
   ▼
Memory Update
   │
   ▼
sdk.ai.organize()
   │
   ▼
Provider
   │
   ▼
Updated Memory
   │
   ▼
UI
   │
   ▼
User
```

---

# Understanding The Workflow

The complete workflow above represents the entire lifecycle of a request.

Although it appears large, every component has only one responsibility.

Instead of solving many different problems, every stage solves exactly one problem before handing the result to the next stage.

This separation keeps the Core stable while allowing every individual component to evolve independently.

The following sections zoom into each stage individually.

---

# User

The workflow always begins with the user.

Everything inside PRIDE exists to help people accomplish tasks more naturally.

The user never interacts directly with Providers, Plugins, SDKs, or internal components.

All interaction begins through the User Interface.

---

# User Interface (UI)

The UI receives the user's request.

Its responsibility is communication.

It does not perform planning.

It does not search memory.

It does not execute plugins.

It simply forwards the request into the Core and later presents the final result back to the user.

The UI is also responsible for displaying progress, requesting permissions, reporting errors, and allowing users to cancel running tasks.

---

# Memory Retrieval

Before planning begins, PRIDE searches for information that may already exist.

Instead of relying on simple keyword matching, Memory Retrieval uses Artificial Intelligence to locate information that is semantically relevant to the current request.

The goal is not to retrieve everything.

The goal is to retrieve only what is most useful for the current task.

Every memory remains stored.

Only the relevant memories are forwarded.

---

# Context Builder

The Context Builder combines:

* User Input
* Relevant Memories

into a complete context.

This stage prepares information for planning.

Rather than simply concatenating data together, it organizes information so the Planner receives only the context necessary to understand the user's objective.

---

# Planning

Planning is part of the Core.

It is intentionally not implemented as a Plugin.

Planning converts the prepared context into a Task Tree.

The Task Tree represents the complete execution strategy.

It defines what must happen.

It does not decide how each task will be implemented.

---

# Capability Registry

The Capability Registry receives the Task Tree.

Its responsibility is discovering which installed Plugin is best suited to execute each task.

Artificial Intelligence is used here because selecting the most appropriate capability often requires semantic reasoning rather than simple keyword comparison.

The Registry never installs Plugins.

It never updates Plugins.

It only selects among the Plugins already available inside the user's environment.

---

# Permission Manager

Before execution begins, every requested capability passes through the Permission Manager.

The Permission Manager is entirely deterministic.

Artificial Intelligence is never used.

Its responsibilities include validating access to:

* Files
* Internet
* Camera
* Microphone
* System Operations
* Sensitive User Data

If permission is denied, execution stops and the UI notifies the user.

If permission is granted, execution continues.

---

# Executor

The Executor coordinates execution.

It reads the Task Tree sequentially.

Tasks are intentionally executed one at a time.

Sequential execution provides:

* predictable behavior
* easier debugging
* lower hardware requirements
* stable memory updates

The Executor never performs reasoning.

Its responsibility is orchestration.

---

# Plugin

Plugins perform domain-specific work.

A Plugin may:

* execute directly

or

* request reasoning through sdk.ai.

The decision depends entirely on the Plugin's implementation.

Plugins never modify the global workflow.

They simply receive input and produce output.

---

# Memory Update

After successful execution, PRIDE updates its memory.

Artificial Intelligence organizes new information by producing metadata such as:

* Title
* Summary
* Tags

This improves future retrieval quality.

The original information remains stored.

Nothing is discarded.

Users remain the owners of their own memories.

---

# Progress

Progress reporting is deterministic.

Artificial Intelligence is not involved.

The UI displays execution status using information provided by the Executor.

Plugins that support detailed progress may publish additional execution states.

Plugins that do not support progress simply display a running state.

---

# Cancellation

Users may cancel execution at any time.

The cancellation request travels through the UI to the Executor.

The Executor attempts to stop the currently running Plugin.

If the running operation cannot be interrupted safely, the result is discarded after completion.

User control always takes priority.

---

# Artificial Intelligence Throughout The Workflow

Artificial Intelligence is intentionally limited to stages that require reasoning.

Current AI-assisted stages include:

* Memory Retrieval
* Context Builder
* Planning
* Capability Registry
* Plugin Reasoning (optional)
* Memory Update

Everything else remains deterministic.

This separation minimizes unnecessary AI usage while preserving flexibility where reasoning provides measurable value.

---

# Workflow Principles

Every implementation inside PRIDE should preserve the following principles.

* One Stable Workflow
* Stable Core
* Sequential Execution
* Single Responsibility
* AI Only Where Reasoning Is Required
* Deterministic Systems Remain Deterministic
* Plugins Never Control The Workflow
* Users Control Their Providers
* Users Own Their Memories
* SDKs Abstract Implementations
* Transparency Before Automation

Whenever future changes are proposed, they should strengthen these principles rather than replace them.

---

# Continue Reading

The Workflow explains how responsibilities move throughout PRIDE.

The next documents explain how those responsibilities are implemented.

Workflow

↓

SDK

↓

Plugins

↓

Implementation

The architecture defines the flow.

The implementation brings that flow to life.
# User

Every workflow begins with the user.

Without a request, no execution occurs.

The user defines the objective.

PRIDE defines the process.

The user never interacts directly with internal components such as Memory, Providers, SDKs, Plugins, or the Core.

Instead, every interaction enters the platform through the User Interface.

This separation allows the internal architecture to evolve without changing how people interact with PRIDE.

---

```text
User
   │
   ▼
UI
```

### Responsibility

The user's only responsibility is expressing an objective.

Examples include:

* Asking a question.
* Creating a document.
* Building software.
* Editing files.
* Executing automation.
* Controlling external applications.

The user never needs to understand how the request will be executed internally.

That responsibility belongs entirely to PRIDE.

---

### Input

The user may provide:

* Natural language.
* Files.
* Images.
* Audio.
* Structured parameters.
* Previous conversation context.

Multiple input types may be combined into a single request.

---

### Output

The output of this stage is simply the user's request being delivered to the User Interface.

No processing occurs at this stage.

No reasoning occurs at this stage.

The user only defines intent.

---

### Why This Stage Exists

Separating the user from the internal architecture ensures that changes inside PRIDE never require users to learn a completely different way of interacting with the platform.

Human interaction remains stable.

Implementation remains flexible.

This reflects one of PRIDE's core principles.

Technology should adapt to people.

People should not adapt to technology.

---

# User Interface (UI)

The User Interface is the communication layer between humans and the PRIDE Core.

Its purpose is not to make decisions.

Its purpose is to translate interaction into a format that the Core can process and later present the results back to the user.

The UI should remain as lightweight as possible.

Business logic belongs inside the Core.

Execution belongs inside Plugins.

Reasoning belongs inside Artificial Intelligence Providers.

The UI exists only to communicate.

---

```text
User
   │
   ▼
UI
   │
   ▼
Memory Retrieval
```

### Responsibility

The User Interface is responsible for:

* Receiving user requests.
* Sending requests into the Core.
* Displaying execution progress.
* Displaying execution results.
* Displaying warnings and errors.
* Requesting permissions from the user.
* Allowing users to cancel running tasks.
* Allowing users to configure Providers.
* Allowing users to manage Plugins.
* Allowing users to manage Memories through dedicated Memory Plugins.

The UI never performs execution itself.

---

### Input

The UI receives:

* User requests.
* User interactions.
* User configuration.
* Files and media.

---

### Output

The UI forwards the request to the Memory Retrieval stage.

No planning occurs.

No Plugin selection occurs.

No Provider selection occurs.

The UI simply forwards the request into the Core.

---

### Why The UI Does Not Perform Reasoning

Reasoning belongs inside Artificial Intelligence.

Execution belongs inside the Core.

The User Interface should remain predictable regardless of which Provider, Plugin, or SDK is installed.

Keeping the UI lightweight allows PRIDE to support different interfaces without changing the underlying architecture.

For example:

* Desktop applications.
* Mobile applications.
* Web applications.
* Terminal interfaces.
* Voice interfaces.

Every interface should follow the same internal workflow.

Only the presentation changes.

The architecture does not.
# Memory Retrieval

Memory Retrieval is the first intelligent stage inside the PRIDE workflow.

Before attempting to solve a problem, PRIDE first asks an important question.

**"Have we learned something that can help?"**

Instead of immediately planning from scratch, PRIDE searches its Memory to discover knowledge that may already exist.

This may include:

* Previous conversations.
* User preferences.
* Existing projects.
* Long-term goals.
* Previous execution results.
* Documents.
* Saved workflows.
* Historical decisions.
* Any other memory owned by the user.

The objective is not to retrieve everything.

The objective is to retrieve only the information that is most relevant to the current request.

This improves both efficiency and accuracy while reducing unnecessary reasoning later in the workflow.

---

```text
UI
 │
 ▼
Memory Retrieval
 │
 ▼
sdk.memory.retrieve()
 │
 ▼
sdk.ai.retrieve()
 │
 ▼
Provider
 │
 ▼
Relevant Memories
 │
 ▼
Context Builder
```

---

## Responsibility

Memory Retrieval is responsible for:

* Searching user memories.
* Finding semantically relevant information.
* Ranking retrieved memories by relevance.
* Returning only the memories that help solve the current request.

Memory Retrieval is **not** responsible for:

* Building context.
* Planning.
* Executing tasks.
* Updating memories.
* Deciding what action should be taken.

Its only responsibility is retrieval.

---

## Input

Memory Retrieval receives:

* Raw user request.
* User identifier.
* Memory database reference.

The request has not yet been interpreted into a complete context.

At this stage, PRIDE only knows what the user has asked.

---

## Internal Process

Memory Retrieval follows two stages.

### Stage 1 — Memory Search

The Memory SDK searches every available memory source.

Examples include:

* Personal memories.
* Conversation history.
* Project memories.
* Plugin memories.
* Long-term memories.
* Workspace memories.

The SDK collects candidate memories that may be related to the request.

At this point, no reasoning has occurred.

The SDK simply gathers possible candidates.

---

### Stage 2 — Semantic Retrieval

After candidate memories have been collected, Artificial Intelligence determines which memories are actually useful.

This process is performed through:

```text
sdk.ai.retrieve()
```

The selected Provider performs semantic reasoning to answer questions such as:

* Which memories are most relevant?
* Which memories are outdated?
* Which memories should be ignored?
* Which memories best represent the user's current objective?

The Provider returns only the memories that are expected to improve later reasoning.

This prevents unnecessary information from polluting the Context Builder.

---

## Why Artificial Intelligence Is Used

Simple keyword matching is insufficient.

Consider the following example.

The user writes:

> "Continue working on PRIDE."

The word "PRIDE" may appear inside hundreds of memories.

Artificial Intelligence understands that the request is likely referring to:

* The current software architecture.
* The most recent workflow discussion.
* Active documentation.
* Ongoing design decisions.

Rather than every memory containing the word "PRIDE."

Semantic retrieval produces significantly more useful context than keyword retrieval alone.

---

## Why Everything Is Still Stored

Memory Retrieval never deletes memories.

Every memory remains owned by the user.

Artificial Intelligence only decides which memories should participate in the current workflow.

The remaining memories continue to exist and may become relevant for future requests.

Retrieval affects visibility.

It does not affect ownership.

---

## Output

The output of Memory Retrieval is:

* User request.
* Ranked relevant memories.

Nothing else.

No context has been built.

No planning has occurred.

No execution has started.

The retrieved memories are forwarded to the Context Builder, where they will be combined into a complete reasoning context.

---

## Design Principles

Memory Retrieval follows several architectural principles.

* Memory belongs to the user.
* Retrieval never modifies memory.
* Retrieval only selects.
* Artificial Intelligence performs semantic ranking.
* The SDK abstracts every storage implementation.
* The Core never accesses memory implementations directly.

These principles allow PRIDE to evolve its memory system without changing the workflow itself.
# Context Builder

After relevant memories have been retrieved, PRIDE still does not fully understand the user's objective.

At this point, two independent pieces of information exist.

The first is the user's request.

The second is the collection of relevant memories returned by Memory Retrieval.

Neither is sufficient on its own.

The user request lacks historical context.

The retrieved memories lack the user's current intention.

The Context Builder exists to combine both into a single coherent understanding before planning begins.

Rather than simply concatenating text together, the Context Builder organizes information into a structured context that can be understood by the Planning engine.

---

```text
Relevant Memories
        │
        │
User Request
        │
        ▼
Context Builder
        │
        ▼
sdk.ai.context()
        │
        ▼
Provider
        │
        ▼
Complete Context
        │
        ▼
Planning
```

---

## Responsibility

The Context Builder is responsible for:

* Combining the current request with retrieved memories.
* Removing redundant information.
* Resolving conflicting information whenever possible.
* Preserving important historical context.
* Producing a clean reasoning context for Planning.

The Context Builder is **not** responsible for:

* Creating execution plans.
* Selecting Plugins.
* Executing tasks.
* Updating Memory.

Its only responsibility is producing the best possible context.

---

## Input

The Context Builder receives:

* The original user request.
* Relevant memories from Memory Retrieval.

No additional information is introduced at this stage.

The Context Builder only works with information already available inside the workflow.

---

## Internal Process

The Context Builder performs several logical steps.

### Stage 1 — Merge

The user's request and relevant memories are merged into a temporary working context.

This allows relationships between present and past information to be understood together.

---

### Stage 2 — Context Organization

Artificial Intelligence reorganizes the merged information into something easier for Planning to reason about.

Examples include:

* Identifying the primary objective.
* Identifying supporting information.
* Identifying constraints.
* Identifying user preferences.
* Identifying previously completed work.
* Identifying missing information.

The objective is clarity rather than completeness.

Planning should receive only information that helps produce better decisions.

---

### Stage 3 — Noise Reduction

Not every retrieved memory remains useful after merging.

Some memories become irrelevant once the complete context has been constructed.

Artificial Intelligence removes unnecessary information while preserving everything that contributes to the current objective.

This keeps later reasoning efficient and reduces unnecessary token usage.

---

## Why Artificial Intelligence Is Used

Building context is fundamentally a reasoning problem.

Consider the following request.

> "Continue the Workflow document."

Relevant memories may include:

* SDK discussions.
* Architecture discussions.
* Philosophy.
* Manifesto.
* Previous workflow decisions.
* Plugin API ideas.
* Memory system discussions.

Only some of these are useful for the current request.

Artificial Intelligence determines:

* Which information directly supports the user's objective.
* Which information is only background knowledge.
* Which information should not participate in planning.

This cannot be solved reliably using deterministic rules alone.

---

## Why The Context Builder Exists Separately

It may seem possible for Planning to build its own context.

PRIDE intentionally separates these responsibilities.

The Context Builder prepares information.

Planning makes decisions.

Keeping these responsibilities independent provides several advantages.

* Planning becomes simpler.
* Context generation can evolve independently.
* Different Planning implementations may reuse the same Context Builder.
* Future improvements to context construction do not require changes to Planning.

This follows the Single Responsibility Principle adopted throughout PRIDE.

---

## Output

The output of the Context Builder is:

* A complete reasoning context.

This context contains:

* User objective.
* Relevant memories.
* Constraints.
* Preferences.
* Historical information.
* Supporting knowledge.

The Context Builder does not determine how the problem should be solved.

It only ensures that Planning receives the best possible understanding of the user's situation.

---

## Design Principles

The Context Builder follows several architectural principles.

* Context is constructed before planning.
* Context generation uses semantic reasoning.
* Only relevant information is forwarded.
* Responsibilities remain separated.
* Planning never performs context construction.
* Context is temporary and exists only for the current workflow.

Once the Complete Context has been produced, it is forwarded to the Planning stage, where execution strategy begins.
# Planning

Planning is the first stage where PRIDE begins making decisions.

Up until this point, the system has only gathered information.

Memory Retrieval searched for relevant knowledge.

The Context Builder organized that knowledge into a coherent understanding.

Neither component decides what should happen next.

That responsibility belongs exclusively to Planning.

Planning transforms understanding into execution.

Its purpose is to convert the Complete Context into a structured Task Tree that the remainder of the workflow can execute predictably.

For this reason, Planning is part of the Core.

It is not implemented as a Plugin.

---

```text
Complete Context
        │
        ▼
Planning (Core)
        │
        ▼
sdk.ai.plan()
        │
        ▼
Provider
        │
        ▼
Task Tree
        │
        ▼
Capability Registry
```

---

## Responsibility

Planning is responsible for:

* Understanding the user's objective.
* Breaking complex requests into executable tasks.
* Determining task order.
* Identifying task dependencies.
* Producing a Task Tree.

Planning is **not** responsible for:

* Selecting Plugins.
* Checking Permissions.
* Executing Tasks.
* Updating Memory.
* Communicating with external software.

Planning defines **what should happen**, not **how it happens**.

---

## Input

Planning receives exactly one input.

The Complete Context.

This context already contains:

* The user's objective.
* Relevant memories.
* User preferences.
* Historical knowledge.
* Existing constraints.
* Supporting information.

Planning never searches Memory directly.

Planning never builds Context.

Planning trusts previous stages to prepare everything it needs.

---

## Internal Process

Planning performs several reasoning steps.

### Stage 1 — Objective Identification

The Provider determines the primary objective.

Example:

User says:

> "Continue writing Workflow.md."

Planning understands that the primary objective is:

Continue documentation rather than designing a new architecture or modifying existing code.

---

### Stage 2 — Task Decomposition

Large objectives are divided into smaller executable tasks.

Example:

```text
Continue Workflow.md

↓

Write Planning section

↓

Write Capability Registry section

↓

Write Permission Manager section

↓

Finish document
```

Every task should represent a clear unit of work.

---

### Stage 3 — Dependency Analysis

Some tasks depend on others.

Planning identifies those relationships.

Example:

```text
Planning

↓

Capability Registry

↓

Permission Manager

↓

Executor
```

Executor cannot be documented before Planning because it depends on Planning.

Planning records these dependencies inside the Task Tree.

---

### Stage 4 — Task Prioritization

Tasks are arranged into execution order.

PRIDE intentionally executes tasks sequentially.

Therefore Planning must produce an ordered Task Tree.

No parallel execution is assumed inside the Core.

---

## Why Artificial Intelligence Is Used

Planning is fundamentally a reasoning problem.

Consider the request:

> "Build a weather application."

Planning must determine:

* Does the user need a UI?
* Does the user need an API?
* Does the user need data storage?
* Does the user need testing?
* Does the user need documentation?

There is no deterministic rule capable of solving every possible request.

Artificial Intelligence provides the flexibility required to convert human objectives into executable plans.

---

## Why Planning Is Part Of The Core

Planning is intentionally implemented inside the Core.

Not as a Plugin.

The reason is architectural stability.

Every workflow requires Planning.

If Planning were implemented as a Plugin:

* The workflow itself would depend on an external capability.
* Different Planning Plugins could produce incompatible execution models.
* The Core would lose control over its own architecture.

Keeping Planning inside the Core guarantees that every request follows the same execution philosophy regardless of which Plugins are installed.

Plugins extend capabilities.

Planning defines architecture.

Architecture belongs inside the Core.

---

## Output

Planning produces a Task Tree.

The Task Tree describes:

* Every task that must be completed.
* The order of execution.
* Dependencies between tasks.

The Task Tree contains no implementation details.

It does not reference Plugins.

It does not reference Providers.

It only describes work that needs to be completed.

Selecting how that work will be completed belongs to the next stage.

---

## Task Metadata Generation

The Planning phase is responsible for producing complete task definitions.

Every task generated during planning shall include both its execution structure and its associated metadata.

Task metadata is considered part of the planning result rather than a separate post-processing step.

At minimum, every generated task shall contain:

* Task Identifier
* Human-readable Title
* Execution Goal
* Capability Reference
* Parent Relationship
* Child Relationships
* Dependency Information
* Initial Execution State

The human-readable Title represents a concise description of the task's primary objective.

Titles are generated as part of the planning process to improve readability throughout the platform.

The Title is informational only.

Task execution shall never depend on the Title itself.

After planning has completed, the generated Task Tree, including all task metadata, is persisted through the Memory SDK.

The Memory SDK and Memory Module shall store task metadata exactly as produced by the Planning phase.

Neither the Memory SDK nor the Memory Module shall generate, modify, summarize, or reinterpret task metadata during persistence.

Subsequent modifications to task metadata may occur only through explicit platform operations defined by other specifications.

---

## Example Task Tree

```text
Goal
│
├── Task 1
│
├── Task 2
│
├── Task 3
│
└── Task 4
```

Each task is intentionally implementation-independent.

This allows different Plugins to solve the same task without changing the overall workflow.

---

## Design Principles

Planning follows several architectural principles.

* Planning belongs inside the Core.
* Planning never executes tasks.
* Planning never selects Plugins.
* Planning only creates execution strategy.
* Every request produces exactly one Task Tree.
* Task Trees are deterministic in structure but generated through semantic reasoning.
* Sequential execution begins after Planning has completed.

Once the Task Tree has been generated, it is forwarded to the Capability Registry, where each task will be matched with the most appropriate installed capability.
# Capability Registry

The Task Tree describes **what** must be accomplished.

It does not describe **who** should accomplish it.

That responsibility belongs to the Capability Registry.

The Capability Registry acts as the discovery layer of PRIDE.

Its purpose is to identify which installed Plugin is best suited to execute each task contained inside the Task Tree.

It never performs execution.

It never modifies the Task Tree.

It never changes the workflow.

It only answers one question.

**"Which capability should execute this task?"**

---

```text
Task Tree
    │
    ▼
Capability Registry
    │
    ▼
sdk.ai.match()
    │
    ▼
Provider
    │
    ▼
Selected Plugin
    │
    ▼
Permission Manager
```

---

## Responsibility

The Capability Registry is responsible for:

* Discovering installed Plugins.
* Understanding Plugin capabilities.
* Matching Tasks with Plugins.
* Returning the best available Plugin.

The Capability Registry is **not** responsible for:

* Executing Plugins.
* Installing Plugins.
* Updating Plugins.
* Downloading Plugins.
* Managing Providers.
* Managing Permissions.

Its only responsibility is capability selection.

---

## Input

The Capability Registry receives:

* The current Task from the Task Tree.
* The list of installed Plugins.
* Metadata describing each Plugin.

The Registry never scans source code.

It works only with metadata exposed by every Plugin.

---

## Plugin Metadata

Every Plugin exposes information describing what it can do.

Examples include:

* Plugin Name
* Plugin Description
* Supported Capabilities
* Required Permissions
* Supported Progress
* Required SDKs

The Registry uses this information during capability matching.

---

## Internal Process

Capability selection follows several stages.

### Stage 1 — Discover Installed Plugins

The Registry asks the Plugin System for every installed Plugin.

Only installed Plugins participate.

Remote Plugins are ignored.

Unavailable Plugins are ignored.

Disabled Plugins are ignored.

---

### Stage 2 — Capability Matching

Artificial Intelligence analyzes:

* Current Task
* Plugin Metadata

The Provider determines which Plugin is most appropriate for the current Task.

This is performed through:

```text
sdk.ai.match()
```

Semantic reasoning allows different Plugins with similar descriptions to be selected correctly without relying only on keywords.

---

### Stage 3 — Validation

After a Plugin has been selected, the Registry validates that the Plugin is actually available.

If the Plugin exists,

the workflow continues.

If no suitable Plugin exists,

the workflow immediately returns to the User Interface.

No execution occurs.

---

## Why Artificial Intelligence Is Used

Plugin descriptions are semantic.

Consider two Plugins.

Plugin A

> Creates websites.

Plugin B

> Generates HTML applications.

Both may solve the same Task.

Artificial Intelligence understands capability rather than keywords.

This allows PRIDE to remain extensible without requiring rigid capability naming conventions.

---

## When No Plugin Exists

Sometimes no installed Plugin can complete the requested Task.

Example:

The user asks PRIDE to edit a Blender project.

No Blender Plugin is installed.

Instead of attempting execution,

the Registry returns:

"No compatible Plugin is currently installed."

The UI may then offer the user the opportunity to install an appropriate Plugin.

The workflow stops safely.

PRIDE never invents capabilities that do not exist.

---

## Output

The output of the Capability Registry is:

* The selected Plugin.

or

* A capability-not-found result.

Nothing else.

Execution has not yet started.

The Registry simply identifies who should perform the work.

---

## Design Principles

The Capability Registry follows several architectural principles.

* Plugins remain replaceable.
* Plugin discovery is semantic.
* Registry never executes Plugins.
* Registry never installs Plugins.
* Registry only performs matching.
* Every Task is matched independently.
* Capability selection occurs before Permissions.

Once a suitable Plugin has been identified, the workflow continues to the Permission Manager, where execution safety is validated before any code is allowed to run.
# Permission Manager

Finding the correct Plugin does not automatically mean the Plugin should be allowed to execute.

Before any code is executed, PRIDE performs one final validation.

Permission.

The Permission Manager exists to protect the user rather than the system.

Its responsibility is determining whether the selected Plugin is allowed to perform the requested action.

Unlike Planning and Capability Matching, this stage never uses Artificial Intelligence.

Permission is deterministic.

A Plugin is either allowed to perform an action, or it is not.

There is no reasoning involved.

---

```text
Selected Plugin
        │
        ▼
Permission Manager
        │
        ├───────────────┐
        │               │
 Permission Granted   Permission Denied
        │               │
        ▼               ▼
    Executor            UI
```

---

## Responsibility

The Permission Manager is responsible for:

* Validating Plugin permissions.
* Validating user permissions.
* Preventing unauthorized execution.
* Returning permission results to the Executor or UI.

The Permission Manager is **not** responsible for:

* Selecting Plugins.
* Planning Tasks.
* Executing code.
* Asking Artificial Intelligence for decisions.

Its only responsibility is validating permissions.

---

## Input

The Permission Manager receives:

* The selected Plugin.
* The current Task.
* The Plugin Manifest.
* User permission settings.

Nothing more.

---

## Internal Process

Permission validation occurs in several deterministic stages.

### Stage 1 — Read Plugin Requirements

Every Plugin declares the permissions it requires.

For example:

```yaml
permissions:
  - filesystem.read
  - filesystem.write
  - internet
```

These requirements are declared by the Plugin developer.

They are never guessed by PRIDE.

---

### Stage 2 — Compare User Permissions

The Permission Manager compares the Plugin requirements with the permissions granted by the user.

Examples include:

* File System
* Internet
* Camera
* Microphone
* Clipboard
* Local Applications
* Operating System Commands
* Network Access

If every required permission has already been granted, execution continues immediately.

---

### Stage 3 — Request Permission

If one or more permissions have not been granted,

the workflow pauses.

The User Interface displays a permission request.

Example:

```text
Website Plugin requests:

✓ Internet

✓ File System Write

[ Allow ]

[ Deny ]
```

Only the user can approve this request.

---

### Stage 4 — Produce Decision

After validation, the Permission Manager returns one of two results.

Permission Granted

or

Permission Denied

No intermediate state exists.

---

## Why Artificial Intelligence Is Not Used

Permission is a security decision.

Security decisions must be predictable.

If permission depended on AI reasoning, identical requests could produce different outcomes.

This would reduce transparency and make the platform difficult to trust.

Instead, every permission follows deterministic rules.

The same request always produces the same result.

This behavior reflects one of PRIDE's core architectural principles.

**Deterministic systems remain deterministic.**

---

## Why Permission Exists Before Execution

PRIDE validates permissions before the Executor starts.

This prevents situations where a Plugin begins execution only to discover halfway through that it cannot access the required resources.

Failing early is safer than failing during execution.

---

## Permission Denied

If permission is denied,

execution stops immediately.

The selected Plugin is never started.

No Provider request is made.

No Memory Update occurs.

The workflow returns to the User Interface.

The user remains in complete control.

---

## Output

The Permission Manager produces exactly one result.

Permission Granted

or

Permission Denied.

If permission is granted,

the selected Plugin and current Task continue to the Executor.

If permission is denied,

the workflow returns directly to the User Interface.

---

## Design Principles

The Permission Manager follows several architectural principles.

* Permission is deterministic.
* Permission never uses Artificial Intelligence.
* The user always owns the final decision.
* Validation occurs before execution.
* Plugins never bypass permissions.
* Every execution passes through the Permission Manager.

Once permission has been granted, the workflow continues to the Executor, where the current task begins execution.
# Executor

Once a task has been planned, a Plugin has been selected, and all required permissions have been granted, execution may finally begin.

This responsibility belongs entirely to the Executor.

The Executor is the execution engine of the PRIDE Core.

It does not think.

It does not plan.

It does not reason.

It does not decide which Plugin should be used.

Its responsibility is far simpler.

The Executor coordinates execution.

It receives a validated Task together with the selected Plugin and ensures that the task is completed correctly before continuing to the next task inside the Task Tree.

---

```text
Permission Granted
        │
        ▼
     Executor
        │
        ▼
 Read Current Task
        │
        ▼
Start Plugin
        │
        ▼
Wait For Result
        │
        ▼
Task Complete?
        │
   ┌────┴────┐
   │         │
 Yes         No
   │         │
   ▼         ▼
Next Task   Handle Failure
   │
   ▼
Task Tree Finished?
   │
┌──┴──┐
│     │
No    Yes
│     │
▼     ▼
Next  Memory Update
Task
```

---

## Responsibility

The Executor is responsible for:

* Reading the Task Tree.
* Executing one task at a time.
* Starting the selected Plugin.
* Waiting for Plugin completion.
* Continuing to the next task.
* Reporting execution progress.
* Handling execution failures.
* Handling cancellation requests.

The Executor is **not** responsible for:

* Planning.
* Selecting Plugins.
* Building Context.
* Retrieving Memory.
* Updating Memory.
* Performing Artificial Intelligence reasoning.

The Executor only executes.

---

## Input

The Executor receives:

* Current Task.
* Selected Plugin.
* Permission result.

By the time execution reaches this stage, every previous component has already completed its work.

No further architectural decisions remain.

---

## Internal Process

Execution follows several deterministic stages.

### Stage 1 — Read Current Task

The Executor reads the current task from the Task Tree.

Only one task is active at any given time.

The remaining tasks wait until the current task has completed.

---

### Stage 2 — Start Plugin

The Executor initializes the selected Plugin.

It prepares:

* Task parameters.
* Context.
* Required SDKs.
* Execution environment.

The Plugin is then started.

The Executor never enters the Plugin's internal logic.

It simply starts execution.

---

### Stage 3 — Monitor Execution

While the Plugin is running, the Executor monitors its state.

Possible states include:

* Initializing
* Running
* Waiting
* Completed
* Failed
* Cancelled

These states are forwarded to the User Interface as execution progress.

The Executor never generates progress messages itself.

It only forwards execution state.

---

### Stage 4 — Receive Plugin Result

When the Plugin finishes, it returns one of three possible outcomes.

Successful Result

Execution Failure

Cancellation

The Executor processes each outcome deterministically.

---

### Stage 5 — Continue Task Tree

If execution succeeds, the Executor checks the Task Tree.

If additional tasks remain,

execution continues with the next task.

If no tasks remain,

execution proceeds to Memory Update.

---

## Sequential Execution

PRIDE intentionally executes tasks sequentially.

Example:

```text
Task 1
   │
   ▼
Task 2
   │
   ▼
Task 3
```

Parallel execution is intentionally avoided inside the Core.

This provides several advantages.

* Predictable execution.
* Lower hardware requirements.
* Simpler debugging.
* Stable memory updates.
* Consistent plugin behavior.

Future implementations may optimize internal Plugin execution.

However,

the Core workflow remains sequential.

---

## Progress Reporting

The Executor is the source of workflow progress.

Examples include:

```text
Preparing...

Executing Plugin...

Waiting For Result...

Task Complete...
```

If a Plugin supports detailed progress,

the Executor forwards Plugin progress directly to the User Interface.

Example:

```text
Rendering Image...

Uploading Files...

Generating Website...
```

If the Plugin does not support progress,

the Executor simply reports that execution is currently in progress.

No Artificial Intelligence is required.

---

## Cancellation

Users may cancel execution at any time.

When cancellation occurs:

The User Interface sends a cancellation request to the Executor.

The Executor forwards the cancellation request to the currently running Plugin.

If the Plugin supports interruption,

execution stops immediately.

If interruption is not possible,

the Executor waits until execution safely completes before discarding the result.

Cancellation never skips directly to the next task.

The current task must first reach a safe stopping point.

---

## Failure Handling

Execution failures do not immediately terminate the entire platform.

Instead,

the Executor records the failure and determines the next action according to the workflow.

Possible actions include:

* Returning an error to the UI.
* Stopping the remaining Task Tree.
* Waiting for user interaction.

The Executor never invents recovery strategies.

It follows deterministic execution rules.

---

## Why Artificial Intelligence Is Not Used

Execution is not a reasoning problem.

Every architectural decision has already been made.

Planning has already determined:

* What should happen.

Capability Registry has already determined:

* Which Plugin should execute.

Permission Manager has already determined:

* Whether execution is allowed.

The Executor simply performs those decisions.

Introducing Artificial Intelligence here would increase complexity without improving execution quality.

Therefore,

the Executor remains entirely deterministic.

---

## Output

The Executor produces one of three outcomes.

* Successful Plugin Result.
* Execution Failure.
* Cancellation.

Successful results continue to Memory Update.

Failures and cancellations return control to the User Interface.

---

## Design Principles

The Executor follows several architectural principles.

* Execution is deterministic.
* One task executes at a time.
* Artificial Intelligence is never used.
* The Executor never performs reasoning.
* Plugins remain isolated from workflow orchestration.
* Progress originates from execution state.
* Cancellation always remains under user control.
* The Task Tree determines execution order.

Once a task has completed successfully, the Plugin returns its result to the workflow, where the final processing stage begins inside Memory Update.
# Plugin

Plugins are where actual work is performed.

Every component before the Plugin prepares for execution.

Memory Retrieval gathers knowledge.

The Context Builder organizes understanding.

Planning creates strategy.

The Capability Registry selects the most suitable Plugin.

The Permission Manager validates security.

The Executor starts execution.

Only after all of those stages have completed does the workflow finally enter the Plugin.

Plugins are therefore the implementation layer of PRIDE.

They do not define the architecture.

They implement capabilities.

---

```text id="s0nhxp"
Executor
    │
    ▼
Start Plugin
    │
    ▼
Plugin
    │
    ├───────────────┐
    │ Need AI?      │ No
    ▼               ▼
sdk.ai.reason()    Execute
    │
    ▼
Provider
    │
    ▼
Plugin Result
    │
    ▼
Executor
```

---

## Responsibility

A Plugin is responsible for solving one specific problem.

Examples include:

* Creating websites.
* Editing images.
* Generating documents.
* Searching files.
* Reading PDFs.
* Executing terminal commands.
* Managing databases.
* Automating applications.

A Plugin is **not** responsible for:

* Planning.
* Building Context.
* Retrieving Memory.
* Selecting itself.
* Updating Memory.
* Managing the workflow.

Plugins solve problems.

The Core manages the workflow.

---

## Input

Every Plugin receives a fully prepared execution request.

This request already contains:

* Task information.
* Parameters.
* Context.
* User configuration.
* Permission approval.

The Plugin should never need to determine:

"What should I do?"

That decision has already been made by Planning.

Instead, the Plugin only needs to answer:

"How do I perform this task?"

---

## Internal Process

Although every Plugin is different internally, every Plugin follows the same high-level lifecycle.

### Stage 1 — Initialization

The Plugin initializes its execution environment.

Examples include:

* Loading configuration.
* Preparing resources.
* Opening files.
* Connecting to services.

No reasoning occurs during initialization.

---

### Stage 2 — Determine Execution Type

Every Plugin determines whether Artificial Intelligence is required.

Two execution paths exist.

#### Deterministic Execution

If the task can be completed using deterministic algorithms,

the Plugin executes immediately.

Examples include:

* File operations.
* Compression.
* Encryption.
* Image resizing.
* Database queries.
* Mathematical calculations.

Artificial Intelligence is unnecessary.

---

#### AI-Assisted Execution

Some tasks require reasoning.

Examples include:

* Writing articles.
* Generating source code.
* Summarizing documents.
* Translating languages.
* Image generation.
* Code explanation.

In these situations the Plugin calls:

```text id="juxv4w"
sdk.ai.reason()
```

The SDK forwards the request to the currently selected Provider.

The Provider performs reasoning.

The result is returned to the Plugin.

The Plugin continues execution using that result.

---

### Stage 3 — Complete Task

Once execution has finished,

the Plugin packages the final result.

The Plugin never communicates directly with the User Interface.

Instead,

the result is returned to the Executor.

The workflow then continues normally.

---

## Why Plugins May Use Artificial Intelligence

Artificial Intelligence inside Plugins is optional.

It depends entirely on the capability being implemented.

Examples.

Website Builder

Creating folders:

No AI.

Generating HTML:

AI.

Saving files:

No AI.

Writing documentation:

AI.

The Plugin decides which parts require reasoning and which parts are purely deterministic.

This minimizes unnecessary Provider requests.

---

## Why The Core Never Enters Plugin Logic

The Core intentionally treats every Plugin as a black box.

It knows:

* Inputs.
* Outputs.

It does not know:

* Internal algorithms.
* Internal architecture.
* Internal optimizations.

This separation provides several advantages.

* Plugins remain replaceable.
* Plugins remain independently maintainable.
* Plugin developers have implementation freedom.
* The Core remains stable.

As long as the Plugin respects the Plugin API,

its internal implementation may evolve without affecting the workflow.

---

## Failure Handling

If execution fails,

the Plugin reports structured error information back to the Executor.

The Plugin never decides how errors should be presented.

It only reports what happened.

The Executor and User Interface determine how the failure is communicated to the user.

---

## Output

Every Plugin produces one of three possible results.

Successful Result

Execution Failure

Cancellation

Successful execution returns a Plugin Result to the Executor.

The Executor then forwards the result to the Memory Update stage.

---

## Design Principles

Plugins follow several architectural principles.

* One Plugin solves one domain.
* Plugins never modify the workflow.
* Plugins remain independent from each other.
* Artificial Intelligence is optional.
* Deterministic operations remain deterministic.
* Plugins communicate only through the Plugin API.
* The Core never depends on Plugin internals.
* Plugins may evolve without changing the Core.

Once the Plugin has successfully completed its work, the resulting output enters the final processing stage of the workflow: Memory Update.
# Memory Update

Execution does not end when a Plugin finishes.

PRIDE is designed to continuously improve its understanding of the user over time.

Every completed task becomes an opportunity to learn.

The purpose of Memory Update is to determine whether information produced during execution should become part of the user's long-term knowledge.

Unlike Memory Retrieval, which only reads existing memories, Memory Update writes new knowledge into the user's Memory.

This makes it the final stage of the workflow.

---

```text id="9vw6n2"
Plugin Result
      │
      ▼
Memory Update
      │
      ▼
sdk.ai.organize()
      │
      ▼
Provider
      │
      ▼
Structured Memory
      │
      ▼
sdk.memory.store()
      │
      ▼
Memory
      │
      ▼
UI
      │
      ▼
User
```

---

## Responsibility

Memory Update is responsible for:

* Receiving the final execution result.
* Determining whether the result should become memory.
* Organizing memory into a structured format.
* Generating metadata.
* Storing the memory.
* Returning the final response to the User Interface.

Memory Update is **not** responsible for:

* Planning.
* Executing Plugins.
* Selecting Providers.
* Building Context.
* Retrieving previous memories.

Its responsibility begins only after successful execution has completed.

---

## Input

Memory Update receives:

* Plugin Result.
* Original User Request.
* Execution Metadata.

Examples of execution metadata include:

* Timestamp.
* Plugin identifier.
* Workspace identifier.
* Session identifier.

This metadata improves future retrieval without changing the original content.

---

## Internal Process

Memory Update follows several stages.

### Stage 1 — Determine Memory Eligibility

Not every execution should become memory.

For example:

Opening Calculator

↓

No memory required.

---

Creating an important architecture document

↓

Memory should be stored.

---

Editing a long-term project

↓

Memory should be updated.

This decision follows deterministic rules defined by the Memory system.

The objective is preventing unnecessary memory growth while preserving valuable information.

---

### Stage 2 — Memory Organization

If the result should be stored,

the Plugin Result is sent to:

```text id="u8elv7"
sdk.ai.organize()
```

The selected Provider transforms the raw result into structured knowledge.

The objective is not rewriting the content.

The objective is making future retrieval significantly easier.

---

### Stage 3 — Metadata Generation

Artificial Intelligence generates metadata describing the memory.

Examples include:

* Title
* Summary
* Keywords
* Tags
* Related Topics
* Project Association

Example.

Raw Result

```text id="v4e09g"
Finished designing the Workflow architecture.
```

Generated Metadata

```yaml id="5f70vj"
Title:
Workflow Architecture

Summary:
Final architecture for the PRIDE execution workflow.

Tags:
- Workflow
- Architecture
- Core

Project:
PRIDE
```

This metadata becomes part of the stored memory.

---

### Stage 4 — Store Memory

Once organization has completed,

the structured memory is stored using:

```text id="5ig8eo"
sdk.memory.store()
```

The Memory SDK determines how storage is implemented.

The Core never interacts directly with storage implementations.

This abstraction allows the Memory system to evolve independently.

---

## Why Artificial Intelligence Is Used

Memory organization is a semantic problem.

Suppose two memories exist.

Memory A

> Finished Workflow documentation.

Memory B

> Completed execution architecture.

Humans immediately understand that both are closely related.

A deterministic tagging system may not.

Artificial Intelligence generates consistent metadata that significantly improves future retrieval quality.

The objective is improving search,

not modifying ownership.

---

## User Ownership

Memory Update never changes ownership.

Every stored memory belongs entirely to the user.

PRIDE does not own user memories.

Providers do not own user memories.

Plugins do not own user memories.

The Core only manages how memories are organized and retrieved.

Users remain free to:

* View memories.
* Edit memories.
* Delete memories.
* Export memories.
* Move memories.

Memory organization never removes user control.

---

## Why Memory Update Is The Final Stage

Memory should only be updated after execution has completed successfully.

If Memory Update occurred earlier,

failed executions could permanently store incorrect information.

Placing Memory Update at the end guarantees that only completed execution contributes to long-term knowledge.

---

## Output

Memory Update produces two outputs.

First,

the structured memory is written into the Memory system.

Second,

the final execution result is forwarded to the User Interface.

The User Interface then presents the completed result to the user.

The workflow has now completed.

---

## Design Principles

Memory Update follows several architectural principles.

* Memory is updated only after successful execution.
* Artificial Intelligence organizes memory.
* Storage remains abstract through the Memory SDK.
* User ownership is preserved.
* Metadata improves future retrieval.
* Memory organization never modifies the workflow.
* The Core never depends on storage implementation details.

With Memory Update completed, one complete execution cycle has finished.

The next user request begins the workflow again from the User Interface, allowing PRIDE to continuously improve through accumulated knowledge while preserving a single, stable execution architecture.
# Artificial Intelligence Usage Rules

One of the fundamental principles of PRIDE is that Artificial Intelligence should only be used when semantic reasoning is genuinely required.

Artificial Intelligence is not a replacement for deterministic software.

Instead, it complements deterministic systems by solving problems that cannot be reliably solved through traditional algorithms alone.

This distinction allows PRIDE to remain predictable, efficient, and resource-friendly while still providing adaptive intelligence.

---

## Core Principle

The Core follows a simple rule.

> Artificial Intelligence is used only when semantic reasoning is required.

If deterministic logic can solve a problem,

Artificial Intelligence must not be used.

---

## AI Decision Matrix

The following table defines where Artificial Intelligence is permitted inside the PRIDE workflow.

| Component           |  Uses AI | Reason                                                |
| ------------------- | :------: | ----------------------------------------------------- |
| Memory Retrieval    |     ✅    | Semantic retrieval of relevant memories.              |
| Context Builder     |     ✅    | Constructing meaningful execution context.            |
| Planning            |     ✅    | Transforming objectives into executable Task Trees.   |
| Capability Registry |     ✅    | Matching Tasks with Plugin capabilities semantically. |
| Permission Manager  |     ❌    | Permission validation must remain deterministic.      |
| Executor            |     ❌    | Execution follows an already completed Task Tree.     |
| Plugin              | Optional | Depends on whether the Plugin requires reasoning.     |
| Memory Update       |     ✅    | Organizing memories and generating metadata.          |

---

## Why Some Components Never Use AI

Certain parts of the workflow must remain deterministic.

Examples include:

* Permission validation.
* Task execution.
* Workflow orchestration.

These components always produce identical outputs when given identical inputs.

Predictability is more important than intelligence in these situations.

---

## Plugins Are Independent

Plugins decide independently whether Artificial Intelligence is required.

Examples include.

### Deterministic Plugin

```text id="7l0vqs"
Resize Image

↓

Execute

↓

Done
```

No reasoning is required.

---

### AI Plugin

```text id="q42kpd"
Generate Documentation

↓

sdk.ai.reason()

↓

Provider

↓

Documentation

↓

Done
```

Reasoning is required.

---

## Advantages Of This Design

Restricting Artificial Intelligence to semantic problems provides several benefits.

* Lower execution cost.
* Faster deterministic execution.
* Easier debugging.
* Predictable workflow.
* Reduced Provider requests.
* Better hardware compatibility.
* Clear architectural responsibilities.

Artificial Intelligence becomes a specialized reasoning engine rather than a dependency for every operation.

---

## Architectural Philosophy

PRIDE does not attempt to make every component intelligent.

Instead,

PRIDE makes every component responsible for one specific purpose.

Deterministic components remain deterministic.

Reasoning components remain intelligent.

This separation creates a workflow that is both flexible and maintainable while preserving long-term architectural stability.
# SDK Flow

The SDK is the communication bridge between the PRIDE Core and external systems.

The Core never communicates directly with Providers.

The Core never communicates directly with Memory implementations.

The Core never communicates directly with Plugins.

Instead,

every external interaction occurs through an SDK.

This architectural rule ensures that every component remains replaceable without changing the workflow itself.

---

## Core Principle

The Core never knows implementation details.

It only knows SDK interfaces.

For example,

instead of calling:

```text id="1s3rvy"
Gemini API

Claude API

OpenAI API

Ollama API
```

Planning simply calls:

```text id="q2atpw"
sdk.ai.plan()
```

The SDK determines how that request reaches the currently selected Provider.

---

## SDK Categories

PRIDE currently defines two primary SDK groups.

### AI SDK

Responsible for semantic reasoning.

Examples:

* Planning
* Context construction
* Memory organization
* Capability matching

---

### Memory SDK

Responsible for memory operations.

Examples:

* Retrieval
* Storage
* Editing
* Searching

Future SDKs may be introduced without changing the workflow.

For example:

* sdk.files
* sdk.network
* sdk.database
* sdk.hardware

These additions do not affect the Core architecture.

---

# AI SDK

The AI SDK provides semantic reasoning capabilities.

The Core never communicates directly with a Provider.

Instead,

the Core requests one specific reasoning operation.

The SDK determines how that request is executed.

---

## AI SDK Responsibilities

The AI SDK is responsible for:

* Sending requests to the selected Provider.
* Normalizing Provider responses.
* Handling Provider-specific APIs.
* Returning standardized outputs to the Core.

The AI SDK never performs reasoning itself.

Reasoning belongs to the Provider.

---

## AI SDK Functions

### sdk.ai.retrieve()

Used by:

* Memory Retrieval

Purpose:

Retrieve the most semantically relevant memories.

---

### sdk.ai.context()

Used by:

* Context Builder

Purpose:

Construct the optimal execution context.

---

### sdk.ai.plan()

Used by:

* Planning

Purpose:

Transform context into a Task Tree.

---

### sdk.ai.match()

Used by:

* Capability Registry

Purpose:

Determine which Plugin best matches the current Task.

---

### sdk.ai.reason()

Used by:

* Plugins

Purpose:

Perform Plugin-specific reasoning.

Examples include:

* Writing
* Coding
* Translation
* Image generation

---

### sdk.ai.organize()

Used by:

* Memory Update

Purpose:

Organize execution results into structured memories.

---

## AI SDK Flow

Every AI request follows the same architecture.

```text id="8prszk"
Core

↓

sdk.ai.*

↓

Selected Provider

↓

Reasoning

↓

SDK Response

↓

Core
```

The Core never depends on a specific Provider.

Only the SDK understands Provider implementations.

---

# Memory SDK

The Memory SDK manages every interaction with user memory.

Like the AI SDK,

the Core never communicates directly with storage systems.

Instead,

every memory operation passes through the Memory SDK.

---

## Memory SDK Responsibilities

The Memory SDK is responsible for:

* Reading memories.
* Writing memories.
* Editing memories.
* Searching memories.
* Abstracting storage implementations.

The Memory SDK never performs semantic reasoning.

Reasoning belongs to the AI SDK.

---

## Memory SDK Functions

### sdk.memory.retrieve()

Used by:

* Memory Retrieval

Purpose:

Retrieve candidate memories.

---

### sdk.memory.store()

Used by:

* Memory Update

Purpose:

Persist structured memories.

---

### Future Memory Functions

Future implementations may include:

```text id="wb3kgv"
sdk.memory.update()

sdk.memory.delete()

sdk.memory.search()

sdk.memory.export()

sdk.memory.import()
```

These functions expand Memory capabilities without affecting the workflow.

---

# Why SDK Exists

Without an SDK,

every Core component would require direct knowledge of:

* OpenAI
* Gemini
* Claude
* Ollama
* Future Providers

Changing Providers would require modifying multiple Core components.

The SDK removes this dependency.

The Core communicates only with SDK interfaces.

The SDK handles everything else.

---

# Provider Independence

Consider the following example.

Planning requests:

```text id="9quewz"
sdk.ai.plan()
```

Today,

the SDK forwards the request to Gemini.

Tomorrow,

the user switches to Claude.

Planning remains identical.

The workflow remains identical.

Only the SDK changes its routing.

This design makes PRIDE fully Provider-agnostic.

---

# Design Principles

The SDK follows several architectural principles.

* The Core never communicates directly with Providers.
* The Core never communicates directly with Memory implementations.
* SDKs abstract implementation details.
* SDK interfaces remain stable.
* Providers remain replaceable.
* Storage systems remain replaceable.
* New SDKs may be introduced without changing the workflow.
* The SDK performs communication, not reasoning.

The SDK is therefore one of the most important abstraction layers inside PRIDE, allowing the architecture to evolve for decades without requiring changes to the Core workflow.
# Workflow Rules

The PRIDE workflow follows a collection of architectural rules.

These rules define how every component must behave.

Unlike implementation details,

these rules are considered part of the Core architecture.

Every future implementation must respect these rules to ensure long-term compatibility.

---

# Rule 1 — One Direction Only

The workflow always moves forward.

A component may pass data to the next stage.

It must never jump backward.

Example.

```text id="e9mq4z"
Planning

↓

Capability Registry

↓

Permission Manager

↓

Executor
```

Not.

```text id="zb6z6l"
Executor

↓

Planning
```

Backward execution is prohibited.

---

# Rule 2 — One Responsibility Per Component

Every component owns exactly one responsibility.

Examples.

Memory Retrieval

Only retrieves memory.

Planning

Only creates Task Trees.

Capability Registry

Only selects Plugins.

Executor

Only executes.

No component should perform another component's responsibility.

---

# Rule 3 — Artificial Intelligence Is Used Only For Semantic Reasoning

Artificial Intelligence exists only where semantic understanding is required.

Examples.

Allowed:

* Memory Retrieval
* Context Builder
* Planning
* Capability Registry
* Plugin (optional)
* Memory Update

Not Allowed:

* Permission Manager
* Executor

Reasoning should never replace deterministic execution.

---

# Rule 4 — Deterministic Components Must Remain Deterministic

Components responsible for validation or execution must always produce identical results for identical inputs.

Examples include:

* Permission validation.
* Execution.
* Workflow transitions.

Predictability is prioritized over intelligence.

---

# Rule 5 — Every Task Must Pass Through The Entire Workflow

No component may be skipped.

Every execution follows the same architecture.

```text id="zzyvnt"
User

↓

UI

↓

Memory Retrieval

↓

Context Builder

↓

Planning

↓

Capability Registry

↓

Permission Manager

↓

Executor

↓

Plugin

↓

Memory Update

↓

UI
```

The architecture never changes based on Plugin type.

Only internal Plugin implementation may differ.

---

# Rule 6 — Plugins Never Control The Workflow

Plugins only solve their assigned tasks.

Plugins never:

* Jump to another component.
* Modify the workflow.
* Select another Plugin.
* Update Memory directly.
* Perform Planning.

The Core always owns workflow control.

---

# Rule 7 — Components Communicate Only Through Defined Interfaces

Every component communicates only with its neighboring component.

Examples.

Planning never communicates directly with Plugins.

Plugins never communicate directly with Memory.

Memory never communicates directly with Providers.

All communication follows the defined workflow.

---

# Rule 8 — Provider Independence

The workflow must never depend on a specific AI Provider.

Changing Providers must never require changes to the workflow.

Only SDK implementations may change.

The workflow remains identical.

---

# Rule 9 — User Always Owns The Final Decision

Certain decisions always belong to the user.

Examples include:

* Granting permissions.
* Selecting the AI Provider.
* Cancelling execution.
* Editing memories.

The Core may recommend actions.

It never overrides explicit user decisions.

---

# Rule 10 — Memory Is Updated Only After Successful Execution

Memory is never updated before execution completes.

Failed executions should not permanently affect long-term knowledge.

Only completed and validated execution results may enter Memory.

---

# Rule 11 — Sequential Execution

The Core executes one Task at a time.

A new Task begins only after the previous Task has completed.

Parallel execution may exist inside individual Plugins,

but never inside the Core workflow.

This guarantees predictable behavior and simplifies debugging.

---

# Rule 12 — The Core Never Depends On Implementations

The Core depends only on contracts.

It never depends on:

* Plugin implementations.
* Provider implementations.
* Memory implementations.

Every replaceable subsystem must expose a stable interface.

This allows implementations to evolve without changing the workflow itself.

---

## Summary

These rules define the behavior of every Core component.

Together they ensure that PRIDE remains:

* Stable.
* Predictable.
* Provider-agnostic.
* Plugin-independent.
* Maintainable.
* Extensible.

Future implementations may introduce new capabilities,

but they must never violate these workflow rules.
# Error Handling

No software system is immune to failure.

PRIDE assumes that failures will occur.

Instead of attempting to eliminate every possible error, the Core is designed to detect failures early, isolate them, and terminate execution safely.

Error handling is therefore considered part of the Core architecture rather than an implementation detail.

---

## Core Principle

Every component is responsible for reporting failures.

No component is responsible for hiding failures.

The objective is:

* Detect early.
* Report clearly.
* Stop safely.
* Preserve system integrity.

---

# Failure Categories

The workflow recognizes several categories of failure.

Each category is handled by a different component.

---

## User Error

Examples include:

* Invalid input.
* Missing required information.
* Unsupported request format.

Handling:

The workflow immediately returns to the User Interface.

No execution continues.

---

## Memory Failure

Examples include:

* Memory unavailable.
* Corrupted memory index.
* Retrieval timeout.

Handling:

The workflow continues without Memory when possible.

Memory Retrieval should never crash the Core.

---

## Planning Failure

Examples include:

* Unable to generate a valid Task Tree.
* Invalid planning result.
* Missing required task information.

Handling:

Execution stops.

The User Interface displays the planning error.

No Plugin execution begins.

---

## Capability Failure

Examples include:

* No compatible Plugin installed.
* Plugin metadata invalid.
* No matching capability found.

Handling:

Execution stops.

The User Interface informs the user that no compatible Plugin is available.

The user may install an appropriate Plugin before trying again.

---

## Permission Failure

Examples include:

* User denies permission.
* Required permission unavailable.
* Restricted environment.

Handling:

Execution stops immediately.

The selected Plugin is never started.

---

## Execution Failure

Examples include:

* Plugin crash.
* Unexpected exception.
* Resource unavailable.
* Internal Plugin error.

Handling:

The Executor terminates the current task safely.

Remaining tasks are not executed.

The error is forwarded to the User Interface.

---

## Provider Failure

Examples include:

* Provider offline.
* API timeout.
* Invalid Provider response.
* Authentication failure.

Handling depends on the current component.

If reasoning cannot continue,

execution stops gracefully.

The workflow never attempts to fabricate Provider responses.

---

## Memory Update Failure

Examples include:

* Storage unavailable.
* Write failure.
* Corrupted memory record.

Handling:

The execution result is still returned to the user.

Memory Update failure must never destroy a successfully completed task.

The user receives the result even if Memory cannot be updated.

---

# Error Propagation

Errors always move upward through the workflow.

Example.

```text id="n2xq7d"
Plugin

↓

Executor

↓

UI

↓

User
```

Errors never move sideways.

Errors never skip components.

---

# Fail Fast Principle

PRIDE follows the Fail Fast principle.

Once a critical failure is detected,

execution stops immediately.

The workflow never attempts to continue using invalid data.

This prevents cascading failures that are more difficult to diagnose.

---

# Error Transparency

Errors should remain understandable.

The Core reports technical failures.

The User Interface translates technical failures into user-friendly messages.

Example.

Internal error.

```text id="l9w6ra"
PluginNotFoundException
```

Displayed message.

```text id="pxw3gm"
The required Plugin is not currently installed.
```

This separation keeps the Core implementation independent from presentation.

---

# Recovery Responsibility

Recovery is not performed by the component that detects the error.

Examples.

Planning detects planning failure.

↓

UI explains the problem.

Executor detects execution failure.

↓

UI displays the failure.

Memory detects storage failure.

↓

Execution result is still shown.

Each component reports.

The workflow decides what happens next.

---

# Design Principles

Error handling follows several architectural principles.

* Failures are expected.
* Failures are isolated.
* Components report failures.
* Components never hide failures.
* Critical failures stop execution immediately.
* Successful execution is never discarded because Memory Update fails.
* Error presentation belongs to the User Interface.
* Workflow integrity is always prioritized over partial execution.

A predictable failure is preferable to an unpredictable success.

By enforcing these principles, PRIDE remains stable even when individual components encounter unexpected conditions.
# Cancellation Flow

Execution may be cancelled at any time by the user.

Cancellation is considered a normal workflow event rather than an error.

Unlike failures, cancellation represents an intentional decision made by the user.

The Core must therefore stop execution safely while preserving system integrity.

---

## Core Principle

The user always has the right to stop execution.

The Core must honor cancellation requests as soon as it is safe to do so.

Cancellation should never leave the system in an inconsistent state.

---

## Cancellation Flow

```text id="5xp1wh"
User
    │
    ▼
UI
    │
    ▼
Executor
    │
    ▼
Current Plugin
    │
    ├───────────────┐
    │               │
Can Stop?         Cannot Stop
    │               │
    ▼               ▼
Stop Plugin     Finish Current Operation
    │               │
    └───────┬───────┘
            │
            ▼
Release Resources
            │
            ▼
Return Cancellation Result
            │
            ▼
UI
            │
            ▼
User
```

---

## Responsibility

Cancellation is coordinated by the Executor.

The User Interface only forwards the user's request.

Plugins only respond to cancellation requests.

The Executor determines when execution can safely terminate.

---

## Cancellation Lifecycle

### Stage 1 — User Requests Cancellation

The cancellation request always begins from the User Interface.

Examples include:

* Pressing the Stop button.
* Closing the execution dialog.
* Cancelling an active task.

The request is immediately forwarded to the Executor.

---

### Stage 2 — Executor Receives Request

The Executor marks the current execution as:

```text id="xv6m4s"
Cancellation Requested
```

No additional Tasks from the Task Tree are started.

Only the currently running Task remains active.

---

### Stage 3 — Notify Plugin

The Executor forwards the cancellation request to the active Plugin.

Every Plugin must support graceful cancellation whenever technically possible.

Examples include:

* Stop generation.
* Abort download.
* Cancel rendering.
* Terminate background execution.

---

### Stage 4 — Safe Termination

If the Plugin can safely stop,

execution terminates immediately.

If immediate interruption is unsafe,

the Plugin finishes the current atomic operation before stopping.

The Core always prioritizes consistency over speed.

---

### Stage 5 — Resource Cleanup

Before leaving execution,

the Plugin releases any temporary resources.

Examples include:

* Temporary files.
* Network connections.
* Database sessions.
* File handles.
* Background workers.

Resource cleanup prevents leaks and ensures future executions remain stable.

---

### Stage 6 — Return Result

The Executor reports that execution was cancelled.

This is not treated as an execution failure.

Instead,

a dedicated cancellation result is returned to the User Interface.

Example.

```text id="0xgxni"
Execution Cancelled
```

---

## Memory Update

Cancelled executions do not update Memory.

Because execution never completed,

there is no verified result that should become long-term knowledge.

The workflow therefore ends immediately after cancellation.

---

## Task Tree Behavior

Cancellation affects the entire remaining Task Tree.

Example.

```text id="mbjqzy"
Task 1

✓ Complete

↓

Task 2

Running

↓

Cancel

↓

Task 3

Not Executed

↓

Task 4

Not Executed
```

Only completed Tasks remain valid.

Remaining Tasks are discarded.

---

## Design Principles

Cancellation follows several architectural principles.

* Cancellation is not an error.
* The user may cancel at any time.
* The Executor coordinates cancellation.
* Plugins terminate safely.
* Resource cleanup always occurs.
* Memory is never updated after cancellation.
* Remaining Tasks are discarded.
* Workflow consistency is prioritized over immediate interruption.

By treating cancellation as a first-class workflow event, PRIDE ensures that users always remain in control while preserving system stability.
# Progress Flow

Execution progress allows users to understand what the system is currently doing.

Unlike execution itself,

Progress Flow is purely informational.

Its purpose is improving transparency rather than affecting the workflow.

Progress never changes execution behavior.

It only reflects the current execution state.

---

## Core Principle

Progress is a visualization of execution.

It is never a source of execution.

The User Interface displays progress.

The Executor produces progress.

Plugins may optionally provide detailed progress information.

---

## Progress Flow

```text id="6sj2dx"
Executor
    │
    ▼
Execution State
    │
    ▼
Progress Event
    │
    ▼
UI
    │
    ▼
User
```

If a Plugin supports detailed progress,

the flow becomes:

```text id="rmtn5y"
Plugin
    │
    ▼
Executor
    │
    ▼
Progress Event
    │
    ▼
UI
    │
    ▼
User
```

The User Interface never guesses progress.

It only displays information received from the Executor.

---

## Responsibility

The Executor is responsible for:

* Producing execution state.
* Forwarding Plugin progress.
* Determining when progress changes.

Plugins are responsible for:

* Reporting detailed progress if supported.

The User Interface is responsible for:

* Displaying progress.
* Updating the visual indicator.
* Informing the user.

---

## Execution States

Every execution moves through a predictable sequence of states.

```text id="ytk4uo"
Preparing

↓

Executing

↓

Waiting

↓

Completed
```

Additional states may include:

```text id="4vpru4"
Cancelled

Failed
```

These states are standardized across the entire Core.

---

## Plugin Progress

Some Plugins are capable of reporting more detailed progress.

Examples include:

Website Builder

```text id="b1n0yw"
Creating Project...

Generating Components...

Installing Dependencies...

Writing Files...

Completed
```

Image Generator

```text id="4tsqk5"
Preparing Prompt...

Generating Image...

Finalizing Result...
```

Document Generator

```text id="bjw5kt"
Analyzing Content...

Writing Document...

Formatting...

Completed
```

The Core forwards these progress updates without modification.

---

## Plugins Without Progress

Not every Plugin can report progress.

Example.

```text id="uyc9u2"
Resize Image
```

This operation may finish almost instantly.

In these situations,

the Executor simply reports:

```text id="jlwmrq"
Executing...
```

until execution completes.

No additional progress information is required.

---

## Progress Frequency

Progress should only be updated when the execution state changes.

Repeatedly sending identical progress information wastes resources and creates unnecessary User Interface updates.

Example.

Correct:

```text id="y4jjs8"
Preparing...

Executing...

Completed
```

Incorrect:

```text id="x73gyn"
Executing...

Executing...

Executing...

Executing...
```

without any actual state change.

---

## Progress Independence

Progress never affects execution.

If progress reporting fails,

execution continues normally.

Examples.

User Interface fails to display progress.

↓

Execution continues.

Plugin cannot report progress.

↓

Execution continues.

Progress is optional information,

not part of execution logic.

---

## Artificial Intelligence

Progress Flow never uses Artificial Intelligence.

Every progress update originates from deterministic execution states.

No reasoning is required.

---

## Design Principles

Progress Flow follows several architectural principles.

* Progress reflects execution.
* Progress never controls execution.
* The Executor owns execution state.
* Plugins may optionally provide detailed progress.
* The User Interface only displays progress.
* Artificial Intelligence is never required.
* Progress failures never interrupt execution.

Progress exists solely to improve transparency while preserving a fully deterministic execution workflow.
# Workflow Design Principles

The PRIDE workflow is not merely an execution sequence.

It is the architectural foundation that governs how every Core component collaborates.

These principles are intentionally technology-independent.

They define how the Core should behave regardless of future implementations, Providers, Plugins, or programming languages.

Every future modification to the Core should preserve these principles.

---

# Principle 1 — One Stable Workflow

PRIDE defines only one execution workflow.

Every user request,

regardless of complexity,

travels through the same architectural pipeline.

Examples include:

* Creating websites.
* Generating code.
* Reading PDFs.
* Editing memories.
* Building applications.

The implementation may differ.

The workflow never does.

---

# Principle 2 — Sequential Execution

The Core executes one Task at a time.

Execution never branches inside the Core.

This provides:

* Predictability.
* Simpler debugging.
* Lower hardware requirements.
* Stable execution order.

Parallelism may exist inside individual Plugins,

but never inside the Core workflow.

---

# Principle 3 — Single Responsibility

Every Core component owns exactly one responsibility.

Examples.

Memory Retrieval retrieves memories.

Planning creates Task Trees.

Executor executes Tasks.

Memory Update stores knowledge.

No component performs another component's responsibility.

This separation minimizes coupling between subsystems.

---

# Principle 4 — Artificial Intelligence Only Where Necessary

Artificial Intelligence is reserved exclusively for semantic reasoning.

If deterministic logic can solve a problem,

deterministic logic must always be preferred.

Reasoning is used only when understanding,

interpretation,

or semantic matching is required.

---

# Principle 5 — Deterministic Core

The Core itself remains deterministic.

Workflow transitions,

permission validation,

execution,

and orchestration always produce identical outputs given identical inputs.

This guarantees reproducible behavior.

---

# Principle 6 — Provider Independence

The Core never depends on any specific Artificial Intelligence Provider.

Providers are implementation details.

Replacing a Provider must never require modifications to the workflow.

Only SDK implementations may change.

---

# Principle 7 — Plugin Independence

Plugins are independent capabilities.

The Core never depends on Plugin internals.

Plugins may evolve,

be replaced,

or be removed,

without affecting the workflow.

As long as Plugins respect the Plugin API,

the Core remains unchanged.

---

# Principle 8 — User Ownership

Users always remain in control.

The Core never overrides explicit user decisions.

Examples include:

* Provider selection.
* Permission approval.
* Execution cancellation.
* Memory editing.

Artificial Intelligence assists users.

It never replaces user authority.

---

# Principle 9 — Stable Abstraction Layers

The Core communicates only through stable interfaces.

Examples include:

* SDK interfaces.
* Plugin API.
* Memory API.

Implementation details remain hidden behind abstraction layers.

This allows internal evolution without architectural changes.

---

# Principle 10 — Forward-Only Workflow

Execution always progresses forward.

The workflow never jumps backward.

Every completed stage passes responsibility to the next stage.

This guarantees a predictable execution lifecycle.

---

# Principle 11 — Transparent Execution

Users should always understand what the system is currently doing.

Execution states,

progress,

permissions,

and failures should remain observable through the User Interface.

Transparency builds trust.

---

# Principle 12 — Fail Safely

Failures are expected.

Unexpected behavior is not.

Whenever execution cannot continue safely,

the workflow terminates gracefully.

Protecting system integrity is always more important than forcing execution to continue.

---

# Principle 13 — Architecture Before Implementation

Workflow defines architecture.

Code implements architecture.

Implementation details may change throughout the lifetime of PRIDE.

The workflow should remain stable.

Future developers should be able to replace entire implementations while preserving the same architectural behavior.

---

# Principle 14 — Long-Term Maintainability

Every architectural decision should reduce future maintenance costs.

Complexity should remain localized.

Changes to one subsystem should have minimal impact on others.

The objective is ensuring that PRIDE remains maintainable for decades rather than months.

---

# Summary

These principles collectively define the identity of the PRIDE Core.

Every implementation,

optimization,

or future extension should be evaluated against these principles before becoming part of the architecture.

Technology will evolve.

Providers will evolve.

Plugins will evolve.

Programming languages will evolve.

The workflow should not.

A stable architecture enables sustainable evolution.
# Component Mapping

This section provides a complete architectural reference for every Core component.

Unlike previous sections,

this chapter does not explain implementation details.

Instead,

it summarizes the role of each component inside the workflow.

This table serves as a quick reference for developers during implementation.

---

| Component           | Primary Responsibility                         | Input                  | Output                           |  Uses AI | SDK                                           | Next Component      |
| ------------------- | ---------------------------------------------- | ---------------------- | -------------------------------- | :------: | --------------------------------------------- | ------------------- |
| User                | Initiates execution                            | Request                | User Input                       |     ❌    | None                                          | UI                  |
| UI                  | Receives user interaction and displays results | User Input             | Standardized Request             |     ❌    | None                                          | Memory Retrieval    |
| Memory Retrieval    | Retrieves relevant long-term knowledge         | Request                | Relevant Memories                |     ✅    | `sdk.memory.retrieve()` + `sdk.ai.retrieve()` | Context Builder     |
| Context Builder     | Constructs execution context                   | Request + Memories     | Execution Context                |     ✅    | `sdk.ai.context()`                            | Planning            |
| Planning            | Generates a complete Task Tree                 | Execution Context      | Task Tree                        |     ✅    | `sdk.ai.plan()`                               | Capability Registry |
| Capability Registry | Selects the most suitable Plugin               | Task Tree              | Selected Plugin                  |     ✅    | `sdk.ai.match()`                              | Permission Manager  |
| Permission Manager  | Validates execution permissions                | Selected Plugin        | Permission Decision              |     ❌    | None                                          | Executor            |
| Executor            | Executes the approved Task Tree                | Task Tree + Permission | Plugin Execution                 |     ❌    | None                                          | Plugin              |
| Plugin              | Solves the requested task                      | Execution Request      | Plugin Result                    | Optional | `sdk.ai.reason()` *(only when required)*      | Memory Update       |
| Memory Update       | Organizes and stores new knowledge             | Plugin Result          | Structured Memory + Final Result |     ✅    | `sdk.ai.organize()` + `sdk.memory.store()`    | UI                  |
| UI                  | Presents the final response                    | Execution Result       | Response                         |     ❌    | None                                          | User                |

---

# Artificial Intelligence Mapping

Only semantic components use Artificial Intelligence.

| Component           |  Uses AI | Purpose                   |
| ------------------- | :------: | ------------------------- |
| Memory Retrieval    |     ✅    | Semantic memory retrieval |
| Context Builder     |     ✅    | Context construction      |
| Planning            |     ✅    | Task planning             |
| Capability Registry |     ✅    | Semantic Plugin matching  |
| Permission Manager  |     ❌    | Deterministic validation  |
| Executor            |     ❌    | Deterministic execution   |
| Plugin              | Optional | Plugin-specific reasoning |
| Memory Update       |     ✅    | Memory organization       |

---

# SDK Mapping

Every SDK invocation inside the workflow.

| Component           | SDK Function            |
| ------------------- | ----------------------- |
| Memory Retrieval    | `sdk.memory.retrieve()` |
| Memory Retrieval    | `sdk.ai.retrieve()`     |
| Context Builder     | `sdk.ai.context()`      |
| Planning            | `sdk.ai.plan()`         |
| Capability Registry | `sdk.ai.match()`        |
| Plugin *(optional)* | `sdk.ai.reason()`       |
| Memory Update       | `sdk.ai.organize()`     |
| Memory Update       | `sdk.memory.store()`    |

---

# Workflow Order

The execution order of the Core never changes.

```text id="3t4zbe"
1. User

2. UI

3. Memory Retrieval

4. Context Builder

5. Planning

6. Capability Registry

7. Permission Manager

8. Executor

9. Plugin

10. Memory Update

11. UI

12. User
```

Every execution follows this exact sequence.

No component may be skipped.

No component may execute out of order.

This ordering is considered part of the PRIDE Core architecture.
# Component Mapping

This section provides a complete architectural reference for every Core component.

Unlike previous sections,

this chapter does not explain implementation details.

Instead,

it summarizes the role of each component inside the workflow.

This table serves as a quick reference for developers during implementation.

---

| Component           | Primary Responsibility                         | Input                  | Output                           |  Uses AI | SDK                                           | Next Component      |
| ------------------- | ---------------------------------------------- | ---------------------- | -------------------------------- | :------: | --------------------------------------------- | ------------------- |
| User                | Initiates execution                            | Request                | User Input                       |     ❌    | None                                          | UI                  |
| UI                  | Receives user interaction and displays results | User Input             | Standardized Request             |     ❌    | None                                          | Memory Retrieval    |
| Memory Retrieval    | Retrieves relevant long-term knowledge         | Request                | Relevant Memories                |     ✅    | `sdk.memory.retrieve()` + `sdk.ai.retrieve()` | Context Builder     |
| Context Builder     | Constructs execution context                   | Request + Memories     | Execution Context                |     ✅    | `sdk.ai.context()`                            | Planning            |
| Planning            | Generates a complete Task Tree                 | Execution Context      | Task Tree                        |     ✅    | `sdk.ai.plan()`                               | Capability Registry |
| Capability Registry | Selects the most suitable Plugin               | Task Tree              | Selected Plugin                  |     ✅    | `sdk.ai.match()`                              | Permission Manager  |
| Permission Manager  | Validates execution permissions                | Selected Plugin        | Permission Decision              |     ❌    | None                                          | Executor            |
| Executor            | Executes the approved Task Tree                | Task Tree + Permission | Plugin Execution                 |     ❌    | None                                          | Plugin              |
| Plugin              | Solves the requested task                      | Execution Request      | Plugin Result                    | Optional | `sdk.ai.reason()` *(only when required)*      | Memory Update       |
| Memory Update       | Organizes and stores new knowledge             | Plugin Result          | Structured Memory + Final Result |     ✅    | `sdk.ai.organize()` + `sdk.memory.store()`    | UI                  |
| UI                  | Presents the final response                    | Execution Result       | Response                         |     ❌    | None                                          | User                |

---

# Artificial Intelligence Mapping

Only semantic components use Artificial Intelligence.

| Component           |  Uses AI | Purpose                   |
| ------------------- | :------: | ------------------------- |
| Memory Retrieval    |     ✅    | Semantic memory retrieval |
| Context Builder     |     ✅    | Context construction      |
| Planning            |     ✅    | Task planning             |
| Capability Registry |     ✅    | Semantic Plugin matching  |
| Permission Manager  |     ❌    | Deterministic validation  |
| Executor            |     ❌    | Deterministic execution   |
| Plugin              | Optional   | Plugin-specific reasoning |
| Memory Update       |     ✅    | Memory organization       |

---

# SDK Mapping

Every SDK invocation inside the workflow.

| Component           | SDK Function            |
| ------------------- | ----------------------- |
| Memory Retrieval    | `sdk.memory.retrieve()` |
| Memory Retrieval    | `sdk.ai.retrieve()`     |
| Context Builder     | `sdk.ai.context()`      |
| Planning            | `sdk.ai.plan()`         |
| Capability Registry | `sdk.ai.match()`        |
| Plugin *(optional)* | `sdk.ai.reason()`       |
| Memory Update       | `sdk.ai.organize()`     |
| Memory Update       | `sdk.memory.store()`    |

---

# Workflow Order

The execution order of the Core never changes.

```text id="3t4zbe"
1. User

2. UI

3. Memory Retrieval

4. Context Builder

5. Planning

6. Capability Registry

7. Permission Manager

8. Executor

9. Plugin

10. Memory Update

11. UI

12. User
```

Every execution follows this exact sequence.

No component may be skipped.

No component may execute out of order.

This ordering is considered part of the PRIDE Core architecture.
# Complete Workflow Summary

The PRIDE Core is built upon a single, stable execution workflow.

Every request, regardless of complexity, follows exactly the same architectural lifecycle.

The implementation of individual components may evolve.

The workflow itself should remain stable.

This stability allows every subsystem inside PRIDE to evolve independently while preserving long-term maintainability.

---

# Complete Workflow

```text id="4d3h8f"
User
   │
   ▼
UI
   │
   ▼
Memory Retrieval
   │
   ▼
Context Builder
   │
   ▼
Planning
   │
   ▼
Capability Registry
   │
   ▼
Permission Manager
   │
   ▼
Executor
   │
   ▼
Plugin
   │
   ▼
Memory Update
   │
   ▼
UI
   │
   ▼
User
```

Every request follows this exact sequence.

No component may be skipped.

No component may execute outside its assigned responsibility.

---

# Execution Lifecycle

A complete execution consists of four major phases.

## Phase 1 — Understanding

The Core understands the user's request.

Components involved:

* Memory Retrieval
* Context Builder

Objective:

Construct the most accurate execution context possible.

---

## Phase 2 — Decision Making

The Core determines how the request should be executed.

Components involved:

* Planning
* Capability Registry
* Permission Manager

Objective:

Generate a complete Task Tree, select the appropriate Plugin, and validate execution permissions.

---

## Phase 3 — Execution

The requested work is performed.

Components involved:

* Executor
* Plugin

Objective:

Execute the approved Task safely and deterministically.

---

## Phase 4 — Learning

The completed execution becomes future knowledge.

Components involved:

* Memory Update

Objective:

Organize useful execution results into structured long-term memory before presenting the final response.

---

# Architectural Characteristics

The PRIDE workflow is intentionally designed around several permanent characteristics.

* One workflow.
* Forward-only execution.
* Single responsibility.
* Deterministic Core.
* Artificial Intelligence only where semantic reasoning is required.
* Provider-independent architecture.
* Plugin-independent architecture.
* User-controlled execution.
* Stable abstraction layers.
* Long-term maintainability.

These characteristics define the identity of the PRIDE Core.

---

# Philosophy

PRIDE is built upon a simple architectural belief.

Technology should adapt to humans.

Humans should never be forced to adapt to technology.

Every component inside the Core exists to support that objective.

Memory exists to remember.

Planning exists to understand.

Plugins exist to perform work.

The Core exists to orchestrate every subsystem into one predictable execution lifecycle.

---

# Final Statement

This document defines the complete execution architecture of the PRIDE Core.

It serves as the primary architectural reference for every future implementation.

Programming languages may change.

Providers may change.

Plugins may change.

Memory implementations may change.

Internal algorithms may change.

The workflow should not.

A stable workflow creates a stable foundation.

A stable foundation enables unlimited evolution.

This concludes the Workflow documentation for the PRIDE Core.

