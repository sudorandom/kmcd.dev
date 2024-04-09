---
categories: ["article"]
tags: ["programming", "grpc", "api", "rest"]
date: "2024-04-16"
description: "Meaningful names help make code searchable so think twice before naming everything 'entity'"
cover: "cover.jpg"
images: ["/posts/greppable/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Making Greppable Code"
slug: "greppable"
type: "posts"
devtoSkip: false
canonical_url: https://sudorandom.dev/posts/greppable
---

Have you ever felt like you're sifting through lines of code, desperately searching for that elusive function or variable? Fear not, for there's a way to make your code more discoverable: **greppable naming conventions**. 

### What is Greppability?

Drawing inspiration from the powerful [Linux command `grep`](https://www.gnu.org/software/grep/manual/grep.html) used for text searching, greppable code refers to code that's easy to search through using clear and meaningful names. Just like `grep` helps you quickly find specific text within a file, greppable names allow you to efficiently locate relevant code sections by searching for terms that reflect their purpose.

### Crafting Greppable Names

Here are some key principles to follow:

* **Specificity:** Ditch generic names like `processData()` or `utils`. Opt for terms that reflect the specific content (`cleanAndValidateUserData()`, `orderId`).
* **Priority:** It is far more important for filenames, class names and methods to have specific names than it is for variables inside of functions. Variables inside of functions already have a lot of implied context, like the filename, class (if your language uses classes) and the function that the variable appears in. All of that extra context should be greppable but the variables probably don't need to be.
* **Verbs for Methods:** Methods describe actions. Use verbs at the beginning of method names to convey their purpose (`calculateOrderTotal()`, `updateCustomerRecord()`).
* **Consistency:** Maintain a consistent naming style throughout your codebase (e.g. camelCase or snake_case) and always use the 'standard' for your language or framework.
* **Abbreviations with Caution:** Use abbreviations sparingly and only for widely recognized terms (e.g. `HTTP`, `XML`). Overuse can hinder readability. If your variable names look like 2010s-era tech startups, you're probably doing it wrong.
* **Dynamic Dispatch:** Doing some more complex programming patterns like dynamic dispatching can greatly hurt the ability for the code to be greppable. It can also add indirection so be sure you use these methods sparingly and add more documentation where needed to help lost souls who are tracing code through the codebase.

### The Search Advantage of gRPC

Here's where things get interesting for APIs. Compared to RESTful APIs with their predefined resource structures, gRPC (and other RPC-based APIs) offer greater flexibility in naming methods. This freedom allows you to leverage greppable naming conventions to their full potential.

For instance, in a RESTful API, an endpoint for updating a user profile might be named `/users/:id`. While functional, this doesn't explicitly convey the action performed. In a gRPC API, however, you could have a method named `UpdateUserProfile(UpdateRequest request)`, which clearly describes the purpose and is much easier to search for in the code.

**The Greppability Benefit**

Meaningful names not only make code easier to understand but also significantly reduce maintenance time. When you (or another developer) revisit your codebase later, clear names eliminate the need to decipher cryptic naming choices.

By embracing specific and consistent naming conventions, you can transform your code into a truly greppable resource, where the functionalities you need are readily discoverable through simple searches.
