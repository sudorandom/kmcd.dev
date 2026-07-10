---
title: "Closing the Alert Is Not Enough"
date: "2026-08-18T10:00:00Z"
categories: ["article"]
tags: ["software-engineering", "architecture", "debugging", "reliability"]
description: "Clearing an alert is only the first step. A good technical investigation explains how the system failed, why the failure was possible, and why the broken code once seemed reasonable."
slug: "closing-the-alert-is-not-enough"
cover: "cover.svg"
images: ["/posts/closing-the-alert-is-not-enough/cover.svg"]
type: "posts"
devtoSkip: true
---

Clearing an alert can be quick. Understanding a failure is harder.

When a PagerDuty alert fires, the first job is to restore service. That part is non-negotiable. Roll back the deploy, restart the worker, disable the feature flag, drain the bad host, or do whatever else gets the system healthy again.

The mistake is treating that as the end of the investigation.

If the checkout service starts timing out, you can often find the obvious broken thing. A query is slow. A dependency is failing. A cache is cold. A config changed. Fix that one thing and the alert clears.

But clearing the alert only proves that the immediate symptom went away. It does not prove that the team understands the failure.

A complete investigation answers three questions: what failed, what allowed it to fail, and **what old constraint made the broken design seem reasonable**.

## Phase 1: What failed

An alert is just a symptom. A 5xx spike, a crash loop, or a latency page tells you something is wrong, but it does not explain what failed.

The first step is to translate the signal into a mechanical description of the failure.

"The service is slow" is not a finding.

"Latency on `/checkout` spiked to 4.2s at 14:37 UTC, starting five minutes after the deploy, and almost every slow request is blocked on `GetEligiblePromotions`" is a finding.

That level of detail matters because it narrows the problem from "the system is behaving strangely" to "this specific path is behaving differently under these conditions."

Suppose checkout starts timing out during peak traffic. Metrics show the latency spike is isolated to one endpoint. Traces show most of the time is spent waiting on a database query. The query text has not changed, but the database plan has.

The service is not generically slow. One request path is issuing a query that scans far more rows than it used to. Under normal traffic, the query finishes slowly enough to be annoying but not slow enough to page anyone. Under peak traffic, it saturates the database connection pool and causes checkout requests to pile up.

Ideally, you can reproduce the issue. Reproducing the same traffic level, data shape, or race condition turns a theory into something you can test.

But production failures are not always polite enough to be reproducible. Some depend on timing, traffic mix, bad data, cloud provider behavior, or a specific sequence of events that you may never recreate exactly.

When that happens, the next best thing is a theory that makes predictions. What evidence should exist if your theory is correct? What metric should move? What log line should appear? What change should prevent the failure from happening again?

That is the real goal of Phase 1: explain the failure well enough that your fix is not just a lucky guess.

## Phase 2: What made the failure possible

Once you know how the system broke, ask why that failure was possible.

A timeout, nil pointer dereference, or panic is not the whole cause. It is where the system finally admitted something had gone wrong.

The more useful question is: what conditions allowed this failure to happen?

In the checkout example, the immediate problem might be a full table scan. The query used to be fast, but now it scans the entire `promotions` table. That explains the timeout, but it still does not explain why the timeout became possible.

So you keep digging.

The query slowed down because an engineer deliberately dropped the index to save database storage and improve write latency. A database observability tool had flagged the index as unused over a 30-day window, prompting a cleanup PR. The reviewer saw the telemetry recommendation and approved it. Both of them missed the seasonal promotion path, which only runs during large yearly campaigns and relies entirely on that specific index.

The failed query matters, but the query is not the whole story. The dangerous condition was that the index looked unused to automation but was still required by a low-frequency business process. The migration review process did not catch that distinction. The available index-usage data did not make it visible. The tests did not include the data shape that made the query expensive.

That changes what you fix.

You can restore the index. You probably should. Production is on fire, and nobody gets bonus points for admiring the flames.

But if you stop there, the same category of failure can happen again. A better fix might include a migration lint rule, a required review step for dropping indexes, query plans in CI for critical paths, or documentation that marks certain indexes as tied to low-frequency workflows.

The point is not to find one magical "root cause" and declare the mystery solved. Real incidents usually have multiple contributing conditions. A bug becomes an incident because several assumptions line up badly.

This is the useful part of the [Five Whys](https://en.wikipedia.org/wiki/Five_whys) technique: not mechanically asking "why" five times, but refusing to stop at the first plausible answer. Keep asking why until you find the conditions that made the failure possible, likely, or invisible.

## Phase 3: Why the old decision seemed reasonable

This is the part most investigations skip.

Once you find the broken code, the temptation is to clean it up. Delete the weird condition. Rename the confusing field. Replace the magic number. Remove the branch that looks unnecessary.

Sometimes that is exactly right. Sometimes the old code really is just wrong.

But code rarely enters a codebase as random nonsense. *Well, maybe it does a bit more, now...* But usually all code solved a real problem under constraints that existed at the time. Those constraints may be gone now or this section is code is being executed for purposes it was never intended for originally.

Before you change the strange part, dig into the history. Run [`git blame`](https://git-scm.com/docs/git-blame), read the old pull request, search for the related ticket, and check whether there was an architecture decision record, or [ADR](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions). You are reconstructing the original engineer's mental model.

This is not about assigning blame. It is [Chesterton's Fence](https://en.wikipedia.org/wiki/G._K._Chesterton#Chesterton's_fence) applied to software: don't tear something down until you understand why it was built.

```go
// Why is this limit 47 and not 50?
if count > 47 {
    return ErrTooManyItems
}
```

A limit like `47` looks suspicious. It practically begs to be rounded up. But there might be a reason. Maybe a downstream API accepts 48 items, and the caller needs one reserved slot for a synthetic entry. Maybe an old mobile client crashes above that size. Maybe the value was chosen to keep a response under a packet, page, or memory limit.

Or maybe it is nonsense.

You do not know until you check.

The same applies to the checkout incident. Why did that index exist in the first place? Why did the seasonal promotion path use a different filter pattern? Why did the engineering team trust a 30-day metric as a complete signal for safe deletion? Why did nobody document that the index was tied strictly to Black Friday traffic?

Those questions can be uncomfortable because they reveal old tradeoffs. Maybe the team moved fast because the promotion system had to launch before Black Friday. Maybe the seasonal path was supposed to be temporary. Maybe the database was small enough at the time that nobody cared about the query plan. Maybe the engineer who knew all of this left two years ago, and the only remaining documentation is a comment in a migration file that says "needed for campaign traffic."

That history matters.

Without it, you might remove a safeguard because it looks like dead code. You might "simplify" a limit that was quietly protecting a dependency. You might replace a weird workaround with a cleaner bug.

Understanding the original intent does not mean preserving the old decision forever. It means you know which constraint you are removing, replacing, or deciding no longer matters.

Sometimes the conclusion is: this made sense then, but it does not anymore.

That is a good outcome. Now you can remove it deliberately, document why the old constraint no longer applies, and leave the next person a better trail than the one you found.

Not every alert requires a week of archaeology, but stopping at the immediate fix guarantees the system will surprise you again.

Fixing the code restores the service. Understanding the context fixes the system.
