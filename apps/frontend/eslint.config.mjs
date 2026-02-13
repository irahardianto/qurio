import pluginVue from "eslint-plugin-vue";
import tseslint from "typescript-eslint";
import globals from "globals";

export default tseslint.config(
  {
    ignores: ["dist/*", "node_modules/*"],
  },

  // Vue recommended
  ...pluginVue.configs["flat/recommended"],

  // TS for .ts/.tsx files
  {
    files: ["**/*.ts", "**/*.tsx"],
    extends: tseslint.configs.recommended,
  },

  // Vue SFCs with TypeScript
  {
    files: ["**/*.vue"],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
        ecmaVersion: "latest",
        sourceType: "module",
      },
      globals: globals.browser,
    },
    plugins: {
      "@typescript-eslint": tseslint.plugin, // ADD THIS
    },
    rules: {
      "vue/multi-word-component-names": "off",
      "@typescript-eslint/no-explicit-any": "off",
      "@typescript-eslint/no-unused-vars": [
        "warn",
        { argsIgnorePattern: "^_" },
      ],
    },
  },

  // Config files (CommonJS)
  {
    files: ["*.config.js", "*.config.cjs"],
    languageOptions: {
      globals: globals.node,
    },
    rules: {
      "@typescript-eslint/no-require-imports": "off",
    },
  },
);
