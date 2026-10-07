import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'

export default tseslint.config(
  {ignores: ['build/**', 'node_modules/**', '.yarn/**']},
  {
    files: ['**/*.{ts,tsx}'],
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    languageOptions: {
      ecmaVersion: 'latest',
      globals: {...globals.browser, ...globals.node},
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      ...reactHooks.configs.flat.recommended.rules,
      '@typescript-eslint/no-unused-vars': ['warn', {argsIgnorePattern: '^_'}],
      'react-refresh/only-export-components': 'off',
    },
  },
  {
    files: ['**/*.js'],
    ignores: ['pwa/sw.template.js', 'public/palette-init.js'],
    extends: [js.configs.recommended],
    languageOptions: {
      ecmaVersion: 'latest',
      globals: {...globals.node},
    },
  },
  // Service Worker 模板运行在 Service Worker 环境中。
  {
    files: ['pwa/sw.template.js'],
    extends: [js.configs.recommended],
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'script',
      globals: {...globals.serviceworker},
    },
  },
  // 首屏配色脚本由 index.html 以普通脚本在浏览器中加载。
  {
    files: ['public/palette-init.js'],
    extends: [js.configs.recommended],
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'script',
      globals: {...globals.browser},
    },
  },
)
