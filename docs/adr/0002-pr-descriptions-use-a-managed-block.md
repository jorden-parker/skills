# A Draft PR description is jointly owned, via a Managed Block

The `pr-draft` skill writes its generated description between `<!-- pr-draft:start -->` and `<!-- pr-draft:end -->`. The skill now only renders a local preview and leaves GitHub untouched; the markers remain so that any future tool that updates a live PR body rewrites only what is between them.

A PR description is a conversation, not an artifact. Reviewers add context, authors add caveats, and someone pastes a screenshot. An agent that regenerates the whole body destroys all of it on every push, which trains people to stop writing there. The alternative — diffing the live body against a fresh generation and asking before overwriting — needs the agent to correctly judge which human edits are worth preserving, and it will get that wrong often enough to be untrustworthy.

Markers move the guarantee out of the agent's judgement and into the format: text outside the block is not rewritten because the agent never reads past the delimiters.

## Consequences

The markers are visible in the rendered description on GitHub as HTML comments — invisible to readers, but present in the raw body, and they must survive any manual edit. If someone deletes them, the skill appends a fresh block rather than guessing where the old one ended. Changing the marker scheme later leaves already-open PRs inconsistent, which is the main reason to settle it now.
