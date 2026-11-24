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
