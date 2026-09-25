# Patch PV DB

A mod for *Hatsune Miku Project DIVA Mega Mix Plus* that combines the song
database entries from all your installed mods into one.

## The problem it solves

The game keeps a list of performance videos (called **PVs**) and the songs that
play with them. When more than one mod edits the **same** PV, the game does not
combine them: it keeps the whole entry from the newest-dated mod and throws the
rest away.

So if one mod adds an instrumental version of a song and another mod adds a cover
version of the *same* song, only one of the two shows up in the game.

**Patch PV DB** fixes this by merging the entries piece by piece, so the
contributions from all your mods end up together in the same PV.

## How it runs

This mod is a small DLL that the mod loader starts when the game launches,
**before** the game reads the song database. At that moment it rewrites this
mod's song database file from scratch, using whatever mods you currently have
installed.

There is nothing to configure and nothing to run by hand. Every time you start
the game, the merged database is regenerated automatically, so it always matches
your current set of mods.

## How the merge works

Every mod has a **priority** (set in the game's mod settings). When two mods
disagree about a value, the higher-priority mod wins.

A PV holds two kinds of information:

- **Single values** — the song title, the BPM, which character sings, and so on.
  When two mods set these differently, the higher-priority mod's value is kept.
- **A list of songs** — a PV always has a main song (entry `0`) and can have extra
  songs (entries `1`, `2`, …) that share the same video. These are **combined**,
  not replaced.

### Example

Suppose two mods both edit PV `pv_100`.

**Mod "Alpha" (higher priority)** provides the main song:

```
pv_100.song_name=Starlight
pv_100.another_song.length=1
pv_100.another_song.0.name=Starlight
pv_100.another_song.0.song_file_name=pv_100.ogg
```

**Mod "Beta" (lower priority)** adds an instrumental version:

```
pv_100.another_song.length=2
pv_100.another_song.0.name=Starlight
pv_100.another_song.0.song_file_name=pv_100.ogg
pv_100.another_song.1.name=Starlight (Instrumental)
pv_100.another_song.1.song_file_name=pv_100_b.ogg
```

**Merged result** — Alpha's main song, plus Beta's instrumental:

```
pv_100.song_name=Starlight
pv_100.another_song.length=2
pv_100.another_song.0.name=Starlight
pv_100.another_song.0.song_file_name=pv_100.ogg
pv_100.another_song.1.name=Starlight (Instrumental)
pv_100.another_song.1.song_file_name=pv_100_b.ogg
```

The song count (`length`) is recalculated automatically, so it never has to be
kept in sync by hand.

### Duplicate songs

If two mods add the **same** extra song to the same PV (the same audio file), the
tool recognizes that they are the same song and keeps it only once, using the
higher-priority mod's details. You will not get two identical entries.

### Main song title

The PV's title normally comes from the highest-priority mod that has the PV. If
that mod does not name the main song directly but does set a `song_name`, the main
song inherits that title — so a lower-priority mod cannot silently rename it.

## How patches work

A mod can also include a small **patch** file (`patch_pv_db.txt`) that tweaks
specific entries without resupplying the whole database. For example, to fix a
typo in a song name:

```
pv_100.another_song.1.name=Starlight (Instrumental, Remastered)
```

A few rules:

- **Patches always win** over the merge, even against a higher-priority mod.
- A patch can only **change** entries that already exist — it cannot add a brand
  new PV from nothing.
- Like the merge, a patch that adds an extra song matches it by audio file, so it
  updates the existing entry instead of creating a duplicate.

## Which entries end up in the final file

Only entries that **more than one mod** contributes to, or that a **patch**
touches, are written to the final file. An entry that exactly one mod provides and
no patch changes is left alone — that mod already handles it, so there is no need
to override it.

Every entry that is written gets its date set to the day the game starts. That is
what makes this mod's database the one the game actually uses.

## Build (for developers)

```sh
go build -buildmode=c-shared -tags dll -o patch_pv_db.dll -trimpath -ldflags "-s -w -buildid= "
```

(cgo must be enabled with a MinGW-w64 `gcc` on `PATH`.)
