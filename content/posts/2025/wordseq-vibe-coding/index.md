---
categories: ["article"]
tags: ["gamedev", "react", "typescript", "ai", "llm", "vibe-coding", "wordseq", "gemini", "chatgpt", "jules"]
date: "2025-05-30T10:00:00Z" # Placeholder date - updated slightly
description: "A skeptical look at AI-driven development, and how 'vibe coding' helped and hindered the UI for my game, wordseq."
cover: "cover-vibe-coding.png" # Placeholder image
images: [] # Add any relevant new images here
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Vibe Coding: My Skeptical Foray into AI for Game UI (and Why I Still Reached for the Reins)"
slug: "vibe-coding-ai-skepticism-wordseq-ui" # Updated slug
type: "posts"
draft: true
---

Let's be honest, the current frenzy around AI feels a bit like a gold rush, and I've been watching from the sidelines with a healthy dose of skepticism. I mean, I have been using AI as a glorified search engine for a while now, but it seems like entire industries are rushing to rip out their foundations and replace them with something shiny and new, without fully understanding the long-term implications. It often feels like swapping out a foundation of concrete for what might turn out to be asbestos.

My general take? AI is neat for 'smarter' autocomplete, a helpful assistant for well-defined, small-scale tasks. But the idea of it independently architecting complex systems, or people using it to build things they fundamentally don't understand, strikes me as not only irresponsible but potentially dangerous. You can't just prompt your way to a robust, maintainable, and ethically sound application if you don't grasp the underlying principles.

