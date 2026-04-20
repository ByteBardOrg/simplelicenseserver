import Prism from "prismjs";
import "prismjs/components/prism-bash";
import "prismjs/components/prism-json";

const languageAliases = {
  shell: "bash",
  sh: "bash",
  zsh: "bash"
};

function resolveLanguage(language = "bash") {
  const normalized = String(language).toLowerCase();
  return languageAliases[normalized] || normalized;
}

export function codeLanguageClass(language = "bash") {
  return `language-${resolveLanguage(language)}`;
}

export function highlightCode(code, language = "bash") {
  const resolvedLanguage = resolveLanguage(language);
  const grammar = Prism.languages[resolvedLanguage];

  if (!grammar) {
    return Prism.util.encode(code);
  }

  return Prism.highlight(code, grammar, resolvedLanguage);
}
