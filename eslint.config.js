// Shared ESLint v9 flat config for the Goyama monorepo.
// Each app's `eslint .` walks up to this root file.
import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';

export default tseslint.config(
  {
    ignores: [
      '**/dist/**',
      '**/node_modules/**',
      '**/.tanstack/**',
      '**/routeTree.gen.ts',
      '**/*.tsbuildinfo',
      'pipelines/**',
      'services/**',
      'corpus/**',
      'data/**',
      'docs/**',
      'packages/schema/**',
    ],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'module',
      globals: { ...globals.browser, ...globals.node },
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': [
        'warn',
        { allowConstantExport: true },
      ],
      // Tolerate unused args/vars prefixed with `_`.
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
    },
  },
  {
    // Config/build files run in Node and use CommonJS globals freely.
    files: ['**/*.config.{js,ts,mjs,cjs}', '**/vite.config.*'],
    languageOptions: {
      globals: { ...globals.node },
    },
  },
);
