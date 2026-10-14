---
name: s2t-diff
description: Compare two secrets or configmaps key by key with s2t diff, for files or live cluster resources. Use when asked what differs between environments, namespaces, or manifests.
---

# s2t diff

Run `s2t diff <A> <B>`. Each side is a file or a live name.

- Same resource across namespaces: `s2t diff db-creds db-creds -n staging -B prod`
- Per-side namespace: `-A` and `-B`.
- ConfigMap: `-k configmap` or `cm/<name>`.
- Output: `+` added, `-` removed, `~` changed. Values are hidden.

Add `--show-values` only if the user asks for values. Then do not paste them into commits, tickets, or chat.
