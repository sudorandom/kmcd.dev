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
draft: true
---

Clearing an alert can be quick. Understanding a failure is harder.

When a PagerDuty alert fires, the first job is to restore service. That part is non-negotiable. Roll back the deploy, restart the worker, disable the feature flag, drain the bad host, or do whatever else gets the system healthy again.

The mistake is treating that as the end of the investigation.

If the checkout service starts timing out, you can often find the obvious broken thing. A query is slow. A dependency is failing. A cache is cold. A config changed. Fix that one thing and the alert clears.

But clearing the alert only proves that the immediate symptom went away. It does not prove that the team understands the failure.

A complete investigation answers three questions: how it broke, why it broke, and why we thought it was a good idea to begin with. The best teams work through all three.

## Phase 1: Identifying the Mechanics

An alert is just a symptom. A 5xx spike, a crash loop, or a latency page tells you something is wrong, but it does not explain what failed.

The first step is to translate the signal into a mechanical description of the failure.

"The service is slow" is not a finding.

"Latency on `/checkout` spiked to 4.2s at 14:37 UTC, starting five minutes after the deploy, and almost every slow request is blocked on `GetEligiblePromotions`" is a finding.

That level of detail matters because it narrows the problem from "the system is behaving strangely" to "this specific path is behaving differently under these conditions."

Suppose checkout starts timing out during peak traffic. Metrics show the latency spike is isolated to one endpoint. Traces show most of the time is spent waiting on a database query. The query text has not changed, but the database plan has.

Now you have mechanics.

The service is not generically slow. One request path is issuing a query that scans far more rows than it used to. Under normal traffic, the query finishes slowly enough to be annoying but not slow enough to page anyone. Under peak traffic, it saturates the database connection pool and causes checkout requests to pile up.

Ideally, you can reproduce the issue. Reproducing the same traffic level, data shape, or race condition turns a theory into something you can test. It lets you prove your fix works instead of deploying a change and hoping the alert stays quiet.

But production failures are not always polite enough to be reproducible. Some depend on timing, traffic mix, bad data, cloud provider behavior, or a specific sequence of events that you may never recreate exactly.

When you cannot reproduce the issue, you still need a clear enough explanation to make predictions. What evidence should exist if your theory is correct? What metric should move? What log line should appear? What change should prevent the failure from happening again?

That is the real goal of Phase 1: explain the failure well enough that your fix is not just a lucky guess.

## Phase 2: Finding the Conditions

Once you know how the system broke, ask why that failure was possible.

A timeout is not the whole cause. A nil pointer dereference is not the whole cause. A panic is not the whole cause. Those are consequences. They are where the system finally admitted something had gone wrong.

The more useful question is: what conditions allowed this failure to happen?

In the checkout example, the immediate problem might be a full table scan. The query used to be fast, but now it scans the entire `promotions` table. That explains the timeout, but it still does not explain why the timeout became possible.

So you keep digging.

The query slowed down because an automated migration cleanup tool dropped an index it flagged as unused. The tool flagged it as unused because it only analyzed recent production traffic. It missed the seasonal promotion path, which only runs during large campaigns and uses a different filter pattern.

Now the investigation is getting somewhere.

The failed query matters, but the query is not the whole story. The deeper problem is that the system had an index that looked unused to automation but was still required by a low-frequency business process. The migration review process did not catch that distinction. The observability around index usage did not make it visible. The tests did not include the data shape that made the query expensive.

That gives you a different class of fix.

You can restore the index. You probably should. Production is on fire, and nobody gets bonus points for admiring the flames.

But if you stop there, the same category of failure can happen again. A better fix might include a migration lint rule, a required review step for dropping indexes, query plans in CI for critical paths, or documentation that marks certain indexes as tied to low-frequency workflows.

The point is not to find one magical "root cause" and declare the mystery solved. Real incidents usually have multiple contributing conditions. A bug becomes an incident because several assumptions line up badly.

This is the useful part of the [Five Whys](https://en.wikipedia.org/wiki/Five_whys) technique: not mechanically asking "why" five times, but refusing to stop at the first plausible answer. Keep asking why until you find the conditions that made the failure possible, likely, or invisible.

## Phase 3: Understanding the Original Intent

This is the part most investigations skip.

Once you find the broken code, the temptation is to clean it up. Delete the weird condition. Rename the confusing field. Replace the magic number. Remove the branch that looks unnecessary.

Sometimes that is exactly right. Sometimes the old code really is just wrong.

But code rarely enters a codebase as random nonsense. It usually solved a real problem under constraints that existed at the time. Those constraints may be gone now. They may still exist. You need to know which.

Before you change the strange part, dig into the history. Run [`git blame`](https://git-scm.com/docs/git-blame), read the old pull request, search for the related ticket, and check whether there was an architecture decision record, or [ADR](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions). You are reconstructing the original engineer's mental model.

This is not about assigning blame. It is about [Chesterton's Fence](https://en.wikipedia.org/wiki/G._K._Chesterton#Chesterton's_fence): don't tear something down until you understand why it was built.

```go
// Why is this limit 47 and not 50?
if count > 47 {
    return ErrTooManyItems
}
```

A limit like `47` looks suspicious. It practically begs to be rounded up. But there might be a reason. Maybe a downstream API accepts 48 items, and the caller needs one reserved slot for a synthetic entry. Maybe an old mobile client crashes above that size. Maybe the value was chosen to keep a response under a packet, page, or memory limit.

Or maybe it is nonsense.

You do not know until you check.

The same applies to the checkout incident. Why did that index exist in the first place? Why did the seasonal promotion path use a different filter pattern? Why did the migration cleanup tool trust recent traffic as a complete signal? Why did nobody document that the index was tied to campaign traffic?

Those questions can be uncomfortable because they reveal old tradeoffs. Maybe the team moved fast because the promotion system had to launch before Black Friday. Maybe the seasonal path was supposed to be temporary. Maybe the database was small enough at the time that nobody cared about the query plan. Maybe the engineer who knew all of this left two years ago, and the only remaining documentation is a comment in a migration file that says "needed for campaign traffic."

That history matters.

Without it, you might remove a safeguard because it looks like dead code. You might "simplify" a limit that was quietly protecting a dependency. You might replace a weird workaround with a cleaner bug.

Understanding the original intent does not mean preserving the old decision forever. It means you change the system with your eyes open.

Sometimes the conclusion is: this made sense then, but it does not anymore.

That is a good outcome. Now you can remove it deliberately, document why the old constraint no longer applies, and leave the next person a better trail than the one you found.

## The Payoff

Going through all three phases takes more time than closing the ticket, but it is how a team stops buying the same incident twice.

The first time, you fix a timeout.

The second time, you notice three teams have been fighting the same database assumption.

The third time, you stop treating it as a bug and start treating it as architecture.

That is the real value of a good investigation. It updates the team's understanding of the system. It turns one annoying failure into better tooling, better reviews, better tests, and better design constraints.

Not every incident needs a week-long archaeology project. Some bugs are small. Some fixes are obvious. Sometimes the right answer really is "we forgot to check for nil."

But when a failure exposes a surprising system behavior, a hidden dependency, or a piece of code nobody understands anymore, stopping at the failing line is too shallow.

The investigation is not done when the alert clears.

It is done when you actually learn something.
