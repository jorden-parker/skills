# Updating skills

`skills/<name>/` is the source of truth. After updating a skill, run
`./refresh <name>` and then `./refresh --check <name>` before reporting completion.
Use `./refresh` without names to refresh all repo skills.

The command checks existing installations across agent apps, backs up recognised
copies outside skill discovery directories, and links them to this checkout.
It leaves absent installations and unrecognised copies untouched. If it reports
an entry for review, inspect its origin and differences before changing it;
preserve unrelated same-name skills and any local edits in a backup.

Repairing installed copies belonging to this repo is part of the skill update
and needs no separate confirmation. Report unresolved entries and any backup
locations. Advise a fresh session when it has already loaded old instructions.
