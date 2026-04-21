---
categories: ["article"]
tags: ["programming", "grpc", "api", "rest", "devops"]
date: "2026-04-21"
cover: "cover.svg"
images: ["/posts/greppable/cover.svg"]
description: "Why naming is about navigation, not just aesthetics, especially in a multi-repo world."
title: "The Case for Greppable Code"
slug: "greppable"
---

{{< disclaimer >}}
This article was originally published in April 2024. It was republished in April 2026 after some significant editing and modernization.
{{< /disclaimer >}}

Imagine staring at a production log with a generic error in a function called `processData()`. You search the codebase, only to find forty different functions with that exact name spread across five repositories. This is the opposite of searchable code.

**Greppability**{{< footnote 3 >}} is a measure of how easily a human can find specific logic using simple text search. Modern IDEs are great, but greppability is the safety net for when they fail: during code reviews, in terminal sessions, while scanning traces, or when navigating a massive mesh of microservices.

### Navigation is the Job

Developers spend a considerable amount of effort on program comprehension, which includes reading and understanding source code. In fact, research shows that programmers spend roughly **70% of their time**{{< footnote 1 >}} on code comprehension.

When you use a generic name like `entity`, you increase the "search cost"{{< footnote 2 >}} of the workday. Research indicates that when programmers deal with high comprehension effort, they navigate and make edits at a significantly slower rate. Choosing specific names like `CustomerBillingRecord` ensures your search result is a surgical strike rather than a list of a hundred collisions.

### Spanning Across Repos

The real power of greppability shows up in **polyglot environments**. When your infrastructure spans multiple repositories and languages, like Go, TypeScript, and Python, your IDE "Jump to Definition" often stops at the edge of the current project. 

In these distributed systems, a unique string is the only universal bridge. A specific domain term, like `SubscriptionRenewalWebhook`, connects a frontend UI component to a backend service implementation across repo boundaries. If both use the same unique name, you can track a feature across the entire stack in seconds without needing a specialized indexer for every language you use.

> **Key idea:** Greppability turns your codebase into a searchable database. By treating unique strings as the "primary keys" of your architecture, you bypass the limitations of IDEs and language boundaries, especially in multi-repo environments where traditional navigation tools often break.

### How to Write for Grep

* **The Log-to-Code Pipeline:** If a method handles a critical business action, name it something distinct. If a system crashes at 3:00 AM and the log says `Error in CalculateRegionalTax()`, you have found the bug before you have even opened your editor.
* **Avoid Generic Sinks:** Names like `data`, `info`, `manager`, or `entity` are search poison. `orderValidationLogic` is longer, but it is a direct hit for a search engine.
* **Establish a Naming Hierarchy:** Focus your "naming budget" where the scope is widest. Filenames, class names, and API methods must be unique. Local variables inside a five-line function, like `i` or `buf`, can stay generic because their search boundary is tiny.
* **Beware of Dynamic Magic:** If your language uses reflection or string interpolation to call methods, such as `this.call("prefix_" + action)`, you have killed the ability to grep for the implementation.

### The gRPC Search Advantage

This is where RPC-based designs offer a distinct advantage over REST. In a REST architecture, searching for an "update" action usually requires a two-step mental grep: find the path (`/users/:id`) and then filter by the HTTP verb (`PUT`). 

In gRPC, the method name is a **globally unique string**. 

| Architecture | Search Precision | Debugging Speed |
| :--- | :--- | :--- |
| **REST** (`PUT /user`) | Low (Many paths use `PUT`) | Slow: requires manual filtering. |
| **gRPC** (`UpdateUserBio`) | **High (1-2 results)** | **Fast: direct jump to logic.** |

In a gRPC environment, `UpdateUserBio()` appears identically in your `.proto` file, your server code, your client code, and your monitoring dashboard. It is the ultimate greppable identifier.

### Summary

**Greppability** is an engineering-first term. It moves naming away from subjective aesthetics and toward functional operability. When you tell a teammate that a name is not very greppable, you are not critiquing their style: you are pointing out a future debugging bottleneck.

{{< references >}}
{{< footnotelist >}}
{{< footnoteitem 1 "Comprehension Effort and Programming Activities: Related? Or Not Related? (2018)" "https://akondrahman.github.io/files/papers/msr18_chall.pdf" >}}
{{< footnoteitem 2 "How Developers Search for Code: A Case Study (Google Research)" "https://research.google/pubs/how-developers-search-for-code-a-case-study/" >}}
{{< footnoteitem 3 "grep (global regular expression print) is a command-line utility for searching plain-text data sets for lines that match a regular expression." >}}
{{< /footnotelist >}}
{{< /references >}}
