---
categories: ["article"]
tags: ["thoughts", "calendar", "standards", "time"]
date: "2026-01-01T10:00:00Z"
description: "Are months considered harmful?"
cover: "cover.svg"
images: ["/posts/fixing-the-calendar/cover.svg"]
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

The calendar we all live by is a shambling mess. It has arbitrary month lengths, misnamed months, and rules so absurd we rely on nursery rhymes just to remember them. For something that structures our lives, it is clunky and illogical. This post is my attempt to design something that actually makes sense.

## What's wrong with our calendar?

Our current calendar is far from optimal. Months have varying numbers of days, which isn't just annoying; it makes planning harder. For example, budgeting is made harder because month lengths don't just vary in days, but in the number of pay periods or expense cycles they contain, leading to unpredictable cash flow. This may seem trivial, but it's a strange flaw in a system we use every day.

Built through centuries of compromises and quick fixes, what began as lunar tracking has decayed into something that does neither job well. While not directly relevant to most modern daily planning, the conceptual failure to maintain a consistent natural rhythm highlights the calendar's fundamental breakage from its origins. This breakdown stems from prioritizing the solar year for stable seasons and holidays, an aim that directly conflicts with the irregular lunar cycle.

The naming convention is another glaring bug. Consider September, October, November, and December. Their Latin roots (*septem* (7), *octo* (8), *novem* (9), and *decem* (10)) clearly point to their original positions. Yet, today they are our 9th, 10th, 11th, and 12th months. This historical mess is the result of early Roman calendar reforms, where the start of the year was shifted, leaving several months with names that no longer match their position.

So what are we left with?
- Months are inconsistent in length.
- The naming of months is inconsistent with their meaning.
- We have vestigial concepts that have lost their ties to their original purpose.

## Part I: 13-Month Year

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

### Where My Dream Falls Apart
Of course, for all its elegance on paper, this system slams into a wall of bitter reality.

The entire global economy runs on four quarters. `13` is a prime number, meaning you can't divide it cleanly into equal quarters. This would necessitate creating awkward `3-3-3-4` month quarters, which ultimately reintroduces the very inconsistency we were attempting to eliminate.

Implementing such a calendar change would be a monumental undertaking, akin to Y2K on a global scale but vastly more complex due to deeply embedded legal, civil, and international coordination challenges. Every piece of software, every legal contract, and every database on Earth would require extensive rewriting. The associated cost would be unfathomable and would ultimately outweigh any benefits that we'd gain.

Introducing a "timeless" day (or two on leap years) into the calendar disrupts the continuous seven-day weekly cycle. It is... awkward. How would you write the date for this day? This disruption would break critical real-world systems like payroll cycles, recurring shifts, and automated cron jobs that rely on an unbroken weekly sequence. Some people still have to work, so there would be a weird exception for planning for this day. Timekeeping software would need to specifically handle this.

After considering the downsides, I have decided that my dream will continue to stay a dream. Months cannot be fixed... at least not without creating more problems. Who really needs months anyway?

---

### Part II: Deprecating the Month

Why do we use months at all? What purpose do they serve in our daily lives? They're a mashup of bad Latin, Roman vanity, and arbitrary divisions that have no real connection to the 7-day cycle that actually governs our schedules. The bug isn't the *implementation* of the month. The month *itself* is the bug.

So, let's deprecate the abstraction.

### Long live the week
The proposal is to use the **week** as the main unit of time beyond the day. A year is just 52 weeks. And get this: this isn't some new idea I dreamed up. It's already part of the **ISO 8601 international standard**.

The formats are clean and unambiguous:
* **Week:** `YYYY-Www`. Example: **`2025-W41`** represents the 41st week of 2025.
* **Week with weekday:** `YYYY-Www-D`. Example: **`2025-W41-2`** represents the 2nd day (Tuesday) of the 41st week of 2025. (Monday is 1, Sunday is 7).

The system is built and standardized; we just don't use it as our default. Here’s how the first few weeks of 2026 look using the ISO 8601 week-date system, with the Gregorian date (day/month) for comparison.

| ISO Week | Mon | Tue | Wed | Thu | Fri | Sat | Sun |
|---|---|---|---|---|---|---|---|
| **2026-W01** | 29/12 | 30/12 | 31/12 | 01/01 | 02/01 | 03/01 | 04/01 |
| **2026-W02** | 05/01 | 06/01 | 07/01 | 08/01 | 09/01 | 10/01 | 11/01 |
| **2026-W03** | 12/01 | 13/01 | 14/01 | 15/01 | 16/01 | 17/01 | 18/01 |
| **2026-W04** | 19/01 | 20/01 | 21/01 | 22/01 | 23/01 | 24/01 | 25/01 |
| **2026-W05** | 26/01 | 27/01 | 28/01 | 29/01 | 30/01 | 31/01 | 01/02 |

