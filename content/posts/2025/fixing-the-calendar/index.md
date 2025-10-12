---
categories: ["article"]
tags: ["thoughts"]
date: "2025-09-22T10:00:00Z"
description: "Can our calendar be better?"
cover: "cover.jpg"
images: ["/posts/fixing-the-calendar/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Fixing the Gregorian Calendar"
slug: "fixing-the-calendar"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/fixing-the-calendar/
draft: true
---

The Gregorian calendar is a piece of legacy software, a mess of inconsistent month lengths and arbitrary rules we’ve all memorized as workarounds. As a developer, I look at broken systems and want to refactor them. This post is that refactoring process: a journey that started with trying to fix the month and ended with the realization that the month itself is the bug.

---

## What's wrong with our calendar?

It's an absolute mess. Months have varying numbers of days, which isn't just annoying; it makes planning harder. For example, budgeting is simpler for a 28-day month than for a 31-day one. This may seem trivial, but it's a strange flaw in a system we use every day.

The current structure was born from slow evolution and a series of adjustments. It's full of compromises and "let's fix this one thing" patches. As a result, we are left with a calendar system that fails at one of its initial goals: tracking the lunar cycle. Lunar months lost out to the solar year because we, reasonably, wanted seasons and holidays to be consistent. This is where the current calendar breaks down, because the lunar cycle and the solar year don't neatly align.

The naming convention is another glaring bug. Consider September, October, November, and December. Their Latin roots (*septem* (7), *octo* (8), *novem* (9), and *decem* (10)) clearly point to their original positions in an older Roman calendar. Yet, today they are our 9th, 10th, 11th, and 12th months. This historical mess is largely thanks to Julius Caesar, who inserted new months at the beginning of the year, shifting the old ones out of place.

So what are we left with?
- Months are inconsistent in length.
- The naming of months is inconsistent with their meaning.
- We have vestigial concepts that have lost their ties to their original purpose.

## Part I: The Seductive Logic of a 13-Month Year

My first pass at a fix was all about logic. I designed a clean, predictable system based on a perfect grid.

### **The Spec: A Perfect Grid**
The idea was a perpetual calendar:
* **13 Months,** each 28 days long.
* **4 Perfect Weeks** per month, starting on a Monday and ending on a Sunday. Always.
* **1 "Year Day"** at the end. `13 × 28 = 364`. The 365th day is a special holiday outside the weekly cycle, with a second "Leap Day" when we need it.

To see the difference, compare a perfect 13th-month calendar month with our current January 2026.

**"Perfect" Month**
| Mon | Tue | Wed | Thu | Fri | Sat | Sun |
|---|---|---|---|---|---|---|
| 1 | 2 | 3 | 4 | 5 | 6 | 7 |
| 8 | 9 | 10 | 11 | 12 | 13 | 14 |
| 15 | 16 | 17 | 18 | 19 | 20 | 21 |
| 22 | 23 | 24 | 25 | 26 | 27 | 28 |

**January 2026**
| Mon | Tue | Wed | Thu | Fri | Sat | Sun |
|---|---|---|---|---|---|---|
| | | | 1 | 2 | 3 | 4 |
| 5 | 6 | 7 | 8 | 9 | 10 | 11 |
| 12 | 13 | 14 | 15 | 16 | 17 | 18 |
| 19 | 20 | 21 | 22 | 23 | 24 | 25 |
| 26 | 27 | 28 | 29 | 30 | 31 | |


The appeal is its deterministic nature. The 10th of *any* month is always a Wednesday. Annoying date-based math for things like budgets, sprints, and payroll just becomes simple. It’s a clean API for time.

### **The Breaking Changes: Where It All Falls Apart**
Of course, for all its elegance on paper, this system slams into a wall of reality. The breaking changes are killers.

* **The Fiscal Quarter Problem.** The entire global economy runs on four quarters. `13` is a prime number. You can't divide it cleanly by `4`. You'd have to create awkward `3-3-3-4` month quarters, which reintroduces the same inconsistency we were trying to fix.
* **The Mother of All Migrations.** This would be Y2K on a global scale. Every piece of software, every legal contract, every database on Earth would need to be rewritten. The cost is basically incalculable.
* **The Unbroken Week.** Tossing in a "timeless" day breaks the continuous seven-day weekly cycle. That cycle has been uninterrupted for millennia and is a cornerstone for major world religions. It's not just a scheduling problem; it's a deeply cultural one.

Fixing the month was creating more problems than it solved. It was a classic dev mistake: I was optimizing a feature without questioning why the feature existed in the first place.

---

### Part II: Why Use Months Anyway? The Case for Weeks

Which led me to the real question: why are we even using months? Seriously, what are they? A bastardization of Latin, the vanity of Roman rulers, and a set of arbitrary units that have nothing to do with our weekly lives. The bug isn't the implementation. The month *itself* is the bug.

So let's just delete the abstraction.

### **The Real Spec: A System We Already Have**
The proposal is to use the **week** as the main unit of time beyond the day. A year is just 52 weeks. And get this: this isn't some new idea I dreamed up. It's already part of the **ISO 8601 international standard**.

The formats are clean and unambiguous:
* **Week:** `YYYY-Www`. Example: **`2025-W41`** represents the 41st week of 2025.
* **Week with weekday:** `YYYY-Www-D`. Example: **`2025-W41-2`** represents the 2nd day (Tuesday) of the 41st week of 2025. (Monday is 1, Sunday is 7).

The system is built and standardized; we just don't use it as our default. Here’s how the first few weeks of 2026 look using the ISO 8601 week-date system, with the Gregorian date (day/month) for comparison. Notice how Week 1 starts on the first Monday of the year.

| ISO Week | Mon | Tue | Wed | Thu | Fri | Sat | Sun |
|---|---|---|---|---|---|---|---|
| **2026-W01** | 29/12 | 30/12 | 31/12 | 01/01 | 02/01 | 03/01 | 04/01 |
| **2026-W02** | 05/01 | 06/01 | 07/01 | 08/01 | 09/01 | 10/01 | 11/01 |
| **2026-W03** | 12/01 | 13/01 | 14/01 | 15/01 | 16/01 | 17/01 | 18/01 |
| **2026-W04** | 19/01 | 20/01 | 21/01 | 22/01 | 23/01 | 24/01 | 25/01 |
| **2026-W05** | 26/01 | 27/01 | 28/01 | 29/01 | 30/01 | 31/01 | 01/02 |

Note that week 1 of 2026 starts in 2025. This can be a bit confusing, but it's fundamentally caused by Earth years not being evenly divisible by 7 days.

### **Addressing the Concerns**
This idea has its own set of hurdles, but they feel more like matters of habit than hard blockers.

* **The Loss of Seasonal Anchors.** This one's easy. Months are already terrible seasonal anchors. "December" is winter here in Copenhagen but high summer in Cape Town. Tying months to seasons is just a Northern Hemisphere bias.
* **The Narrative Void.** The argument that we need months for "chapter breaks" in our year assumes we're incapable of making new patterns. We could easily create new milestones around 10-week blocks or the four 13-week quarters. The narrative doesn't disappear; it just gets refactored.
* **The "Impossible" Migration.** Here's the shift in thinking. The migration doesn't have to be a revolution; it can be an **evolution**. As I've learned living in Denmark, using week numbers for planning is completely normal. It coexists with the standard calendar. I've had people invite me to events in "week 45." The first time, I had to look it up. The fifth time, I just knew. Intuitive understanding is built through use.

## The Migration Path

The end result of this thought exercise is that I'm just going to personally try to use week numbers more. I've added week numbers to my calendar apps and maybe next time I

We don't need to burn the old system down. We just need to start using the better one that's been running in parallel all along.

1.  **Start Using Weeks.** In your next meeting invite, add the week number. "Let's sync up in Week 45."
2.  **Adopt Ordinal Dates.** For logging, file naming, or anywhere you need a simple, unambiguous timestamp, use the ordinal format `YYYY-DDD` (e.g., `2025-280`).
3.  **Favor the Right Tools.** Use applications that properly support ISO 8601.
4. Add week numbers to your

Look, this isn't a fast change. It's not supposed to be. But by pushing for a better standard that *already exists*, we can start the slow process of deprecating the mess we have now. The migration starts with us.
