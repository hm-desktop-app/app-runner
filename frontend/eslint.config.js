import js from '@eslint/js';
import globals from 'globals';
import vue from 'eslint-plugin-vue';
import tseslint from 'typescript-eslint';
import prettier from 'eslint-config-prettier';

export default tseslint.config(
    { ignores: ['dist/**', 'bindings/**', 'node_modules/**'] },
    js.configs.recommended,
    ...tseslint.configs.recommended,
    ...vue.configs['flat/recommended'],
    {
        files: ['**/*.{ts,vue,js}'],
        languageOptions: {
            ecmaVersion: 2023,
            sourceType: 'module',
            globals: { ...globals.browser, ...globals.node }
        }
    },
    {
        // The SFC parser has to hand <script lang="ts"> blocks to the TS parser.
        files: ['**/*.vue'],
        languageOptions: { parserOptions: { parser: tseslint.parser } }
    },
    {
        rules: {
            'vue/multi-word-component-names': 'off',
            // Icons and tiles are internal SVG strings, not user content.
            'vue/no-v-html': 'off',
            // localStorage probes deliberately ignore failures.
            'no-empty': ['error', { allowEmptyCatch: true }]
        }
    },
    prettier
);