Note that week 1 of 2026 starts in December of 2025. This might seem confusing, but it’s a deliberate design choice. The ISO 8601 standard prioritizes a consistent, unbroken 7-day week. To do this, it defines the first week of the year as the one containing the first Thursday. This means a year's first week can start in the previous calendar year. It's a trade-off: the Gregorian calendar breaks weekday consistency at the year's boundary, while the week-date system prioritizes the weekly cycle *over* the year boundary. In practice, you often stop caring which 'year' a Monday belongs to, because your planning horizon is almost always weeks ahead, not abstract year boundaries.

### Addressing the Concerns
This idea has its own set of hurdles, but they feel more like matters of habit than hard blockers.

The fiscal quarter problem is solved. Remember how the 13-month calendar failed because 13 is a prime number? The week-based system fixes this beautifully. A 52-week year divides perfectly into four 13-week quarters. This is already a common practice in many industries (often called a 4-4-5 calendar) because it makes financial reporting and year-over-year comparisons much more consistent. Far from being a problem, this is a major advantage.

Months are poor seasonal anchors anyway. "December" is winter here in Copenhagen but high summer in Cape Town. Tying months to seasons is just a Northern Hemisphere bias.

We can create new narratives. The argument that we need months for "chapter breaks" in our year assumes we're incapable of making new patterns. We could easily create new milestones around 10-week blocks or the four 13-week quarters. The narrative doesn't disappear; it just gets refactored.

The migration can be an evolution, not a revolution. Here's the shift in thinking. The migration doesn't have to be a revolution; it can be an **evolution**. As I've learned living in Denmark, using week numbers for planning is completely normal. It coexists with the standard calendar. I've had people invite me to events in "week 45." The first time, I had to look it up. The fifth time, I also had to look it up. But I assume that with enough usage I will build an intuition.

## The Migration Path

The end result of this thought exercise is a clear personal commitment: I'm going to lean into using week numbers more in my own life. This isn't a radical proposition; in countries like Denmark and much of Europe, using week numbers for scheduling is completely normal. The migration doesn't have to be a revolution; it’s about consciously adopting a better system that already exists in parallel.

It’s an evolutionary change, and here’s how you can start:

**Add Week Numbers to Your Calendar.** Most digital calendars support this as a display option. Making the week number visible is the first step to making it intuitive. Try it for three months.

**Start Using Weeks in Planning.** In your next meeting invite, add the week number. "Let's sync up in Week 45." Will people look at you weird when you ask that? Probably. But notice how often months stop mattering in your planning.

Look, this isn't a fast change. It's not supposed to be. But by pushing for a better standard that *already exists*, we can start the slow process of deprecating the mess we have now.

---
## References & Further Reading

For those interested in a deeper dive into the topics discussed, here are some of the resources used in researching this article:

*   **On the History of the Roman and Gregorian Calendars:**
    *   The Editors of Encyclopaedia Britannica. ["Roman Republican calendar"](https://www.britannica.com/science/Roman-republican-calendar). *Britannica*.
    *   ["The Origin of the Month Names"](https://www.almanac.com/content/origin-month-names). *Almanac.com*.

*   **On Proposed Alternative Calendars:**
    *   ["International Fixed Calendar"](https://en.wikipedia.org/wiki/International_Fixed_Calendar). *Wikipedia*.
    *   Aldrich, Jeremy. ["The International Fixed Calendar"](https://jeremy-aldrich.com/the-international-fixed-calendar/).

*   **On the ISO 8601 Week-Date System:**
    *   ["ISO week date"](https://en.wikipedia.org/wiki/ISO_week_date). *Wikipedia*.

*   **On Lunar vs. Solar Calendars:**
    *   Taylor, Elise. ["What Is the Difference Between a Lunar & a Solar Calendar?"](https://sciencing.com/what-is-the-difference-between-a-lunar-a-solar-calendar-13710243.html). *Sciencing*.
---

*   **On the 4-4-5 Calendar:**
    *   ["4-4-5 Calendar"](https://en.wikipedia.org/wiki/4-4-5_calendar). *Wikipedia*.
