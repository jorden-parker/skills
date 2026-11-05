import type { Rule } from "eslint";
import type { Comment } from "estree";

/**
 * Bans comments in source files.
 *
 * Directive comments are exempt on purpose: banning them would take
 * `eslint-disable`, `@ts-expect-error`, and `prettier-ignore` with them, and
 * the escape hatches have to survive the rule.
 */

const DIRECTIVE =
  /^\s*(eslint\b|oxlint\b|prettier-ignore\b|biome-ignore\b|stylelint-\w|@ts-\w|@jsx\b|@license\b|@preserve\b|global\b|globals\b|exported\b|istanbul\b|c8\s|v8\s|webpack\w)/;

const isDirective = (comment: Comment): boolean =>
  DIRECTIVE.test(comment.value);

const isTripleSlash = (comment: Comment): boolean =>
  comment.type === "Line" && comment.value.startsWith("/");

const isShebang = (comment: Comment): boolean =>
  (comment.type as string) === "Shebang" ||
  (comment.range?.[0] === 0 && comment.value.startsWith("!"));

const rule: Rule.RuleModule = {
  meta: {
    type: "suggestion",
    docs: {
      description:
        "Require code to explain itself rather than carry comments that go stale.",
    },
    schema: [],
    messages: {
      noComment:
        "Delete this comment. Name the value, extract the function, or put the reasoning in the commit message.",
    },
  },
  create(context) {
    const source = context.sourceCode;
    return {
      Program() {
        for (const comment of source.getAllComments()) {
          if (isDirective(comment)) continue;
          if (isTripleSlash(comment)) continue;
          if (isShebang(comment)) continue;
          context.report({ loc: comment.loc!, messageId: "noComment" });
        }
      },
    };
  },
};

export default rule;
