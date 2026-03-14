# blgrep

Search for a domain across all source lists used by [Hagezi's DNS blocklists](https://github.com/hagezi/dns-blocklists).

## Why?

I use [Hagezi Multi Pro](https://github.com/hagezi/dns-blocklists#multi-pro) on my Pi-hole/AdGuard Home setup, and I'm very happy with it.

At work, we build interactive product finders, surveys, and quizzes ([Poltio](https://www.poltio.com)). Our widgets sometimes get flagged as "annoying" by upstream blocklists that Hagezi aggregates from. When that happens, we reach out to the list maintainers to explain that we don't track users or store any activity without consent, and ask them to remove the entry.

As the number of upstream sources grows, manually finding *which* list is blocking us became tedious.

So I wrote this tool to grep all of them at once.

## Usage

```bash
go install github.com/hionay/blgrep@latest

blgrep poltio
```

The tool fetches Hagezi's [sources.md](https://raw.githubusercontent.com/hagezi/dns-blocklists/refs/heads/main/sources.md) at runtime, downloads every listed source concurrently, and prints any lines matching your query.

## Example output

```
Fetching source list from hagezi/dns-blocklists...
Found 247 source URLs. Searching for "poltio"...

LIST: https://easylist-downloads.adblockplus.org/easyprivacy.txt
  LINE 4821: ||poltio.com^$third-party

Found 1 match(es).
```