However, despite this skepticism, the challenge of crafting a polished UI for my daily word game, [wordseq](https://wordseq.com), nudged me to dip a toe into these waters. After diving deep into the Go code for puzzle generation (which I wrote about [here](./wordseq)), it was time to build the face of the game. Could AI at least help me get the *look* and *feel* right, even if I didn't trust it with game puzzle generation?

So I started vibe coding. Vibe coding is less about expecting AI to write flawless, complex applications from scratch, and more about using it to rapidly get the *vibe* right. Getting AI to translate a mental image into a tangible starting point. My overall impression from this experiment? AI is indeed pretty decent at getting that raw material into the air. But give it too complex of a target, or ask it to make sound architectural decisions, and it will completely fall apart.

For `wordseq`'s frontend, built with React and TypeScript, I experimented primarily with **Gemini** and **ChatGPT**, and also gave **Jules** a try. My goal was to see if these tools could accelerate development, especially on the UI front where my own design skills are... not great. I'm a backend developer after all.

## Hitting the Ground Running

True to the "vibe coding" hypothesis, AI truly excelled at hitting the ground running with the initial UI. I could describe a component – "Create a responsive grid for letters with a subtle animation on swap," or "Design a clean modal for showing game stats" – and the AI would often spit out a surprisingly good first pass.

* **Aesthetic Boost:** Does the `wordseq` interface look way better than what I could have cobbled together on my own in the same timeframe? Absolutely. The AI tools were great at suggesting modern layouts, color palettes (even if I tweaked them), and generally making things look polished.
* **Boilerplate Reduction:** Getting basic component structures, event handlers, and initial styling in place was often much faster. It felt like having a junior dev who could quickly sketch out ideas.
* **Discovering New Techniques (Accidentally):** Sometimes the AI would use a CSS trick or a React pattern I wasn't familiar with, leading to a learning moment (after some head-scratching and debugging).

For instance, generating the initial letter tiles, the input area, and the visual feedback for correct or incorrect moves (like the wiggles and color changes I detailed in my previous post) got a significant head start from AI-generated snippets. It was like saying, "I want a tile that does this when clicked," and getting a foundational React component to then build upon. The "vibe" was achieved quickly.

## Off the rails

{{< diagram >}}
{{< image src="complexity.png" width="600px" class="center" alt="A graph showing initial AI productivity high, then sharply dropping off as complexity increases, while manual coding shows a steadier, more consistent progression." >}}
{{< /diagram >}}

{{< diagram >}}
{{< image src="herding-robots.png" width="600px" class="center" alt="A metaphor for trying to guide multiple AIs, perhaps illustrated as a person trying to herd chaotic, individualistic robots." >}}
{{< /diagram >}}


* **State Management Nightmares:** As `wordseq`'s game logic became more involved (tracking sequences, optimal paths, undo states), the AI's suggestions for managing this state in React often became convoluted, inefficient, or just plain wrong. The initial "vibe" might have looked good, but the underlying structure was often a house of cards.
* **Nuance Deafness:** Explaining very specific interactions or conditional rendering logic that depended on multiple factors felt like trying to teach calculus to a toddler. The AI would often miss key details or introduce subtle bugs.
* **The "Close, But No Cigar" (and often questionable) Code:** A lot of the generated code *looked* plausible on the surface, but wouldn't quite work, or worse, would work but in a completely nonsensical or unmaintainable way. This meant a significant amount of time was spent not just debugging, but completely re-working the code to be more reasonable. This is where the time-saving aspect started to look very shaky.

However, the honeymoon phase of rapid visual progress usually ended when complexity ramped up, or when I started to "dig under the vibes" to see what was actually holding it all together. Hint: it was insane nonsense, terrible technical decisions, brittle assumptions, leaky abstraction, etc.

{{< diagram >}}
{{< image src="slop.png" width="600px" class="center" alt="" >}}
{{< /diagram >}}

This led to a crucial realization, reinforcing my initial skepticism: **to effectively use AI to help me write React, I had to learn a good amount of React myself.** I needed to be able to understand the AI's suggestions, identify their (often severe) flaws, and ultimately guide it – or, more often, correct and heavily refactor its output – to arrive at sensible solutions. It was less "AI writing code for me" and more "me sifting through AI-generated chaos and leading it on a very tight leash."

Some AIs were better than others at grasping context or maintaining coherence over longer interactions, but the fundamental pattern was the same. The process of iterating with the AI, then "doing the work" to understand its often bizarre choices and refactor them, probably took a lot longer for complex features compared to me just knuckling down, deepening my React knowledge, and applying myself directly.

## Was "Vibe Coding" Worth It, Given the Skepticism?

So, the million-dollar question: was this AI-assisted "vibe coding" approach worth it for `wordseq`'s UI, especially viewed through a skeptical lens?

**Pros:**

* **Better Initial Aesthetics:** The game looks more professional than if I'd done all the UI from scratch under the same time pressure. The "vibe" was achieved.
* **Rapid Prototyping of Visuals:** Great for quickly getting visual ideas on the screen, a genuine flurry of "raw material."
* **Forced Learning (Deep Dives):** Ironically, having to unravel and fix the AI's "insane decisions" pushed me to understand React and web fundamentals at a deeper level than if I'd just muddled through on my own with simpler components.

**Cons:**

* **Time Sink on Deconstructing and Refactoring:** Debugging and then substantially re-working AI code for anything non-trivial was often slow and frustrating. The "raw material" often needed heavy processing.
* **Illusion of Speed, Reality of Rework:** What felt like quick progress initially often got bogged down in extensive refactoring to make the code maintainable and logical.
* **Still Needed to Learn (and Then Some):** You can't escape understanding the underlying technology if you want a robust application. In fact, you need to understand it even *better* to effectively manage and correct AI output.

For just hitting the ground running with UI, getting that initial "vibe" down, AI seems excellent. It gave me a much stronger visual and structural base to build upon. But for the intricate wiring, the complex state, sound architecture, and the specific game logic interactions, direct human coding, understanding, and significant rework were irreplaceable.

## Reflections: Skepticism Validated, with Nuance

This journey with AI-assisted UI development has been an educational one. It's shown me that these tools can be powerful accelerators for *certain parts* of the development process – specifically, generating that initial visual "vibe" and a heap of raw material.

However, my core skepticism remains largely intact, if not reinforced. They are not (yet) a replacement for deep technical knowledge, sound architectural judgment, and meticulous problem-solving skills. Relying on them blindly for anything beyond superficial tasks feels like a gamble. The need to "dig under the vibes" and fundamentally rework the AI's often questionable decisions was a consistent theme.

Moving forward with `wordseq`, I feel more confident in my own React abilities, partly thanks to having to "teach" and extensively correct the AI. I'm now better equipped to tackle more complex UI features, like the "infinite mode" I'm planning, or even potentially a Danish version of the game – and I'll be doing so with a much clearer understanding of when and how (and how *little*) to involve AI.

The "vibe" is set, and the foundation is much prettier thanks to some initial AI collaboration. Now, the serious, human-led craftsmanship continues, building on that aesthetic start but ensuring the structure underneath is solid and sensible.

---
What are your experiences using AI for coding? Are you all-in on the hype, a fellow skeptic, or somewhere in between? Has it been a magical accelerator, a frustrating time-sink, or a source of "insane decisions" you've had to fix? I'd love to hear your thoughts!

* **Play wordseq daily:** [wordseq.com](https://wordseq.com "wordseq")
* Find me on [Blue Sky](https://bsky.app/profile/kmcd.dev "kmcd.dev on bluesky") or [Mastodon](https://infosec.exchange/@sudorandom "@sudorandom on infosec.exchange, mastodon")!

Thanks for reading!