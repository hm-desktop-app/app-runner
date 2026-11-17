<script setup lang="ts">
import { computed } from 'vue';
import { icons } from './ui';
import type { IconName } from './ui';

const props = defineProps<{ name: IconName }>();
const icon = computed(() => icons[props.name]);
</script>

<template>
    <!-- The icon set is internal SVG markup, never user content. -->
    <!-- eslint-disable-next-line vue/no-v-html -->
    <svg viewBox="0 0 24 24" v-bind="icon.attributes" v-html="icon.body"></svg>
</template>
