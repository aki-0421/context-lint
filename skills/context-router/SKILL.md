---
name: context-router
description: Use this skill whenever creating, editing, moving, renaming, deleting, splitting, consolidating, or organizing Markdown documentation. It maintains routing front matter and keeps `index.md` table-of-contents routes current so AI-readable documentation stays discoverable.
---

# Context Router

Use this skill to keep managed documentation routable while doing ordinary documentation work. Do not apply this skill if `context-lint` is not available in the repository.

## Scope Check

Before applying routing rules, decide whether the affected path is managed by `context-lint`.

For existing files, run:

```bash
context-lint managed check <file>
```

For directories or unclear scopes, inspect the managed set:

```bash
context-lint managed list <directory>
context-lint managed tree <path>
```

Apply this skill only to managed Markdown documents. If a file is not managed, skip this skill for that file and do not add front matter, indexes, or links only because of this skill.

For new files, inspect the parent directory first with `managed list` or `managed tree`, create the file when it belongs in a managed documentation area, then run `managed check` after creation to confirm it is managed.

If `context-lint` is unavailable or the repository has no usable config, report that the routing checks could not be applied. Do not install or reconfigure `context-lint` unless the user asks for setup work.

## Front Matter

Every managed Markdown document should have leading YAML front matter containing these fields:

```yaml
---
title: <doc_title>
description: <doc_description>
read_when:
  - <read_conditions>
---
```

Manage only `title`, `description`, and `read_when`.

If front matter already contains other fields, preserve them exactly in meaning. Do not delete, rename, reorder unnecessarily, or edit unrelated fields added by users, other tools, or other skills.

When updating the managed fields:

- Keep `title` human-readable and specific to the document.
- Keep `description` to one concise sentence describing the document's purpose.
- Keep `read_when` as a YAML list of concrete conditions for opening the document.
- Prefer information already present in the document body over invented metadata.
- Preserve the document body unchanged except for edits required by the user's task.

## Index Routing

Maintain `index.md` as the local routing table for managed documentation directories that contain multiple managed documents or managed child documentation directories.

Use `context-lint list <directory>` to identify direct managed files in a directory and read existing front matter for titles, descriptions, and `read_when` values. When child directories may need their own routing, run `context-lint list <child-directory>` on those child directories as well.

Create an `index.md` when a managed directory needs a local table of contents and no index exists. Keep it concise: a short heading, a one-sentence purpose if useful, and a routing table.

Prefer this table shape:

```markdown
| Document | Read when |
| --- | --- |
| [guide.md](guide.md) | Changing the workflow described by this guide. |
| [subdir/index.md](subdir/index.md) | Looking for documents owned by subdir. |
```

For child directories, link to the child directory's `index.md` instead of linking directly to nested files from the parent index.

When `index.md` already exists:

* Update only the table-of-contents or routing section.
* Preserve unrelated sections, prose, examples, decisions, and notes.
* If there is no obvious routing section, add a `## Index` section and leave the rest of the file alone.
* Keep links relative to the `index.md` file.

## Validation

After editing managed documentation, run only `context-lint`:

```bash
context-lint
```
