---
categories: ["article"]
tags: ["gamedev", "react", "wordseq", "typescript", "go", "games"]
date: "2025-05-15T10:00:00Z"
description: "Taking a peek under the covers of making a daily puzzle games"
cover: "cover.png"
images: ["/posts/wordseq/wordseq.svg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "I made a daily word game"
slug: "wordseq"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/wordseq/
draft: true
---

Ever stare at a word so long it stops looking like a word? Like naming a variable ‘data’ for the 8th time and suddenly wondering what ‘data’ even means?

That effect is called [semantic satiation](https://en.wikipedia.org/wiki/Semantic_satiation). I made a game that dives headfirst into that theme.

In `wordseq`, you swap letters to form words, but the deeper you go, the more the grid feels like a linguistic fever dream. One moment you're proud to find "plop", the next you're doubting if "plop" was ever a real word or just a sound effect from a comic book. You win by dragging yourself back to meaning. Enjoy the victory while you can, because there's a new puzzle tomorrow.

I'm excited to share a look behind the scenes of my new daily word puzzle game, `wordseq`. If you haven't tried it yet, you can [play the latest puzzle here!](https://wordseq.com). `wordseq` is a game where you swap adjacent letters to form new words, aiming to find the longest possible sequence. Here's what it looks like to play the game (by a super unrealistically fast player).

{{< diagram >}}
{{< image src="gameplay.gif" class="center" alt="wordseq gameplay" >}}
{{< /diagram >}}

One of the most challenging and rewarding aspects of building `wordseq` was developing the system that generates the daily puzzles. My goal was to create levels that are not just solvable, but also consistently engaging, fair, and offer that satisfying "aha!" moment where the answer you've been looking for hits you in the face. Conversely, I want to avoid the "Huh, is this actually the solution?" moments.

In this post, we'll explore the intricate process of generating these daily puzzles. We'll cover everything from initial grid creation and word validation to ensuring puzzles are solvable and use interesting vocabulary, leveraging techniques like concurrency in Go and smart dictionary management.

{{< diagram >}}
<a href="http://wordseq.com" target="_blank">{{< image src="wordseq.svg" height="150px" class="center" alt="wordseq logo" >}}</a>
{{< /diagram >}}

## Core Gameplay Mechanics

Before we dive into generation, here's a quick overview of how to play `wordseq`:
* **The Core Mechanic:** Players swap any two letters that are directly next to each other (horizontally or vertically).
* **Word Formation:** Each swap *must* result in at least one new word spanning the length of a row (this length can vary by difficulty). This new word is formed by the entire row or column where the swap occurred.
* **The Goal:** The objective is to find the longest possible sequence of valid words by making these swaps. This is what we call the "optimal path."
* **Difficulty Levels:** `wordseq` offers 'normal', 'hard', and 'impossible' modes, which influence parameters like grid size, word length, and the complexity of the solution paths the generator aims for.

### Movement
Here is what the movement looks like.

Bad move, results in no new words. Don't you like the little wiggle? It's the small things that really shape the experience:
{{< diagram >}}
{{< image src="bad-move.gif" width="400px" class="center" alt="letters wiggling and turning red" >}}
{{< /diagram >}}

Non-optimal move, a move that sets you up to fail. You may think that you made a clever move and discovered a word just slightly askew, but there are many times where there's another word that you need find first to best set up the longest sequence. You will be greeted by a not-as-scary orange color. Usually you will want to undo your move and look for another:
{{< diagram >}}
{{< image src="non-optimal-move.gif" width="400px" class="center" alt="letters swapping and a newly completed word turning orange" >}}
{{< /diagram >}}

And, of course, the optimal move, the move that finds a new word but also sets you up for the most followup words.
{{< diagram >}}
{{< image src="optimal-move.gif" width="400px" class="center" alt="letters swapping and a newly completed word turning green" >}}
{{< /diagram >}}

The challenge with this game isn't only finding/completing words, but also setting yourself up for the next one. Don't worry, there's no downside for finding the 'wrong' paths.

## The Tech Stack

The level generation is a computationally intensive process, so I opted to build the generation script in [**Go**](https://go.dev/), primarily for its excellent performance and built-in support for concurrency, but also it's the language that I know best right now. This script outputs JSON data for each level. There will be more on the level format later.

The frontend, where you play `wordseq`, is built with:
* [**React**](https://react.dev/) (using Vite for a fast development experience)
* [**TypeScript**](https://www.typescriptlang.org/) for type safety
* [**Tailwind CSS**](https://tailwindcss.com/) for styling

The JSON level data from the Go script is fetched by the React frontend to power each day's puzzle.

## Generating Engaging Levels

Crafting a "good" `wordseq` puzzle involves several steps and considerations. We want puzzles that aren't too easy, don't rely on super obscure words, and have a satisfying solution depth. Here's how the generator tackles this:

### Step 1: Initial Grid Generation

* **It starts with Randomness:** The process starts by generating a grid (e.g., 5x5) filled with random letters. However, just picking letters completely at random can lead to grids full of very rare letters (like Q, Z, X, J) which makes forming words difficult or leads to trivial "write-offs" where those letters are ignored. To combat this, the generator uses **approximate letter frequency** (based on English language letter usage) to populate the grid. This creates a more balanced and natural distribution of letters, similar to what you might find in a Scrabble bag.

* **Initial Validation:** Before any moves are possible, the generator verifies that the starting grid contains no valid words matching the puzzle's target length. This ensures the puzzle begins only with the player's first action and avoids potential confusion where the initial state might seem like part of the solution, or where swapping back to the original layout could be misinterpreted as finding a word.

This check involves looking for rows in each row and column, like this:

{{< d2 width="500px" >}}
style: {
  fill: transparent
}

classes: {
  SELECTED: {style.fill: "#2b9464"}
}

Grid0: {
  label: ""
  grid-gap: 1 # Minimal gap between cells
  grid-rows: 4

  # Row 0
  "0,0": r
  "0,1": i
  "0,2": r
  "0,3": n
  # Row 1
  "1,0": y
  "1,1": r
  "1,2": s
  "1,3": m
  # Row 2
  "2,0": t
  "2,1": r
  "2,2": o
  "2,3": i
  # Row 3
  "3,0": e
  "3,1": v
  "3,2": d
  "3,3": o
}

Grid0."0,0" -> Grid0."0,3": "" {
  style: {
    stroke: "#5555FF"
    stroke-width: 2
  }
  target-arrowhead: {
    shape: arrow
  }
}

Grid0."1,0" -> Grid0."1,3": "" {
  style: {
    stroke: "#5555FF"
    stroke-width: 2
  }
  target-arrowhead: {
    shape: arrow
  }
}

Grid0."2,0" -> Grid0."2,3": "" {
  style: {
    stroke: "#5555FF"
    stroke-width: 2
  }
  target-arrowhead: {
    shape: arrow
  }
}

Grid0."3,0" -> Grid0."3,3": "" {
  style: {
    stroke: "#5555FF"
    stroke-width: 2
  }
  target-arrowhead: {
    shape: arrow
  }
}

Grid1: {
  label: ""
  grid-gap: 1 # Minimal gap between cells
  grid-rows: 4

  # Row 0
  "0,0": r
  "0,1": i
  "0,2": r
  "0,3": n
  # Row 1
  "1,0": y
  "1,1": r
  "1,2": s
  "1,3": m
  # Row 2
  "2,0": t
  "2,1": r
  "2,2": o
  "2,3": i
  # Row 3
  "3,0": e
  "3,1": v
  "3,2": d
  "3,3": o
}

Grid1."0,0" -> Grid1."3,0": "" {
  style: {
    stroke: "#FF5555"
    stroke-width: 2
  }
  target-arrowhead: {
    shape: arrow
  }
}

Grid1."0,1" -> Grid1."3,1": "" {
  style: {
    stroke: "#FF5555"
    stroke-width: 2
  }
  target-arrowhead: {
    shape: arrow
  }
}

Grid1."0,2" -> Grid1."3,2": "" {
  style: {
    stroke: "#FF5555"
    stroke-width: 2
  }
  target-arrowhead: {
    shape: arrow
  }
}

Grid1."0,3" -> Grid1."3,3": "" {
  style: {
    stroke: "#FF5555"
    stroke-width: 2
  }
  target-arrowhead: {
    shape: arrow
  }
}

{{< /d2 >}}

It is important to note that words spent going upwards and to the left aren't seen as words. The game would be far too complex if that were the case.

### Step 2: Building the Exploration Tree - Finding All Possibilities

This is where the magic (and a lot of computation) happens. For a given initial grid that passes validation, the generator explores all possible valid game sequences:
* **Iteration:** From the current grid state, it identifies every possible adjacent letter swap.

{{< d2 >}}
direction: down
style: {
    fill: transparent
}

# --- Class for grid states that are not expanded further ---
classes: {
  NotExpandedState: {
    style: {
      stroke: red
      stroke-width: 2
      stroke-dash: 3
      opacity: 0.6
    }
  }
}

S0: {
  shape: square
  label: D O S G
  width: 60
  height: 60
  style: {
    font-size: 14
  }
}

S0_M2: {
  class: NotExpandedState
  shape: square
  label: O D S G
  width: 60
  height: 60
  style: {
    font-size: 14
  }
}

# Move 3 (Swap S0_R0C0, S0_R1C0)
S0_M3: {
  class: NotExpandedState
  shape: square
  label: D S O G
  width: 60
  height: 60
  style: {
    font-size: 14
  }
}

S0_M1: {
  shape: square
  label: D O G S
  width: 60
  height: 60
  style: {
    font-size: 14
  }
}

# --- Level 1 Edges (Unlabeled) ---
S0 -> S0_M1
S0 -> S0_M2
S0 -> S0_M3

{{< /d2 >}}

* **Validating Swaps (The Big Dictionary):** For each potential swap, it temporarily performs the swap and then checks if the new grid configuration forms at least one new word that spans the length of the grid (horizontally or vertically). This check uses a **large, comprehensive dictionary**. The reason for this large dictionary is critical for player experience: if a player sees a word on the grid and forms it, the game *must* recognize it, even if it's a bit uncommon. It's incredibly frustrating for a game to not accept a word you know is valid!

{{< d2 width="700px" >}}
direction: right
style: {
    fill: transparent
}

classes: {
  SELECTED: {style.fill: "#2b9464"}
}

# Grid representation
Grid0: {
  label: Initial State
  grid-gap: 1 # Minimal gap between cells
  grid-rows: 4

  # Row 0
  "0,0": r
  "0,1": i
  "0,2": r
  "0,3": n
  # Row 1
  "1,0": y
  "1,1": r
  "1,2": s
  "1,3": m
  # Row 2
  "2,0": t
  "2,1": r
  "2,2": o {class: SELECTED}
  "2,3": i {class: SELECTED}
  # Row 3
  "3,0": e
  "3,1": v
  "3,2": d
  "3,3": o
}

# Grid representation
Grid1: {
  label: "found: trio"
  grid-gap: 1
  grid-rows: 4

  # Row 0
  "1-0,0": r
  "1-0,1": i
  "1-0,2": r
  "1-0,3": n
  # Row 1
  "1-1,0": y
  "1-1,1": r
  "1-1,2": s
  "1-1,3": m {class: SELECTED}
  # Row 2 (Swapped)
  "1-2,0": t
  "1-2,1": r
  "1-2,2": i
  "1-2,3": o {class: SELECTED}
  # Swapped from i
  # Row 3
  "1-3,0": e
  "1-3,1": v
  "1-3,2": d
  "1-3,3": o
}

# Grid representation
Grid2: {
  label: "found: trim"
  grid-gap: 1
  grid-rows: 4

  # Row 0
  "2-0,0": r
  "2-0,1": i
  "2-0,2": r
  "2-0,3": n
  # Row 1 (Swapped)
  "2-1,0": y
  "2-1,1": r
  "2-1,2": s
  "2-1,3": o
  # Swapped from m
  # Row 2 (Swapped)
  "2-2,0": t
  "2-2,1": r
  "2-2,2": i
  "2-2,3": m
  # Swapped from o
  # Row 3
  "2-3,0": e
  "2-3,1": v
  "2-3,2": d
  "2-3,3": o
}

# --- Transitions (Moves) ---

# Edge representing the first move found
# Connect the container nodes, not the internal grids
Grid0 -> Grid1

# Edge representing the second (nested) move found
Grid1 -> Grid2

{{< /d2 >}}

* **Recursive Exploration:** If a swap is valid (i.e., forms at least one new word), this new grid state and the move that led to it become a new "node" in an exploration tree. The process then recursively explores all valid swaps from *this* new state, and so on. This continues until no more valid moves can be made from a state, or a predefined maximum search depth is reached.

{{< d2 >}}
direction: down
style: {
    fill: transparent
}

# --- Class for grid states that are not expanded further ---
classes: {
  NotExpandedState: {
    style: {
      stroke: red
      stroke-width: 2
      stroke-dash: 3
      opacity: 0.6
    }
  }
}

S0: {
  shape: square
  label: D O S G
  width: 60
  height: 60
  style: {
    font-size: 14
  }
}

S0_M2: {
  class: NotExpandedState
  shape: square
  label: O D S G
  width: 60
  height: 60
  style: {
    font-size: 14
  }
}

# Move 3 (Swap S0_R0C0, S0_R1C0)
S0_M3: {
  class: NotExpandedState
  shape: square
  label: D S O G
  width: 60
  height: 60
  style: {
    font-size: 14
  }
}

S0_M1: {
  shape: square
  label: D O G S
  width: 60
  height: 60
  style: {
    font-size: 14
  }
}

# --- Level 1 Edges (Unlabeled) ---
S0 -> S0_M1
S0 -> S0_M2
S0 -> S0_M3

# Move 1 from S0_M1 (Swap R0C0, R0C1)
S0_M1_M1: {
  class: NotExpandedState
  shape: square
  label: O D G S
  width: 60
  height: 60
  style: {
    font-size: 14
  }
}

# Move 2 from S0_M1 (Swap R1C0, R1C1)
S0_M1_M2: {
  class: NotExpandedState
  shape: square
  label: D G O S
  width: 60
  height: 60
  style: {
    font-size: 14
  }
}

# Move 3 from S0_M1 (Swap R0C0, R1C0)
S0_M1_M3: {
  class: NotExpandedState
  shape: square
  label: D O S G
  width: 60
  height: 60
  style: {
    font-size: 14
  }
}

# --- Level 2 Edges (Unlabeled) ---
S0_M1 -> S0_M1_M1
S0_M1 -> S0_M1_M2
S0_M1 -> S0_M1_M3
{{< /d2 >}}

### Step 3: Analyzing the Tree

Once an exploration tree (or a significant portion of it) is built for an initial grid:
* **Calculating `maxDepthReached`:** For every node in the tree (each representing a game state reached by a valid move), the generator calculates the `maxDepthReached`. This value signifies the length of the longest possible sequence of *further* valid moves that can be made starting from that particular node.
* **Determining Overall Puzzle Depth:** The overall "solution length" for the puzzle (which the frontend knows as `gameData.maxDepthReached`) is the highest `maxDepthReached` value found among the *initial* possible moves from the starting grid. This represents the longest chain of words the player can achieve.
* **Filtering Puzzles by Length:** We use `RequiredMinTurns` and `RequiredMaxTurns` (configurable parameters for the generator) to filter these potential puzzles. Grids that lead to solutions that are too short (not challenging enough) or excessively long (potentially too tedious) are discarded.

### Step 4: Word Dictionaries

This is where the two-dictionary approach comes into play:
* **The Large Dictionary (for Gameplay Logic):** As mentioned, this is used during the tree exploration step to ensure any valid word a player might form is recognized. At the time of writing, the large dictionary has 129,493 words in it.
* **The Smaller, Curated Dictionary (for Puzzle Quality):** After a potential puzzle is generated, the generator collects *all unique words* that appear in *any* valid path within that entire tree. These words are then checked against this smaller, more "usable" or common dictionary. If the given puzzle uses words that are too obscure or too obscene then the puzzle is discarded entirely. *Many* potential puzzles are discarded from this check. This is a key step to ensure the final puzzles feel fair and use vocabulary that players are likely to know or find satisfying to discover. At the time of writing, this dictionary has 6,956 words - an absolutely tiny amount compared to the large one.
* **Controlling Word Variety:** Another filter is `MaxUniqueWords`. If a potential puzzle contains too many distinct words across all its possible solution paths, it might be too chaotic. This parameter helps keep the word set focused.

### Step 5: Concurrency

Generating a single grid, building its exploration tree, and then validating it against all these criteria can take time. To find enough high-quality puzzles for daily release, I needed to process *many* initial random grids. Doing this sequentially was fairly slow, so I ended up splitting up the work by using go routines. This greatly improved the throughput. It's honestly so nice to have a highly parallelizable, CPU-bound problem to work with. It's a nice break from the world of the web where I/O is typically the bottleneck.

### Step 6: Outputting the Puzzle Data

Once a grid and its exploration tree pass all the filters, it's deemed a "good" puzzle. The generator then outputs JSON with the details about a puzzle. Here's a trivial example with a single possible move for illustration purposes:

```json
{
  "initialGrid": [
    ["i","s","w","y"],
    ["e","v","a","p"],
    ["r","a","b","a"],
    ["h","l","l","a"]
  ],
  "wordLength": 4,
  "maxDepthReached": 1,
  "explorationTree": [
    {
      "move": {
        "from": [1, 0],
        "to": [1, 1]
      },
      "wordsFormed": ["seal"],
      "maxDepthReached": 0,
      "nextMoves": []
    }
  ]
}
```

* `initialGrid`: The starting letter configuration.
* `wordLength`: The target word length for this puzzle.
* `maxDepthReached`: The length of the optimal solution path(s).
* `explorationTree`: The full tree structure, containing all valid moves, the words they form, and the `maxDepthReached` from each node.

This JSON file is then what the React frontend loads each day to power the game you play. This is what an extremely simple grid looks like, with only one valid move from the initial state.

## Lessons Learned

* **Dictionaries are Key:** The quality and scope of your dictionaries profoundly impact both gameplay fairness and puzzle quality. The two-dictionary system was vital.
* **Iterative Refinement:** Level generation isn't something you get right on the first try. It requires constant tweaking of parameters, testing, and playing the generated levels yourself.
* **Concurrency is Your Friend:** For computationally heavy tasks like this, leveraging concurrency (like Go offers) is almost essential for practical generation times.
* **Define "Good" as early as you can:** Having clear criteria for what constitutes a good puzzle (solvable, right length, good words) helps guide the entire generation logic. I was lucky to have my wife, who's a daily word game solver as a test user. Her feedback was (and is) invaluable. Making a game is an iterative process. A lot of times you legitimately don't know if the game you're making is fun. Who knows, maybe I made a game only my wife loves. And honestly? If she’s the only one who loves it, that still feels like a win. (But I do hope you’ll enjoy it too.)
* **Tools to help testing are worth it:** I didn't mention this yet, but I have some tooling now to make testing a lot easier. Not only do I get clear display of the solutions (cheap mode!) but I can push a button and have the game played for me. This has helped a lot when I needed to get to certain game states quickly.
* **Data structures and Game Design:** I really love how this game relies heavily on computer science fundamentals and data structures. At the end of the day this is just a non-obvious tree traversal with a UI on top of it.

## What's Next for wordseq's Puzzles?

I think there's a lot of additional things I can use to judge the difficulty of a given grid. I could judge how often completed words switch rows for the next word, or (if possible), how often it alternates from rows to columns. Often, the puzzle will be easier when it's only one row that is constantly updating. Changing one letter of an existing word is a lot easier for a mind to handle than re-evaluating the whole board every turn.

There's always more refinement that can be done with the dictionary. The small dictionary’s pretty solid now, though an occasional spicy word still sneaks through. Oops.

On the frontend side, I feel like I'm getting close to a decent interface. I want to build an "infinite mode" where you can play random levels (identified by an ID so you can link them and go back to those puzzles later). I actually think this wouldn't be too crazy with the way the components are laid out now.

I'm also contemplating how hard it might be to implement a Danish version as I am learning Danish.

## Conclusion

Building the level generator for `wordseq` has been a fascinating journey into algorithms, data structures, and the subtle art of puzzle design. It's a complex system, but seeing it produce fun and challenging puzzles each day is incredibly rewarding.

I hope this peek behind the curtain was interesting!
* **Play wordseq daily:** [wordseq.com](https://wordseq.com "wordseq")
* I'd love to hear your **feedback** on the puzzles or any thoughts you have on level generation. Drop a comment below or find me on [Blue Sky](https://bsky.app/profile/kmcd.dev "kmcd.dev on bluesky") or [Mastodon](https://infosec.exchange/@sudorandom "@sudorandom on infosec.exchange, mastodon")!

Thanks for reading!
