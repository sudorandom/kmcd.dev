---
categories: ["article"]
tags: []
date: "2025-07-22T10:00:00Z"
description: ""
cover: "cover.png"
images: ["/posts/morse/cover.png"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Morse Code"
slug: "morse"
type: "posts"
draft: true
---

I'm continuing my journey into discovering more about older technology by making silly games. This time, I made a website that allows you to test your speed skills in the most useful skill ever! Morse code! Now you can test your skills with writing and interpreting morse code! And it even gives you your (likely abysmal) words per minute (wpm). (Plus, it's great practice for [Keep Talking and Nobody Explodes](https://keeptalkinggame.com/))

Head over to [https://morse.kmcd.dev](https://morse.kmcd.dev) and try the [Morse Speed Typer](https://morse.kmcd.dev) today! And if you need to brush up on your skills, check out the [learning page](https://morse.kmcd.dev/learn).

---

## What is Morse Code?

Morse code is a character encoding system used in telecommunication that represents letters, numbers, and punctuation marks as sequences of short and long signals. These signals, often called **dots** (or "dits") and **dashes** (or "dahs"), can be transmitted in various ways, such as through sound, light flashes, or electrical pulses.

Developed in the 1830s by Samuel Morse and Alfred Vail, it revolutionized long-distance communication. Before we had fiber optics and satellites, we had the simple, elegant language of dits and dahs clicking away across telegraph wires.

---

## The Rhythm of the Code: Timing ⏱️

Morse code isn't just about the dots and dashes; it's about the silence in between. The timing is crucial for distinguishing letters and words. The entire system is based on the length of a single dot.

* A **dot** (dit) is the basic time unit: 1 unit long.
* A **dash** (dah) is three times longer than a dot: 3 units long.
* The **space** between parts of the same letter (e.g., the gap between the `.` and `-` in 'A') is 1 unit long.
* The **space** between letters in a word is 3 units long.
* The **space** between words is 7 units long.

This precise rhythm is what allows a trained operator to "read" the code by ear.

---

## The Original Binary Code

Long before computers used 1s and 0s, Morse code was encoding information using just two states: a short signal and a long signal (or, more fundamentally, "signal on" and "signal off"). In this sense, Morse code is one of the earliest forms of a **binary code**.

Just as modern computers use a standardized system like ASCII or Unicode to map binary digits to characters, Morse code provides a map from dits and dahs to the letters of the alphabet. It's a brilliant, early example of how complex information, human language, can be broken down into simple, transmittable units.

---

## Cool Facts from the World of Morse 🌍

Morse code has a rich history filled with interesting trivia and conventions.

* **SOS is not an acronym.** The famous distress signal, `...---...`, was chosen because its pattern is simple and unmistakable. The continuous sequence of three dots, three dashes, and three dots is easy to recognize even through heavy static or interference. It doesn't stand for "Save Our Ship" or "Save Our Souls," though those are memorable mnemonics!

    {{< morse-viz "... --- ..." "SOS" >}}

* **K (Invitation to Transmit).** In Morse conversations, sending a single letter 'K' (`-.-`) is an invitation for the other person to start transmitting. It's the equivalent of saying "over" or "your turn."

    {{< morse-viz "_._" "K" >}}

* **73 and 88.** The world of ham radio, which heavily uses Morse code, developed its own numeric shorthand. `73` means "Best regards," and `88` means "Love and kisses," typically used when signing off with a close friend or partner.

    {{< morse-viz "--... ...--" "73" >}}
    {{< morse-viz "---.. ---.." "88" >}}

* **What hath God wrought!** On May 24, 1844, this was the first official message sent by Samuel Morse on the telegraph line between Washington, D.C., and Baltimore. The phrase, from the Book of Numbers, was suggested by Annie Ellsworth, the daughter of a friend. The message in Morse is:

    {{< morse-viz ".-- .... .- -" "WHAT" >}}
    {{< morse-viz ".... .- - ...." "HATH" >}}
    {{< morse-viz "--. --- -.." "GOD" >}}
    {{< morse-viz ".-- .-. --- ..- --. .... -" "WROUGHT" >}}

---

## Conquering the Ocean: The First Global Connection

It’s easy to take for granted that we can send a message across the world in an instant. But before the 1860s, the fastest way to get information across the Atlantic was on a ship. A message sent from London to New York would take at least 10 days to arrive, and a reply would take just as long. A simple business transaction or a piece of breaking news could take nearly a month to cross the ocean and return. The world was vast, and the oceans were a barrier to communication.

The transatlantic telegraph cable changed everything. After a failed attempt in 1858, the first *lasting* connection was established in 1866. Laying a single, insulated copper wire across more than 3,000 kilometers of the treacherous North Atlantic seabed was one of the greatest engineering feats of the 19th century.

Suddenly, the 10-day journey of a message became a matter of minutes. The world shrank in a way that was previously unimaginable. For the first time, continents could converse in near real-time. News, financial data, and personal messages that once traveled at the speed of a steamship now traveled at the speed of electricity. It was the birth of our global network.

That single wire, painstakingly laid over 150 years ago, was the ancestor of the incredible web of cables that powers our internet today. Instead of one copper wire carrying a few words per minute in Morse code, we now have hundreds of undersea fiber-optic cables carrying terabits of data every second. They are the backbone of our modern world, transmitting everything from video calls to this very article.

To see the stunning evolution from that first transatlantic wire to the dense global network of today, **explore this [interactive map of modern undersea fiber-optic cables](https://map.kmcd.dev/)**. It’s the direct legacy of the dits and dahs that first conquered the ocean.

---

## The Enduring Echo of Dits and Dahs

From revolutionizing global communication on telegraph wires to its enduring legacy in maritime and amateur radio, Morse code is far more than a historical artifact. It stands as a powerful testament to human ingenuity. It was the original binary that paved the way for the digital world we navigate today. The simple, rhythmic language of dits and dahs demonstrates a foundational principle: that complex information can be broken down, transmitted, and understood through a simple, elegant system.

So while it may no longer be the backbone of our global network, learning Morse code offers a unique connection to the history of technology. It’s a chance to appreciate the rhythm and precision that started it all.

Ready to see how you measure up? Tap into history and test your skills on the [Morse Speed Typer](https://morse.kmcd.dev). You might just find a new appreciation for the simple `... --- ...` that connected the world.