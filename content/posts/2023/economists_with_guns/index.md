---
categories: ["article", "project"]
tags: ["eve-online", "gaming", "evepraisal"]
date: "2023-08-11"
description: "This is a story about how I came to create evepraisal.com, a popular tool for Eve Online that is commonly used to price check cargo scans, contracts, EFT fittings, assets, and more. If none of that means anything to you that's totally fine! I barely know, too."
images: ["/posts/evepraisal.com/thumbnail.png"]
featured: "thumbnail.png"
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Economists with (virtual) Guns"
slug: "economists-with-guns"
type: "posts"
devtoSkip: true
mastodonID: "112277295705206443"
---

> Note: This post was originally posted in 2015. I am re-posting it with some small edits. Updates are at the bottom.

This is a story about how I came to create evepraisal.com, a popular tool for Eve Online that is commonly used to price check cargo scans, contracts, EFT fittings, assets, and more. If none of that means anything to you that's totally fine! I barely know, too.

Eve Online has always been an interesting game to me. For the uninitiated, Eve Online is a space-themed single-sharded MMO game that boasts an incredibly realistic economy and allows players to take high-stakes risks. This is the game that has had several extremely large fights over player-controlled territories involving thousands of people. It’s a game where politics, espionage and propaganda are effective weapons against your enemy.

### Burn Jita

2 years ago, during a large player-made event called “Burn Jita”, an in-game alliance named Goonswarm and various other null-sec entities sought to blockade the biggest market hub in the game in hopes that the established trading center of Eve Online would be significantly impacted. I happened to be in an alliance comprising “various other null-sec entities” so I participated in the event. How could I not? Here’s how the scheme worked:

- Designated people would cargo scan ships passing through nearby systems headed toward Jita. The information would be relayed through in-game and out-of-game channels.
- A fleet of ships would be waiting to strike juicy targets foolish enough to enter or leave the trade hub with valuable cargo. The ships they were flying were cheap and could deal a lot of instant damage.
- Once a transport ship with a significant amount of goods entered the system the fleet would strike. Jita is located in high sec which means every attacking ship would pretty quickly be destroyed by the computer-controlled police force, but the damage output of the entire fleet assured that the transport ship would turn into a pile of wreckage and loot.
- Rinse; repeat.

Now take a second to appreciate the fact that several of these fleets were operating at any given time and the ships used to do these attacks were provided, free of charge, by each participating alliance.

My role in the process above was being one of the people who did step 1. I did a fair amount of ganking with the fleet but what I really enjoyed was pointing to a person and then waiting 10 minutes to see that his ship turned into a mangled wreck and his goods (or what’s left of them) were stolen to aid the further destruction of people like him. It was my (and my counterpart’s) actions that started the chain of events leading to the death of an enterprising player who paid no attention to player events. The act of firing space bullets was a detail.

### Necessity is the mother of invention

After a few hours of scanning the cargo of every ship in sight, I realized that many freighters were stuffing their cargo hold with lots of useless junk to hide the true value of the cargo that they were carrying. This was a problem. Deciding who to attack was a time-sensitive task and not having a reliable cost estimate of what the ship contained made things harder than they had to be.

That’s where something clicked in my mind and over the next 3 hours I made a web-based tool that completely solved the ambiguity of this entire process and changed ganking into a legitimate economically-driven career that boasts cost/benefit analysis similar to (yet more exciting than) most of the other more reputable careers in Eve Online. That’s a big claim, but I don’t think I am far off.

### Let me explain…

When you cargo-scan a potential target’s cargo hold, you get a list of all its contents. Up until recently, all you could do was look at the list and that was it. However, CCP recently added the ability to copy/paste from many different places in the Eve Online UI including the cargo scan result window. Cool, I thought. I could have users to give me a list of all the items in the potential target’s hanger. Now I just needed a way to calculate the prices. Eve Online has an amazing developer community. There were (and are) existing websites that had daily database dumps of aggregate pricing data for every market item in the game. I downloaded one of the database dumps and used that data as my price reference. After that, it was a matter of making a small website where people could give it cargo scan results and it would give back a reasonably accurate guess of the cargo’s value in less than a second. These estimates could be much better than what a human could do manually at the same time.

I randomly linked the website that I had just created in one of the fleet channels and it was a short time later that someone in Goonswarm broadcasted the link to the Goonswarm jabber server. I instantly got a flood of interested people trying out the tool and, to my surprise, the site held up fine. Later, I added the ability to share results so that the many scanners wouldn’t have to communicate the estimate they’re seeing from my website which further streamlined the process. In hindsight, what seemed like a small feature that was fairly easy to implement turned out to be the biggest selling point. The immediate feedback I got from people was invaluable and shaped my thoughts on the problem. To this day I’ve never had a bad experience asking people (even those who didn’t like the tool) what could be better.

The Burn Jita event continued for a couple more days and instead of cargo scanning ships, I was working on improving the tool that others used to find targets to gank more efficiently. The rest is history.

### Update (2015)

The website slowly evolved into what is now known as Evepraisal. I added support for many different formats and switched from using price data from a static database dump to using more frequently updated sources like Eve-Central and Eve-marketdata.com. I’ve heard from several people who are career gankers in Eve who leverage Evepraisal to be profitable.

I am still waiting for the day that CCP makes a suitable replacement to my tool, but until then I’ll still be slowly improving on the tool that’s become a ganker’s best friend and a scammer’s worst nightmare.

### Update (2020)

It’s sad to say but I’ve let Evepraisal kind of stagnate for a while. It sits in a weird position for me of being an extremely neat facet of my development as a programmer but also it weird to work on because I do not play Eve Online anymore. Since 2015, I’ve made several large changes to Evepraisal that were almost invisible to users. I rewrote the entire app in a different programming language. I sourced the market data from CCP’s own API (that didn’t exist before), which is very lucky because Eve-central eventually completely dissipated. I sourced type information from CCP’s static data dump. After making those two changes the amount of issues Evepraisal has had is extremely small. The requests for new features obviously come in, but I wanted the tool to be targeted at doing one specific thing well. For a while, I hoped that other tools would meet or surpass Evepraisal’s functionality. It took a long time but I believe there are several really good alternatives.

However, there have been two major issues. First, the Google Sheets addon that I made and published became unpublished due to Google completely changing how you administer Google Sheets add-ons. They also required a lot more from developers to improve their storefront… But it was just too many hoops for me to deal with. I attempted but I hit one roadblock too many and gave up. The second issue wasn’t my fault either. To make things more consistent, CCP started changing the names of several items. In the past when this happens it isn’t a big deal since the static data dump that CCP provides always had updates. But ever since the beginning of 2020, CCP has not kept the static data up-to-date with the name changes. It boggles my mind why they would do it this way. I’ve checked and there’s no reference to the new names anywhere in that package of static data files. So, alas, I resorted to actually changing Evepraisal’s code to support type aliases and hard-coded that list of aliases.

So here I am. A lot of stuff is happening in my life and I still don’t want to spend too much time on a 7-year-old project. I do have many, many ideas of how I might want to present market data. Eve Online is still an amazing MMO because of its open APIs and willingness to support third-party developers. And it’s the only MMO that regularly has actual large-scale player-driven events. With all of that said I plan to keep Evepraisal running for as long as people use it.

### Update (2023)
I've decided to shut it down. [See more about the reasoning here](/posts/goodbye-evepraisal/) but you can kind of see my attitude 3 years ago.
