---
title: "Three "
date: "2026-08-18T10:00:00Z"
categories: ["article"]
tags: ["software-engineering", "architecture", "technical-debt", "productivity"]
description: "TODO"
slug: "three-phases-of-investigation"
cover: "cover.svg"
images: ["/posts/three-phases-of-investigation/cover.svg"]
type: "posts"
devtoSkip: true
draft: true
---

* **The Premise:** Fixing a bug is easy; understanding a failure is hard. A technical investigation is incomplete if you stop at the failing line of code.
* **The Core Argument:** To truly resolve an issue and prevent its recurrence, an engineer must uncover the full picture: *how* it broke, *why* it broke, and *why we thought it was a good idea in the first place*.
* **The Payoff:** Embracing this comprehensive approach reduces future production incidents, fosters empathy for legacy systems, and accelerates your growth as a developer.

---

### Phase 1: The "How" (Identifying the Mechanical Failure)

* **Moving from Symptom to Cause:** When an alert fires (e.g., a spike in 5xx errors or a pod crash loop), you are only seeing the symptom. The first step is isolating the immediate mechanical breakdown.
* **Bounding the Problem:** Using your telemetry—metrics, logs, and distributed traces—to draw a box around the failing component.
* **Replicating the State:** Determining the exact sequence of events, inputs, or environmental triggers required to reproduce the failure. You haven't found the "how" until you can reliably make it break.

### Phase 2: The "Why" (Root Cause Analysis)

* **Going Deeper:** The "how" might be a `nil` pointer or a timeout, but the "why" is the underlying system dynamic that allowed that state to exist.
* **Investigating System Dynamics:** Was this a race condition exposed by higher concurrency? Was it a resource exhaustion issue triggered by an upstream dependency? Did a recent configuration change silently alter the environment?
* **The "Five Whys" Technique:** Iteratively questioning the failure until you reach a fundamental flaw in the logic, architecture, or deployment pipeline.

### Phase 3: The "Context" (The Original Intent)

* **The Missing Link:** This is where most investigations prematurely end. Code rarely enters a codebase broken; it usually worked perfectly within the constraints and assumptions of its time.
* **Software Archeology:** Digging into `git blame`, pull request discussions, and design documents. You are looking for the original context in which this solution made sense.
* **Chesterton's Fence:** The principle that you should not tear down a fence until you understand why it was put up. What were the original developers optimizing for?
* *Example:* Was this feature built rapidly to hit a hard deadline? Was it designed for a scale of 100 requests per minute, but the system now handles 10,000?


* **Empathy and Evolution:** Recognizing that past decisions weren't malicious or incompetent. Understanding the historical context prevents you from implementing a "fix" that accidentally reintroduces an older, forgotten bug.

---

### Conclusion: The Complete Picture

* **The compounding value of deep investigations:** By completing all three phases, you aren't just patching a hole; you are updating your mental model of the entire system.
* **The Engineering Mindset:** High-performing developers don't just react to broken code. They seek to understand the lifecycle of a feature from its inception to its failure, using those lessons to build fundamentally more resilient architecture moving forward.