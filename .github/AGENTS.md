# Codex Instructions

The canonical repository instructions are stored in
`.github/instructions/*.instructions.md`. The `.xal` language specification
copied from the canonical xaligo repository is stored in
`.github/instructions/07-xal-spec/`.

Before planning, editing, reviewing, or running commands:

1. Read `.github/instructions/index.instructions.md` completely.
2. Read every general and task-specific file selected by that index.
3. Re-evaluate the selected files whenever the task scope changes.

Treat each `applyTo` value as an applicability rule. Treat the copied XAL
specification as the output contract; do not create a converter-specific
dialect or silently relax its constraints.
