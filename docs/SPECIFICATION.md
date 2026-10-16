# Specification

## Page Flow

Navbar on all pages with the profile in the top right with a logout dropdown and colour scheme switcher.

### Main Library Page

Shows the sidebar with the categories. The right side shows a gallery, list, or condensed list of
all games in that category, with the ability to sort (including random sort), filter, and search
the list.

## Features

- Log in with Steam as the sole method of authentication.
- On-demand manual sync of a games library including played hours and review rating.
- Integration into IGDB's time to beat.
- Database keeps track of categories (see below) including computed categories.
- Random sort option for game library views.
- Library view "heatmap" option that changes the opacity or colour of titles
- Reviews feature with different categories (see below).
- Docker Compose file for easy self-hosting.
- Game store page in an iframe when inspecting game details.
- Any SDK that must be created to work with required third parties.
- Bulk editing of games via "select" + action _and_ mass actions like "mark all backlog games with
  > 1h as in progress."
- Dark/light/system mode.

### Stretch Goals

- Automatic scheduled game library sync.
- Graceful handling of private or missing accounts and other errors.
- Data export and import.
- GDPR account deletion.

### Possible Features

- Wishlist management.
- Hosted version supported with advertisements.
- REST/GraphQL API.

### Non-Features

- Any social features.
- A "profile" of any kind.
- AI-powered features. If users really want to have this, they'd need to implement it through the
  API.

## Categories

- Done: games that are marked "complete" for one reason or another.
  - Dropped: games that were not worth your time.
  - One-shot: games that were completed but not going to be played again.
  - Replay: games that were completed but might be played again, or staple games like multiplayer
    games (Valorant,
    League of Legends, MMOs, etc.).
  - Uncategorised: other/undecided.
- In Progress: games that are being actively played.
- Backlog: a list of games.
  - Acclaimed: games that have excellent reviews.
  - Short: games that have below a single standard deviation of normal playtime.
  - Obscure: games that are not well-known.
  - Panned: games that have middling or poor reviews.

## Reviews

The main outcome of a game is a review. This is not just a text review with a rating but also one
of three signals to put into the Done column.

- Dropped: rate the game and drop it.
- Finished: rate the game once it's finished.
- Replayable: rate the game to mark it can be replayed again.
- Done: rate the game and mark as "done," hedging how the player feels about replaying it.

Reviews cannot be made for a game until it is added to Done for the first time. This is a
philosophical decision; you can review a game you haven't finished if you _drop_ the game, but
otherwise how can a player know how they feel about a game without giving it a fair shake? Reviews
are conceptually tied, in Satchel, to the act of no longer playing a game _or_ knowing the game
will be replayed.

Workaround is to move to Done then back to the category, but this will not get first-class support.
