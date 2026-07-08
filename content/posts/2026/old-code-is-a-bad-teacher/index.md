---
title: "Old Code Is a Bad Teacher"
date: "2026-08-11T10:00:00Z"
categories: ["article"]
tags: ["software-engineering", "architecture", "technical-debt", "productivity"]
description: "Large codebases teach developers by example. When obsolete patterns remain searchable and copyable, technical debt spreads unless you clearly mark what not to imitate."
slug: "old-code-is-a-bad-teacher"
cover: "cover.svg"
images: ["/posts/old-code-is-a-bad-teacher/cover.svg"]
type: "posts"
devtoSkip: true
---

A large codebase does more than run software. It teaches people how to write the next piece of it.

Usually, that is a good thing. Developers do not start every task from a blank page. They search for similar code, copy the shape of it, and adapt it to the problem in front of them. This is not laziness. It is how people learn a system.

Need to add a span to a trace? Find another handler that does it. Need to register a new service? Copy the structure from one that already works. In a healthy codebase, this is a superpower. Over time, preferred ways to do common tasks become the "paved path." They make onboarding faster, make services easier to recognize, and reduce the number of decisions every engineer has to make before they can ship useful work.

But there is a catch: the codebase keeps teaching people even after the lesson is out of date.

## Standards Expire

The "right way" to build something today will eventually become the old way. You migrate to a new tracing standard. You replace an ORM. You change how services talk to each other. You introduce a better configuration system. You decide that new services should use a different framework, transport, deployment model, or observability stack. That is normal. A codebase that never changes is usually either dead or terrifying.

The obvious answer is to migrate everything, but in a large codebase, full migration is often not worth it. Some old services are stable, boring, and barely touched. Rewriting them just so they resemble newer services is often a bad trade. Leaving old code alone can be the pragmatic choice.

The problem is not that the old code exists. The problem is that old code often looks normal.

## Developers will Copy-Paste

Developers learn by example, especially new developers. When someone joins a team, they search the codebase for similar work and find a service that does something close to what they need. If that service uses an outdated database layer, an old tracing package, or a retired service pattern, they may copy it without realizing they are copying a historical artifact.

From their perspective, the code works. It passed review at some point. It lives in the same repository as everything else. So why would they assume it is the wrong example?

This is how technical debt spreads: not through a big architectural decision or a dramatic rewrite gone wrong, but through small, reasonable acts of imitation. Someone copies a deprecated client, uses an old test helper, or scaffolds a new service from a legacy template because it has similar business logic. Now the old pattern has reproduced.

If this keeps happening, your new standard never really becomes the standard. It becomes one more option in a growing pile. That is how large codebases end up with five ways to talk to the database, three logging libraries, and a Slack thread every few months where someone asks, “Wait, which one are we supposed to use now?”

## Make Legacy Code Noisy

If you decide to leave old patterns in place, you need to make their status obvious. You cannot rely on memory, tribal knowledge, or the one senior engineer who knows which package is cursed. The warning needs to appear where developers actually work: in the editor, in code search, and in CI.

### Deprecate Loudly

Use your language’s deprecation tools. If you have `@Deprecated`, `[Obsolete]`, `// Deprecated:`, or an equivalent mechanism, use it aggressively. Modern IDEs will strike through deprecated symbols or surface warnings inline, creating useful friction before the code gets copied.

A deprecation message should also say what to use instead:

Bad:

```go
// Deprecated.
```

Better:

```go
// Deprecated: use postgresv2.Client instead. New services should not import this package.
```

The goal is to redirect people before they copy the code into a new feature.

### Name Honestly

Names carry permission. A package named `events` sounds safe. A package named `legacyevents` sounds suspicious. A file named `UserEventPublisher.ts` looks like a reasonable thing to copy. A file named `UserEventPublisher_LegacyDoNotCopy.ts` makes people pause.

If a module exists only to keep an old system alive, do not give it a clean, timeless name. Put `legacy` or `do_not_copy` directly in the file path or package name. Ugly names are harder to accidentally cargo-cult.

### Block New Imports

Warnings can be ignored, but compiler or linter errors cannot. If an old pattern should not spread, enforce the boundary with tooling. Use static analysis, lint rules, or build constraints to prevent new files from importing legacy packages.

For instance, TypeScript teams can configure ESLint’s `no-restricted-imports` rule to block imports from a legacy module:

```json
{
  "rules": {
    "no-restricted-imports": [
      "error",
      {
        "patterns": [
          {
            "group": ["src/legacy/orm", "src/legacy/orm/*"],
            "message": "Use src/postgresv2/Client instead. Legacy ORM is deprecated."
          }
        ]
      }
    ]
  }
}
```

This puts a fence around the code. The tooling depends on your stack, but the boundary must be automatic. Humans forget. CI is annoying, but at least it is consistently annoying.

### Provide a Blessed Example

You cannot only tell people what not to copy; you must show them what to copy. A strong paved path needs a current reference implementation. That might be a template repository, a scaffolding tool, or a "kitchen-sink" example service. It should answer the boring but critical questions: How do I create a new service? How do I add tracing? How do I talk to the database? How do I expose metrics? How do I configure retries and timeouts? How do I write tests? How do I deploy it?

This reference must be actively maintained. A reference service that rots becomes another bad teacher. When a developer asks, “What should I copy?”, there should be one obvious answer.

## Old Code Is Not a Moral Failure

A mature codebase is an archaeological record. You can often look at a service and tell which era of the company’s architecture produced it. That is expected. Old code is not automatically bad; sometimes the best engineering decision is to leave it alone. But old code becomes dangerous when it keeps pretending to be current.

This perspective conflicts slightly with the classic “Broken Window Theory” of software engineering, which suggests that visible neglect invites more neglect. That is a useful warning, but it can push teams toward low-value cleanup work just for consistency. Isolated, clearly marked legacy code is not a broken window. It is closer to a sealed-off room: not where new work should happen, but still present because removing it would cost more than leaving it alone.

The real broken window is old code that still looks like an approved pattern. When legacy code is allowed to influence new construction, the debt spreads. When it is labeled, fenced off, and blocked from new imports, you can protect new work without pretending every old service deserves a rewrite.

Keep the fossils if you need them. Just label them clearly, fence them off, and make sure nobody mistakes them for the paved path.
