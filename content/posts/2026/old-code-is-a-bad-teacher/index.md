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
draft: true
---

A large codebase does more than run software. It teaches people how to write the next piece of it.

That is usually a good thing. Developers do not start every task from a blank page. They search for similar code, copy the shape of it, and adapt it to the problem in front of them. This is not laziness. It is one of the main ways people learn a system.

Need to add a span to the current trace? Find another handler that does it. Opening a new database connection? Search for an existing repository or client. Registering a new service? Copy the structure from one that already works. In a healthy codebase, this is a superpower, and it is one of the keys to making developers effective at solving problems.

Over time, the organization settles on preferred ways to do common tasks. There is a standard tracing middleware. There is a database wrapper that handles pooling, metrics, and retries. There is a scaffolding tool for new services. There is a common layout for configuration, logging, tests, and deployment. These patterns become the paved path. They make onboarding faster, make services easier to recognize, and reduce the number of decisions every engineer has to make before they can ship useful work.

But there is a catch: the codebase keeps teaching people even after the lesson is out of date.

## Standards Expire

The “right way” to build something today will eventually become the old way. You migrate to a new tracing standard. You replace an ORM. You change how services talk to each other. You introduce a better configuration system. You decide that every new service should use a different framework, transport, deployment model, or observability stack.

That is normal. A codebase that never changes is usually either dead or terrifying.

The obvious answer is to migrate everything to the new standard. Sometimes that is exactly what you should do. If the old pattern is insecure, unreliable, or actively blocking product work, it needs to go. But in a large codebase, full migration is often not worth it.

Some old services are stable. Some are barely touched. Some sit in corners of the business where rewriting them would create more risk than value. Spending weeks modernizing a boring internal service just so it looks like the newer services may be a bad trade. Leaving old code alone can be the pragmatic choice.

The problem is not that the old code exists. The problem is that old code often looks normal.

## The Copy-Paste Hazard

Developers learn by example, especially new developers. When someone joins a team, they will search the codebase for similar work. They will find a service that already does something close to what they need. If that service uses an outdated database layer, an old tracing package, or a retired service pattern, they may copy it without realizing they are copying a historical artifact.

From their perspective, the code works. It passed review at some point. It lives in the same repository as everything else. It may even be part of a critical production system. So why would they assume it is the wrong example?

This is how technical debt spreads. Not through a big architectural decision or a dramatic rewrite gone wrong, but through small, reasonable acts of imitation. Someone copies a deprecated client into a new feature. Someone uses an old test helper because it was the first one they found. Someone creates a new service from a legacy service because the legacy service had the closest business logic. Now the old pattern has reproduced.

If this keeps happening, your new standard never really becomes the standard. It becomes one more option in a growing pile of options. That is how large codebases end up with five ways to talk to the database, three logging libraries, two service templates, and a Slack thread every few months where someone asks, “Wait, which one are we supposed to use now?”

Computers are very literal, but codebases are weirdly good at folklore.

## Mark What Should Not Be Copied

If you decide to leave old patterns in place, you need to make their status obvious. You cannot rely on memory, tribal knowledge, or the one senior engineer who knows which package is cursed. You also cannot assume people will read an architecture document before copying a working example.

The warning needs to appear where developers actually are: in the editor, in code search, in review, and in CI.

### Deprecate Loudly

Use your language’s deprecation tools. If you have `@Deprecated`, `[Obsolete]`, `// Deprecated:`, or an equivalent mechanism, use it aggressively. Deprecated APIs should look deprecated in the editor. Modern IDEs will often strike through deprecated symbols or surface warnings inline, which is exactly the kind of friction you want.

A deprecation message should also say what to use instead.

Bad:

```go
// Deprecated.
```

Better:

```go
// Deprecated: use postgresv2.Client instead. New services should not import this package.
```

It also helps when the replacement API has a similar shape to the old one. If the migration requires every caller to rethink the entire interaction model, people are more likely to postpone it, work around it, or keep copying the deprecated version. A better replacement should not only be better in theory. It should be easy to adopt in practice.

The goal is not just to flag the old code as obsolete. The goal is to redirect people before they copy it into something new.

### Rename Legacy Code Honestly

Names carry permission. A package named `events` sounds safe. A package named `legacyevents` sounds suspicious. A file named `UserEventPublisher.ts` looks like a reasonable thing to copy. A file named `UserEventPublisher_LegacyDoNotCopy.ts` makes people pause.

Yes, the blunt name is ugly, but that is also the point.

If a module should only exist to keep an old system alive, do not give it a clean, timeless name. Make its status obvious. Put `legacy`, `deprecated`, `old`, or `do_not_copy` directly in the path if you need to. Subtle warnings are easy to miss. Ugly names are harder to accidentally cargo-cult.

### Block New Imports

Deprecation warnings are helpful, but they are still warnings. If the old pattern really should not spread, enforce the boundary with tooling.

Use static analysis, lint rules, build constraints, or dependency checks to prevent new code from importing legacy packages. For instance, TypeScript teams can configure ESLint’s `no-restricted-imports` rule to block imports from a legacy module:

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

This is especially useful when the old code still needs to exist for a while. You are not deleting it. You are putting a fence around it.

The exact tool depends on the stack. TypeScript teams can use ESLint. Java teams might use ArchUnit to define boundary tests in code. Go teams might use import restrictions, package structural constraints, or a small custom analyzer. The important part is that the boundary is automatic. Humans forget. CI is annoying, but at least it is consistently annoying.

### Maintain a Gold Standard Example

You cannot only tell people what not to copy. You need to give them something better to copy.

A strong paved path needs a current reference implementation. That might be a template repository, a scaffolding command, a kitchen-sink service, or a small example app that shows the recommended patterns in one place. It should answer the boring questions: How do I create a new service? How do I add tracing? How do I talk to the database? How do I expose metrics? How do I configure retries and timeouts? How do I write tests? How do I deploy it?

This example needs to stay maintained. A reference service that rots becomes another bad teacher. The ideal outcome is simple: when a developer asks, “What should I copy?”, the answer should be obvious.

## Old Code Is Not a Moral Failure

A mature codebase is an archaeological record. You can often look at a service and tell which era of the company’s architecture produced it. That is fine and expected.

Old code is not automatically bad. Sometimes it is stable, boring, and valuable. Sometimes the best engineering decision is to leave it alone. But old code becomes dangerous when it keeps pretending to be current code.

This perspective conflicts slightly with the classic “Broken Window Theory” of software engineering, which suggests that visible neglect invites more neglect. That is a useful warning, but it can also push teams toward low-value cleanup work just to make everything look consistent. Isolated, clearly marked legacy code is not quite the same thing as a broken window. It is closer to a sealed-off room: not part of the current living space, not where new work should happen, but still present because removing it would cost more than leaving it alone.

The broken window is not the mere existence of old code. The broken window is old code that still looks like an approved pattern. When legacy code is allowed to influence new construction, the debt spreads. When it is labeled, fenced off, and blocked from new usage, you can preserve the integrity of the active parts of the codebase without pretending every old service deserves a rewrite.

The goal is not to erase every outdated pattern from history. The goal is to stop those patterns from spreading into new work.

Keep the fossils if you need them. Just label them clearly, fence them off, and make sure nobody mistakes them for the paved path.
